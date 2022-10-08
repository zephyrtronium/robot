-- "Delete" a message.
-- We delete a message by recording a reason why the message is deleted.
-- Nominally this would be an UPDATE, but we don't necessarily guarantee that
-- the INSERT to record the message happens-before that update.
INSERT INTO Message(id, user, deleted) VALUES (?1, ?2, ?3)
ON CONFLICT DO UPDATE
SET deleted=excluded.deleted;
