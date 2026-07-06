-- Visual asset pipeline: persist generated avatar/clue image URLs alongside
-- the internal generation prompts. The public DTOs expose only *_url and
-- *_status; prompts stay server-side.

ALTER TABLE characters
    ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS avatar_status TEXT NOT NULL DEFAULT 'none';

ALTER TABLE clues
    ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS image_status TEXT NOT NULL DEFAULT 'none';
