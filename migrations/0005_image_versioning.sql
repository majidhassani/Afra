-- Image versioning: a monotonically increasing version per asset so the client
-- can cache-bust regenerated avatars/clue images (append ?ver=N to storage
-- URLs). Data-URL assets ignore it harmlessly; real object-storage URLs use it.

ALTER TABLE characters
    ADD COLUMN IF NOT EXISTS avatar_version INT NOT NULL DEFAULT 0;

ALTER TABLE clues
    ADD COLUMN IF NOT EXISTS image_version INT NOT NULL DEFAULT 0;
