package brain

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/zephyrtronium/robot/tpool"
)

// Speaker produces random messages.
type Speaker interface {
	// Speak generates a full message and appends it to w.
	// The prompt is in reverse order and has entropy reduction applied.
	Speak(ctx context.Context, tag string, prompt []string, w []byte) ([]byte, error)
}

type Prompt struct {
	Terms []string
}

var (
	tokensPool  tpool.Pool[[]string]
	builderPool tpool.Pool[[]byte]
)

// Speak produces a new message from the given prompt.
func Speak(ctx context.Context, s Speaker, tag, prompt string) (string, error) {
	w := builderPool.Get()
	toks := Tokens(tokensPool.Get(), prompt)
	defer func() {
		builderPool.Put(w[:0])
		tokensPool.Put(toks[:0])
	}()
	w = slices.Grow(w, len(prompt))
	for i, t := range toks {
		w = append(w, t...)
		w = append(w, ' ')
		toks[i] = ReduceEntropy(t)
	}
	slices.Reverse(toks)
	w, err := s.Speak(ctx, tag, toks, w)
	if err != nil {
		return "", fmt.Errorf("couldn't speak: %w", err)
	}
	return string(bytes.TrimSpace(w)), nil
}
