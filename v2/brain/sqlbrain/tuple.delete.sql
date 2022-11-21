{{- /* We want to delete exactly one tuple that matches the input. */ -}}
{{- /* Unfortunately, SQLite3's DELETE ... LIMIT 1 is behind a compile-time */ -}}
{{- /* option which is disabled by default. The best solution I've come up */ -}}
{{- /* with is to insert all rows into a temporary table, delete them from */ -}}
{{- /* tuples, and insert all but one back. Any other solution would seem */ -}}
{{- /* to require having a PK, which we want to avoid. */ -}}

ATTACH DATABASE '' AS aux;

CREATE TEMPORARY TABLE aux.hold (
    id INTEGER PRIMARY KEY,
    msg BLOB,
    {{- range $i, $_ := $.Iter}}
    p{{$i}} TEXT,
    {{- end}}
    suffix TEXT
);

INSERT INTO aux.hold (
    msg,
    {{- range $i, $_ := $.Iter}}
    p{{$i}},
    {{- end}}
    suffix
) SELECT
    msg,
    {{- range $i, $_ := $.Iter}}
    p{{$i}},
    {{- end}}
    suffix
FROM main.Tuple
    INNER JOIN Message ON Tuple.msg = Message.id
WHERE tag = :tag
    {{- range $i, $_ := $.Iter}}
    AND p{{$i}} IS :p{{$i}}
    {{- end}}
    AND suffix IS :suffix
    AND LIKELY(deleted IS NULL);

DELETE FROM main.Tuple
WHERE msg IN (SELECT msg FROM aux.hold)
    {{- range $i, $_ := $.Iter}}
    AND p{{$i}} IS :p{{$i}}
    {{- end}}
    AND suffix IS :suffix;

INSERT INTO main.Tuple (
    msg,
    {{- range $i, $_ := $.Iter}}
    p{{$i}},
    {{- end}}
    suffix
) SELECT
    msg,
    {{- range $i, $_ := $.Iter}}
    p{{$i}},
    {{- end}}
    suffix
FROM aux.hold
WHERE id != (SELECT id FROM aux.hold LIMIT 1);

DROP TABLE aux.hold;
DETACH DATABASE aux;
