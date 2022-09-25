
CREATE TABLE Tuple (
    msg BLOB REFERENCES Message(id),
    {{- range $i, $_ := $.Iter }}
    p{{$i}} TEXT,
    {{- end }}
    suffix TEXT
) STRICT;

CREATE INDEX IdxTupleMsg ON Tuple(msg);
CREATE INDEX IdxTuplePN ON Tuple(p{{ $.NM1 }});
