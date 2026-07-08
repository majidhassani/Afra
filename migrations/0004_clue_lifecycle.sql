-- Clue evidence lifecycle: discovered → inspected → confirmed.
-- The `discovered` boolean is kept for back-compat; `status` drives the
-- evidence board and progression (confirmed evidence unlocks the map and
-- feeds completion readiness).

ALTER TABLE clues
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'discovered';

-- Backfill: any already-discovered clue starts at the 'discovered' lifecycle
-- stage; undiscovered clues are irrelevant until found.
UPDATE clues SET status = 'discovered' WHERE discovered AND status = 'discovered';
