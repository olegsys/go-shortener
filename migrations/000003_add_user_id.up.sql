ALTER TABLE short_urls ADD COLUMN user_id TEXT;

UPDATE short_urls SET user_id = '' WHERE user_id IS NULL;

ALTER TABLE short_urls ALTER COLUMN user_id SET NOT NULL;

DROP INDEX IF EXISTS short_urls_original_url_uindex;

CREATE UNIQUE INDEX IF NOT EXISTS short_urls_user_original_uindex ON short_urls (user_id, original_url);
