
-- Message holds message metadata.
CREATE TABLE Message (
    id      BLOB PRIMARY KEY, -- Message UUID.
    user    BLOB NOT NULL, -- Obfuscated user hash.
    tag     TEXT, -- Tag used to learn the message.
    time    INTEGER, -- Message send timestamp. Can be null for migrated data.
    deleted TEXT -- Message delete reason. Null indicates not deleted.
) STRICT;

CREATE UNIQUE INDEX IdxMessageIDs ON Message(id);
CREATE INDEX IdxMessageTags ON Message(tag);
