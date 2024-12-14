WITH m AS (
	SELECT
		tag,
		id,
		time,
		user
	FROM messages
	WHERE tag = :tag AND (time > :startTime OR time = :startTime AND id > :startID) AND deleted IS NULL
	ORDER BY time, id
	LIMIT :n
), k AS (
	SELECT
		m.id,
		m.time,
		m.user,
		knowledge.suffix
	FROM m JOIN knowledge ON m.tag = knowledge.tag AND m.id = knowledge.id
	ORDER BY LENGTH(knowledge.prefix)
)
SELECT
	id,
	time,
	user,
	TRIM(GROUP_CONCAT(suffix, '')) AS msg
FROM k
GROUP BY id, time, user