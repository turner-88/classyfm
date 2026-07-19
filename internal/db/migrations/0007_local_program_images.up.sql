-- Rehost program images locally instead of hotlinking classyfm.co.id (the old
-- station site). Two source images (communitalk-with-yeni-maiasnita,
-- story-of-songs) are already 404 on classyfm.co.id itself and unrecoverable
-- via the Wayback Machine, so those are cleared to NULL to fall back to the
-- existing placeholder icon instead of a permanently broken remote image.
UPDATE programs SET image_url = '/static/img/programs/daylight-time.jpg' WHERE id = 1;
UPDATE programs SET image_url = '/static/img/programs/comfort-time.jpg' WHERE id = 2;
UPDATE programs SET image_url = '/static/img/programs/relax-time.jpg' WHERE id = 3;
UPDATE programs SET image_url = '/static/img/programs/classy-nite-vibes.png' WHERE id = 4;
UPDATE programs SET image_url = NULL WHERE id = 5;
UPDATE programs SET image_url = '/static/img/programs/bebas-pusing.png' WHERE id = 6;
UPDATE programs SET image_url = '/static/img/programs/classy-sport.jpg' WHERE id = 7;
UPDATE programs SET image_url = NULL WHERE id = 8;
UPDATE programs SET image_url = '/static/img/programs/request-time.jpg' WHERE id = 9;
UPDATE programs SET image_url = '/static/img/programs/minangkabau-rancak.jpg' WHERE id = 10;
