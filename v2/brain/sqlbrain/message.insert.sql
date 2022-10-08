-- Insert a new message.
-- Inserting messages happens-before inserting tuple data. Two goroutines can
-- concurrently insert a message if it is deleted immediately upon receive. So,
-- we use upserts (both here and in deletion) to ensure that a delete is always
-- recorded. Here, if another goroutine inserts a record to delete this
-- message, we still update the fields the "delete" won't set.
INSERT INTO Message(id, user, tag, time) VALUES (?, ?, ?, ?)
ON CONFLICT DO UPDATE
SET
    tag=excluded.tag,
    time=excluded.time
RETURNING deleted;
