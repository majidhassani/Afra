-- Playable Mission upgrade: explicit mission stages, report center, and
-- AI-generated scenario board art.
--
-- stages: JSONB array of stage objects (like objectives) — the visible
-- game-level structure (Arrival → ... → Debrief). Backfilled lazily for
-- missions generated before this migration.
--
-- board_art: JSONB map of board_type → {url, status, version} for
-- AI-generated scenario backgrounds. Prompts never leave the server.

ALTER TABLE missions
    ADD COLUMN IF NOT EXISTS stages JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS board_art JSONB NOT NULL DEFAULT '{}';

-- Report center: player-submitted reports validated deterministically by the
-- backend (clue/suspect/progress/incident/final). Accepted reports drive
-- stage completion.
CREATE TABLE IF NOT EXISTS mission_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    linked_clue_ids JSONB NOT NULL DEFAULT '[]',
    suspect_character_id UUID,
    verdict TEXT NOT NULL,
    feedback TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mission_reports_mission
    ON mission_reports (mission_id, created_at DESC);
