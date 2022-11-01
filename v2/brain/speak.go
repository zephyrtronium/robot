package brain

import (
	"context"
	"fmt"
	"strings"
)

// Speaker produces random messages.
type Speaker interface {
	// Order is the length of prompts given to Speak.
	Order() int
	// New finds a prompt to begin a random message. When a message is
	// generated with no prompt, the result from New is passed directly to
	// Speak; it is the speaker's responsibility to ensure it meets
	// requirements with regard to length and matchable content. Only data
	// originally learned with the given tag should be used to generate a
	// prompt.
	New(ctx context.Context, tag string) ([]string, error)
	// Speak generates a full message from the given prompt. The prompt is
	// guaranteed to have length equal to the value returned from Order, unless
	// it is a prompt returned from New. If the number of tokens in the prompt
	// is smaller than Order, the difference is made up by prepending empty
	// strings to the prompt. The speaker should use the given reduce function
	// on all tokens, including those in the prompt, when generating a message.
	// Empty strings at the start and end of the result will be trimmed. Only
	// data originally learned with the given tag should be used to generate a
	// message.
	Speak(ctx context.Context, reduce func(string) string, tag string, prompt []string) ([]string, error)
}

// Speak produces a new message from the given prompt.
func Speak(ctx context.Context, s Speaker, tag, prompt string) (string, error) {
	toks := Tokens(nil, prompt)
	var p []string
	if len(toks) == 0 {
		// No prompt; get one from the speaker instead.
		var err error
		p, err = s.New(ctx, tag)
		if err != nil {
			return "", fmt.Errorf("couldn't get a new prompt: %w", err)
		}
	} else {
		// Make sure the prompt is the right size and has empty strings to
		// make up the difference.
		n := s.Order()
		switch {
		case len(toks) < n:
			p = make([]string, n-len(toks), n)
			p = append(p, toks...)
			toks = toks[:0]
		case len(toks) >= n:
			k := len(toks) - n
			toks, p = toks[:k], toks[k:]
		}
	}
	r, err := s.Speak(ctx, ReduceEntropy, tag, p)
	if err != nil {
		return "", fmt.Errorf("couldn't speak: %w", err)
	}
	return strings.Join(append(toks, trim(r)...), " "), nil
}

// trim removes empty strings from the start and end of r.
func trim(r []string) []string {
	for k := len(r) - 1; k >= 0; k-- {
		if r[k] != "" {
			r = r[:k+1]
			break
		}
	}
	for k, v := range r {
		if v != "" {
			return r[k:]
		}
	}
	return nil
}
