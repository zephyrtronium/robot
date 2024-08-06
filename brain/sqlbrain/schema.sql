PRAGMA journal_mode = WAL;

CREATE TABLE knowledge (
	-- Tag or tenant for the entry.
	tag TEXT NOT NULL,
	-- Message ID, particularly UUID.
	id BLOB NOT NULL,
	-- Prefix stored with entropy-reduced tokens in reverse order,
	-- with each token terminated by \x00 in the string.
	prefix BLOB NOT NULL,
	-- Full-entropy suffix.
	suffix BLOB NOT NULL,
	-- Reason for delete, if any.
	-- Values may include:
	-- 'FORGET', for tuples deleted by content;
	-- 'CLEARMSG', for messages deleted by ID;
	-- 'CLEARCHAT', for messages deleted by userhash;
	-- 'TIME', for messages deleted in a time range;
	-- or NULL, for tuples which have not been deleted.
	-- These values are only for analytics; any non-null value indicates the
	-- tuple should be treated as deleted.
	deleted TEXT
) STRICT;

CREATE TABLE messages (
	-- Tag or tenant for the message.
	tag TEXT NOT NULL,
	-- Message ID, particularly UUID.
	id BLOB NOT NULL,
	-- Message timestamp as nanoseconds from the UNIX epoch.
	-- May be null for messages imported from other sources or for messages
	-- deleted before being fully learned.
	time INTEGER,
	-- Sender userhash.
	-- May be null for messages imported from other sources or for messages
	-- deleted before being fully learned.
	user BLOB,
	-- Reason for delete, if any.
	-- Same meaning as in knowledge, except that the value 'FORGET' will never
	-- appear (since that is specifically for operating on tuples).
	-- Denormalized here to allow soft deletes of messages before they are
	-- actually learned.
	deleted TEXT,

	PRIMARY KEY(tag, id)
) STRICT;

CREATE INDEX ids ON knowledge (tag, id);
CREATE INDEX prefixes ON knowledge (tag, prefix);
CREATE INDEX times ON messages (tag, time);
CREATE INDEX users ON messages (user);
