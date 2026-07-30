-- Replays the pre-0023 programs.host free text (preserved in db-data.sql) into the
-- broadcaster_id FK added by 0024. Only the two programs whose host text resolves to an
-- existing broadcaster are filled; "Hari" (classy-sport, story-of-songs, request-time) has
-- no broadcasters row and is left NULL on purpose. Where the host listed several people,
-- the first name wins - the rest of the team stays visible via per-slot assignments.
--
-- Matched on slug rather than the ids from the dump (both slug columns are UNIQUE) so this
-- stays correct if prod ids ever diverge from local. The broadcaster_id IS NULL guard makes
-- a re-run a no-op and never clobbers a default set by hand in the admin panel.
UPDATE programs p
JOIN broadcasters b ON b.slug = 'yeni-maiasnita'
SET p.broadcaster_id = b.id
WHERE p.slug = 'communitalk-with-yeni-maiasnita' AND p.broadcaster_id IS NULL;

UPDATE programs p
JOIN broadcasters b ON b.slug = 'andahayani'
SET p.broadcaster_id = b.id
WHERE p.slug = 'bebas-pusing' AND p.broadcaster_id IS NULL;
