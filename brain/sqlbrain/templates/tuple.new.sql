-- Select all start-of-message terms with a given tag.
-- The template input requires $.NM1 to be (order-1).
SELECT suffix
FROM MessageTuple
WHERE tag = ?
    AND p{{$.NM1}} = ''
