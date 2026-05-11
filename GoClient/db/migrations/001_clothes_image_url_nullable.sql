-- Existing deployments only: migrate clothes.image_url from NOT NULL UNIQUE to nullable + partial unique index.
-- Fresh installs use GoClient/db/init.sql directly.

ALTER TABLE clothes DROP CONSTRAINT IF EXISTS clothes_image_url_key;

ALTER TABLE clothes ALTER COLUMN image_url DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_clothes_image_url_unique ON clothes (image_url) WHERE image_url IS NOT NULL;
