-- Clears only what the up migration set: the join on the assigned broadcaster means a
-- default changed by hand since is left alone.
UPDATE programs p
JOIN broadcasters b ON b.id = p.broadcaster_id AND b.slug = 'yeni-maiasnita'
SET p.broadcaster_id = NULL
WHERE p.slug = 'communitalk-with-yeni-maiasnita';

UPDATE programs p
JOIN broadcasters b ON b.id = p.broadcaster_id AND b.slug = 'andahayani'
SET p.broadcaster_id = NULL
WHERE p.slug = 'bebas-pusing';
