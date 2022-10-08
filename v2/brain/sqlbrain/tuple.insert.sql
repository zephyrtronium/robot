-- Insert a sequence of tuples.
WITH MsgID AS (
    VALUES (?)
)
INSERT INTO Tuple(msg, {{range $i, $_ := $.Iter}}p{{$i}}, {{end}}suffix)
VALUES
    ((SELECT * FROM MsgID LIMIT 1), {{range $.Iter}}?, {{end}}?)
    {{- range slice $.Tuples 1}},
    ((SELECT * FROM MsgID LIMIT 1), {{range $.Iter}}?, {{end}}?)
    {{- end}};
