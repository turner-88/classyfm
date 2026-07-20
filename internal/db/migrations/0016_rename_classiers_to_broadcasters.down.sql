UPDATE broadcasters
SET photo_url = REPLACE(photo_url, '/static/img/broadcasters/', '/static/img/classiers/')
WHERE photo_url LIKE '/static/img/broadcasters/%';

RENAME TABLE broadcasters TO classiers;
