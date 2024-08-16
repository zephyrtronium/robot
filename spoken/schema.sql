CREATE TABLE spoken (
	-- Tag or tenant for the entry.
	-- Note this is the speaking tag, not the tenant's learning tag.
	tag TEXT NOT NULL,
	-- Message text, without emote or effect.
	msg TEXT NOT NULL,
	-- Trace of message IDs used to generate the message,
	-- stored as a JSONB array.
	trace BLOB NOT NULL,
	-- Message timestamp as nanoseconds from the UNIX epoch.
	time INTEGER NOT NULL,
	-- Various metadata about the message, stored as a JSONB object.
	-- May include:
	-- 	"emote": Emote appended to the message.
	-- 	"effect": Name of the effect applied to the message.
	-- 	"cost": Time in nanoseconds spent generating the message.
	meta BLOB NOT NULL
) STRICT;

-- Covering index for lookup.
CREATE INDEX traces ON spoken (tag, msg, time DESC, trace);
