package brain

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/zephyrtronium/robot/tpool"
)

// Speaker produces random messages.
type Speaker interface {
	// Speak generates a full message and appends it to w.
	// The prompt is in reverse order and has entropy reduction applied.
	Speak(ctx context.Context, tag string, prompt []string, w *Builder) error
}

var (
	tokensPool  tpool.Pool[[]string]
	builderPool = tpool.Pool[*Builder]{New: func() any { return new(Builder) }}
)

// Speak produces a new message and the trace of messages used to form it
// from the given prompt.
// If the speaker does not produce any terms, the result is the empty string
// regardless of the prompt, with no error.
func Speak(ctx context.Context, s Speaker, tag, prompt string) (string, []string, error) {
	w := builderPool.Get()
	toks := Tokens(tokensPool.Get(), prompt)
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
