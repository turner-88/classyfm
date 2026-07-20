RENAME TABLE classiers TO broadcasters;

UPDATE broadcasters
SET photo_url = REPLACE(photo_url, '/static/img/classiers/', '/static/img/broadcasters/')
WHERE photo_url LIKE '/static/img/classiers/%';
