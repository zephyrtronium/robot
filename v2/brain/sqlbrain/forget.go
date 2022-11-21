package sqlbrain

import (
	"context"
	_ "embed"
	"time"

	"github.com/google/uuid"
	"github.com/zephyrtronium/robot/v2/brain"
)

// Forget deletes tuples from the database. To ensure consistency and accuracy,
// the ForgetMessage, ForgetDuring, and ForgetUserSince methods should be
// preferred where possible.
func (br *Brain) Forget(ctx context.Context, tag string, tuples []brain.Tuple) error {
	panic("unimplemented")
}

// ForgetMessage removes tuples associated with a message from the database.
// The delete reason is set to "CLEARMSG".
func (br *Brain) ForgetMessage(ctx context.Context, msg uuid.UUID) error {
	panic("unimplemented")
}

// ForgetDuring removes tuples associated with messages learned in the given
// time span. The delete reason is set to "TIMED".
func (br *Brain) ForgetDuring(ctx context.Context, tag string, since, before time.Time) error {
	panic("unimplemented")
}

// ForgetUserSince removes tuples learned from the given user hash since a
// given time. The delete reason is set to "CLEARCHAT".
func (br *Brain) ForgetUserSince(ctx context.Context, user [32]byte, since time.Time) error {
	panic("unimplemented")
}

//go:embed tuple.delete.sql
var deleteTuple string
