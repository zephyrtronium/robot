-- Select a start-of-message term with a given tag.
-- The template input requires $.NM1 to be (order-1).
WITH InitialSet AS (
    SELECT
        RANDOM() AS ordinal,
        suffix
    FROM MessageTuple
    WHERE tag = ?
        AND p{{$.NM1}} IS NULL
)
SELECT suffix
FROM InitialSet
ORDER BY ordinal
LIMIT 1
