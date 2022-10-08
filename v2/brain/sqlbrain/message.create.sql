-- Define both Message and Tuple tables. Tuple depends on Message for an FK,
-- and putting the CREATE TABLEs for both in one file ensures they serialize.

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

-- Tuple holds actual Markov chain tuples.
CREATE TABLE Tuple (
    msg BLOB REFERENCES Message(id),
    {{- range $i, $_ := $.Iter }}
    p{{$i}} TEXT,
    {{- end }}
    suffix TEXT
) STRICT;

CREATE INDEX IdxTupleMsg ON Tuple(msg);
CREATE INDEX IdxTuplePN ON Tuple(p{{ $.NM1 }});
