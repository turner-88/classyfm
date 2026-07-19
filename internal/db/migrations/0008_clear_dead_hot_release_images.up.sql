-- All 10 hot_release news items' thumbnails hotlink classyfm.co.id/file/post/thumbs/*,
-- and every one of them is already 404 at the source (confirmed by hand) with
-- no Wayback Machine copy, so clear them to NULL to fall back to the existing
-- placeholder icon instead of a permanently broken <img>.
UPDATE news_items SET image_url = NULL WHERE id = 771;
UPDATE news_items SET image_url = NULL WHERE id = 772;
UPDATE news_items SET image_url = NULL WHERE id = 773;
UPDATE news_items SET image_url = NULL WHERE id = 774;
UPDATE news_items SET image_url = NULL WHERE id = 775;
UPDATE news_items SET image_url = NULL WHERE id = 776;
UPDATE news_items SET image_url = NULL WHERE id = 777;
UPDATE news_items SET image_url = NULL WHERE id = 778;
UPDATE news_items SET image_url = NULL WHERE id = 779;
UPDATE news_items SET image_url = NULL WHERE id = 780;
