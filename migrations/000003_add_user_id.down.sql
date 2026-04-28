DROP INDEX IF EXISTS short_urls_user_original_uindex;

CREATE UNIQUE INDEX IF NOT EXISTS short_urls_original_url_uindex
    ON short_urls (original_url);

ALTER TABLE short_urls
    DROP COLUMN IF EXISTS user_id;
