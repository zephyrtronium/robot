-- Select a single suffix.
-- For performance reasons, we always require the final prefix element to match
-- exactly. For the remaining elements, we want a higher probability to select
-- a given tuple as more of its terms match, weighted toward later elements.
-- The template inputs include $.Fibonacci, a slice of (order-1) consecutive
-- Fibonacci numbers; $.NM1, which is (order-1); and $.MinScore, which is the
-- minimum sum of the Fibonacci numbers corresponding to matching terms for a
-- tuple to be considered.
-- We use one Fibonacci number less than you might expect, because the final
-- prefix element must be an exact match anyway. Using consecutive Fibonacci
-- numbers expresses the goal that we can drop one term if the two previous
-- ones match.
-- The SQL statement inputs are :tag and :p0, :p1, ... as named parameters.
WITH InitialSet AS (
    SELECT {{- range $i, $_ := $.Fibonacci}}
        p{{$i}},
        {{- end}}
        suffix
    FROM MessageTuple
    WHERE tag = :tag
        AND p{{$.NM1}} IS :p{{$.NM1}}
), Scored AS (
    SELECT 0 {{- range $i, $w := $.Fibonacci -}}
        + {{$w}}*(p{{$i}} IS :p{{$i}})
        {{- end}} AS score,
        suffix
    FROM InitialSet
), Thresholded AS (
    SELECT suffix
    FROM Scored
    WHERE score >= {{$.MinScore}}
)
SELECT suffix
FROM Thresholded
