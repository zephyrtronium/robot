{{- /* We want to delete exactly one tuple that matches the input. */ -}}
{{- /* Unfortunately, SQLite3's DELETE ... LIMIT 1 is behind a compile-time */ -}}
{{- /* option which is disabled by default. The best solution I've come up */ -}}
{{- /* with is to insert all rows into a temporary table, delete them from */ -}}
{{- /* tuples, and insert all but one back. Any other solution would seem */ -}}
{{- /* to require having a PK, which we want to avoid. */ -}}

{{- /* We also define this as a sequence of templates each containing one */ -}}
{{- /* statement, because SQLite3 doesn't support preparing multiple */ -}}
{{- /* statements at a time, and the SQL parameters in each differ. */ -}}

{{- define "tuple.delete.0" -}}
CREATE TEMPORARY TABLE delete_hold (
    id INTEGER PRIMARY KEY,
    msg BLOB,
    {{- range $i, $_ := $.Iter}}
    p{{$i}} TEXT,
    {{- end}}
    suffix TEXT
);
{{- end -}}

{{- define "tuple.delete.1" -}}
INSERT INTO delete_hold (
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
{{- end -}}

{{- define "tuple.delete.2" -}}
DELETE FROM main.Tuple
WHERE msg IN (SELECT msg FROM delete_hold)
    {{- range $i, $_ := $.Iter}}
    AND p{{$i}} IS :p{{$i}}
    {{- end}}
    AND suffix IS :suffix;
{{- end -}}

{{- define "tuple.delete.3" -}}
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
FROM delete_hold
WHERE id != (SELECT id FROM delete_hold LIMIT 1);
{{- end -}}

{{- define "tuple.delete.4" -}}
DROP TABLE delete_hold;
{{- end -}}
