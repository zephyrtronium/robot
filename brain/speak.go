package brain

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/zephyrtronium/robot/deque"
	"github.com/zephyrtronium/robot/tpool"
)

var (
	tokensPool    tpool.Pool[[]string]
	prependerPool tpool.Pool[deque.Deque[string]]
	bytesPool     tpool.Pool[[]byte]
	builderPool   = tpool.Pool[*Builder]{New: func() any { return new(Builder) }}
)

// Speak produces a new message and the trace of messages used to form it
// from the given prompt.
// If the brain does not produce any terms, the result is the empty string
// regardless of the prompt, with no error.
func Speak(ctx context.Context, s Interface, tag, prompt string) (string, []string, error) {
	w := builderPool.Get()
	toks := tokens(tokensPool.Get(), prompt)
	defer func() {
		w.Reset()
		builderPool.Put(w)
		tokensPool.Put(toks[:0])
	}()
	w.grow(len(prompt) + 1)
	for i, t := range toks {
		w.prompt(t)
		toks[i] = ReduceEntropy(t)
	}
	slices.Reverse(toks)
	err := s.Speak(ctx, tag, toks, w)
	if err != nil {
		return "", nil, fmt.Errorf("couldn't speak: %w", err)
	}
	if len(w.Trace()) == 0 {
		return "", nil, nil
	}
	return strings.TrimSpace(w.String()), slices.Clone(w.Trace()), nil
}

// Think produces a new message and the trace of message IDs used to form it
// from the given prompt.
// If the brain does not produce any terms, the result is the empty string
// regardless of the prompt, with no error.
func Think(ctx context.Context, s Interface, tag, prompt string) (string, []string, error) {
	w := bytesPool.Get()
	toks := tokens(tokensPool.Get(), prompt)
	for i, t := range toks {
		w = append(w, t...)
		toks[i] = ReduceEntropy(t)
	}
	slices.Reverse(toks)
	search := prependerPool.Get().Prepend(toks...)
	defer func() {
		bytesPool.Put(w[:0])
		tokensPool.Put(toks[:0])
		prependerPool.Put(search.Reset())
	}()

	var ids []string
	// We handle the first search specially.
	// TODO(zeph): skip terminating choices
	id, tok, l, err := next(ctx, s, tag, search.Slice())
	if len(tok) == 0 {
		return "", nil, err
	}
	k, ok := slices.BinarySearch(ids, id)
	if !ok {
		ids = slices.Insert(ids, k, id)
	}
	w = append(w, tok...)
	search = search.DropEnd(search.Len() - l - 1).Prepend(ReduceEntropy(tok))
	for range 1024 {
		id, tok, l, err := next(ctx, s, tag, search.Slice())
		if len(tok) == 0 {
			// This could mean the message is done, there was no match for
			// the prefix, or an error occurred.
			return string(bytes.TrimSpace(w)), ids, err
		}
		k, ok := slices.BinarySearch(ids, id)
		if !ok {
			ids = slices.Insert(ids, k, id)
		}
		w = append(w, tok...)
		search = search.DropEnd(search.Len() - l - 1).Prepend(ReduceEntropy(tok))
	}
	return string(bytes.TrimSpace(w)), ids, nil
}

// next finds a single next term from a brain given a prompt.
func next(ctx context.Context, s Interface, tag string, prompt []string) (id, tok string, l int, err error) {
	wid := make([]byte, 0, 64)
	wtok := make([]byte, 0, 64)
	var skip Skip
	var n uint64
	for {
		var seen uint64
		n, seen, err = term(ctx, s, tag, prompt, &wid, &wtok, &skip, n)
		if err != nil {
			return "", "", 0, err
		}
		// Try to lose context.
		// We want to do so when we have context but see zero options at all,
		// we have a long context and almost no options,
		// or at random with even a short context.
		//
		// Note that in the third case we use a 1/2 chance; it seems high, but
		// n.b. the caller will recover the last token that we discard.
		// This logic is also what makes the first case necessary: we can
		// randomly drop context, find a result that is impossible otherwise,
		// and then suddenly halt generation mid-sentence on the next token
		// because we don't shorten back to where we were.
		if len(prompt) > 1 && seen == 0 || len(prompt) > 4 && seen <= 2 || len(prompt) > 2 && rand.Uint32()&1 == 0 {
			prompt = prompt[:len(prompt)-1]
			continue
		}
		// Note that this also handles the case where there were no results.
		return string(wid), string(wtok), len(prompt), nil
	}
}

// term gets the thought for a single prompt and skip sequence with a starting
// skip length, returning the new skip length and the total number skipped.
func term(ctx context.Context, s Interface, tag string, prompt []string, wid, wtok *[]byte, skip *Skip, n uint64) (uint64, uint64, error) {
	var seen uint64
	for f := range s.Think(ctx, tag, prompt) {
		seen++
		if n > 0 {
			n--
			continue
		}
		if err := f(wid, wtok); err != nil {
			return n, seen, fmt.Errorf("couldn't think: %w", err)
		}
		n = skip.N(rand.Uint64(), rand.Uint64())
	}
	return n, seen, nil
}
