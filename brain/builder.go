package brain

import "slices"

// Builder builds a spoken message along with its message trace.
type Builder struct {
	w  []byte
	id []string
}

// Append adds a term to the builder.
func (b *Builder) Append(id string, term []byte) {
	b.w = append(b.w, term...)
	k, ok := slices.BinarySearch(b.id, id)
	if !ok {
		b.id = slices.Insert(b.id, k, id)
	}
}

// prompt adds a term without an ID.
func (b *Builder) prompt(term string) {
	b.w = append(b.w, term...)
}

// grow reserves sufficient space to append at least n bytes without reallocating.
func (b *Builder) grow(n int) {
	if cap(b.w)-len(b.w) >= n {
		return
	}
	t := make([]byte, len(b.w), len(b.w)+n)
	copy(t, b.w)
	b.w = t
}

// String returns the built message.
func (b *Builder) String() string {
	return string(b.w)
}

// Trace returns a direct reference to the message trace.
func (b *Builder) Trace() []string {
	return b.id
}

// Reset restores the builder to an empty state.
func (b *Builder) Reset() {
	b.w = b.w[:0]
	clear(b.id) // allow held strings to release
	b.id = b.id[:0]
}
