UPDATE knowledge
SET deleted = 'FORGET'
WHERE tag = :tag
	AND id = (
		SELECT id
		FROM knowledge
		WHERE tag = :tag
			AND prefix = :prefix
			AND suffix = :suffix
			AND LIKELY(deleted IS NULL)
		LIMIT 1
	)
	AND prefix = :prefix
	-- We don't need to match suffix because every prefix of a given message is unique.
