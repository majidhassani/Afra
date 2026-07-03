-- AgentVerse Mission Engine upgrade.
-- New mission-generalized schema + wallet + data conversion from the
-- detective/case schema. Legacy tables are retained during the transition.
-- Note: the mission clock column is named mission_time (not current_time)
-- because CURRENT_TIME is reserved in SQL.

-- ---------------------------------------------------------------------------
-- Missions
-- ---------------------------------------------------------------------------

CREATE TABLE missions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'generating',
    difficulty TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    briefing TEXT NOT NULL DEFAULT '',
    objectives JSONB NOT NULL DEFAULT '[]',
    public_state JSONB NOT NULL DEFAULT '{}',
    result JSONB,
    center_lat DOUBLE PRECISION NOT NULL DEFAULT 0,
    center_lng DOUBLE PRECISION NOT NULL DEFAULT 0,
    map_zoom INT NOT NULL DEFAULT 12,
    mission_time TIMESTAMPTZ NOT NULL DEFAULT '2026-01-01 08:00:00+00',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX idx_missions_user_id ON missions(user_id);
CREATE INDEX idx_missions_status ON missions(status);

CREATE TABLE world_bibles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL UNIQUE REFERENCES missions(id) ON DELETE CASCADE,
    truth JSONB NOT NULL DEFAULT '{}',
    hidden_state JSONB NOT NULL DEFAULT '{}',
    character_secrets JSONB NOT NULL DEFAULT '{}',
    clue_truth JSONB NOT NULL DEFAULT '{}',
    map_truth JSONB NOT NULL DEFAULT '{}',
    timeline_truth JSONB NOT NULL DEFAULT '[]',
    failure_rules JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE map_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT '',
    latitude DOUBLE PRECISION NOT NULL DEFAULT 0,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'hidden',
    risk_level INT NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    visual_prompt TEXT NOT NULL DEFAULT '',
    available_actions JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_map_locations_mission_id ON map_locations(mission_id);

CREATE TABLE characters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'field',
    age INT NOT NULL DEFAULT 0,
    public_profile TEXT NOT NULL DEFAULT '',
    personality JSONB NOT NULL DEFAULT '{}',
    current_location_id UUID REFERENCES map_locations(id) ON DELETE SET NULL,
    trust_level INT NOT NULL DEFAULT 50,
    stress_level INT NOT NULL DEFAULT 10,
    mood TEXT NOT NULL DEFAULT 'neutral',
    dialogue_style TEXT NOT NULL DEFAULT '',
    avatar_prompt TEXT NOT NULL DEFAULT '',
    thumbnail_prompt TEXT NOT NULL DEFAULT '',
    visual_style_tags JSONB NOT NULL DEFAULT '[]',
    private_state JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_characters_mission_id ON characters(mission_id);

CREATE TABLE clues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    location_id UUID REFERENCES map_locations(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    type TEXT NOT NULL,
    short_description TEXT NOT NULL DEFAULT '',
    detailed_description TEXT NOT NULL DEFAULT '',
    visual_description TEXT NOT NULL DEFAULT '',
    avatar_or_thumbnail_prompt TEXT NOT NULL DEFAULT '',
    discovered BOOLEAN NOT NULL DEFAULT false,
    reliability INT NOT NULL DEFAULT 50,
    importance TEXT NOT NULL DEFAULT 'medium',
    related_character_ids JSONB NOT NULL DEFAULT '[]',
    public_data JSONB NOT NULL DEFAULT '{}',
    internal_truth JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_clues_mission_id ON clues(mission_id);
CREATE INDEX idx_clues_location_id ON clues(location_id);
CREATE INDEX idx_clues_discovered ON clues(discovered);

CREATE TABLE interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    character_id UUID REFERENCES characters(id) ON DELETE CASCADE,
    interaction_type TEXT NOT NULL DEFAULT 'character_chat',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_interactions_mission_id ON interactions(mission_id);
CREATE UNIQUE INDEX idx_interactions_unique_thread
    ON interactions(mission_id, interaction_type, COALESCE(character_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE TABLE interaction_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id UUID NOT NULL REFERENCES interactions(id) ON DELETE CASCADE,
    sender TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_interaction_messages_interaction_id ON interaction_messages(interaction_id);

CREATE TABLE mission_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_mission_events_mission_id ON mission_events(mission_id);

CREATE TABLE journal_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mission_id UUID NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_journal_notes_mission_id ON journal_notes(mission_id);

-- ---------------------------------------------------------------------------
-- Player profiles
-- ---------------------------------------------------------------------------

CREATE TABLE player_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL DEFAULT '',
    rank TEXT NOT NULL DEFAULT 'Recruit',
    level INT NOT NULL DEFAULT 1,
    xp INT NOT NULL DEFAULT 0,
    total_missions INT NOT NULL DEFAULT 0,
    completed_missions INT NOT NULL DEFAULT 0,
    failed_missions INT NOT NULL DEFAULT 0,
    success_rate REAL NOT NULL DEFAULT 0,
    favorite_mission_type TEXT NOT NULL DEFAULT '',
    total_clues_found INT NOT NULL DEFAULT 0,
    total_ai_interactions INT NOT NULL DEFAULT 0,
    total_locations_visited INT NOT NULL DEFAULT 0,
    total_coins_spent INT NOT NULL DEFAULT 0,
    total_coins_earned INT NOT NULL DEFAULT 0,
    badges JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Wallet
-- ---------------------------------------------------------------------------

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance INT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    reserved_balance INT NOT NULL DEFAULT 0 CHECK (reserved_balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mission_id UUID REFERENCES missions(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    amount INT NOT NULL,
    balance_after INT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_wallet_transactions_user_id ON wallet_transactions(user_id);
CREATE INDEX idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);

CREATE TABLE wallet_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mission_id UUID,
    action_type TEXT NOT NULL,
    amount INT NOT NULL CHECK (amount >= 0),
    status TEXT NOT NULL DEFAULT 'active', -- active | settled | released
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_wallet_reservations_wallet_id ON wallet_reservations(wallet_id);
CREATE INDEX idx_wallet_reservations_status ON wallet_reservations(status);

CREATE TABLE pricing_rules (
    action_type TEXT PRIMARY KEY,
    coins INT NOT NULL CHECK (coins >= 0),
    active BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE llm_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mission_id UUID,
    agent_name TEXT NOT NULL DEFAULT '',
    action_type TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    estimated_cost NUMERIC NOT NULL DEFAULT 0,
    coins_charged INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_llm_usage_logs_user_id ON llm_usage_logs(user_id);
CREATE INDEX idx_llm_usage_logs_mission_id ON llm_usage_logs(mission_id);

CREATE TABLE rewarded_ads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    coins INT NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_rewarded_ads_user_claimed ON rewarded_ads(user_id, claimed_at);

CREATE TABLE purchase_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    product_id TEXT NOT NULL,
    receipt_hash TEXT NOT NULL UNIQUE,
    coins INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending', -- pending | verified | rejected
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_purchase_receipts_user_id ON purchase_receipts(user_id);

-- ---------------------------------------------------------------------------
-- Agent runs: attach missions
-- ---------------------------------------------------------------------------

ALTER TABLE agent_runs ADD COLUMN mission_id UUID;
CREATE INDEX idx_agent_runs_mission_id ON agent_runs(mission_id);

-- ---------------------------------------------------------------------------
-- Pricing seed (server-side prices; clients are never trusted)
-- ---------------------------------------------------------------------------

INSERT INTO pricing_rules (action_type, coins) VALUES
    ('mission_start', 150),
    ('character_chat', 3),
    ('ai_guidance', 2),
    ('clue_explain', 2),
    ('clue_inspect', 3),
    ('location_search', 5),
    ('location_ask', 2),
    ('advance_time', 4),
    ('final_judgment', 20);

-- ---------------------------------------------------------------------------
-- Data conversion from the detective schema (ownership and IDs preserved)
-- ---------------------------------------------------------------------------

-- Cases become detective-type missions.
INSERT INTO missions (id, user_id, type, title, status, difficulty, summary, briefing, created_at, updated_at, completed_at)
SELECT id, user_id, 'detective', title,
       CASE status
           WHEN 'open' THEN 'active'
           WHEN 'solved' THEN 'completed'
           ELSE status
       END,
       difficulty, summary, summary, created_at, updated_at, solved_at
FROM cases;

-- Case bibles become world bibles.
INSERT INTO world_bibles (mission_id, truth, hidden_state, character_secrets, clue_truth, timeline_truth, created_at, updated_at)
SELECT case_id,
       truth,
       jsonb_build_object(
           'hidden_facts', hidden_facts,
           'false_leads', false_leads,
           'hidden_antagonist_id', culprit_id::text,
           'motive', motive
       ),
       suspect_secrets,
       evidence_truth,
       real_timeline,
       created_at, updated_at
FROM case_bibles;

-- Locations become map locations.
INSERT INTO map_locations (id, mission_id, name, type, latitude, longitude, status, risk_level, description, available_actions, created_at, updated_at)
SELECT id, case_id, name, type, latitude, longitude,
       CASE WHEN discovered THEN 'discovered' ELSE 'hidden' END,
       20, description,
       '["inspect_area","talk_to_character","ask_ai","view_clues"]'::jsonb,
       created_at, updated_at
FROM locations;

-- Suspects become characters.
INSERT INTO characters (id, mission_id, name, role, category, age, public_profile, personality,
                        trust_level, stress_level, dialogue_style, private_state, created_at, updated_at)
SELECT id, case_id, name, job, 'field', age,
       public_profile::text, personality,
       trust_level, stress_level, '',
       jsonb_build_object(
           'is_antagonist', is_culprit,
           'hidden_knowledge', secrets,
           'lie_profile', lie_profile,
           'private_memory', private_memory,
           'known_facts', known_facts,
           'relation_to_victim', relation_to_victim
       ),
       created_at, updated_at
FROM suspects;

-- Evidence becomes clues.
INSERT INTO clues (id, mission_id, location_id, title, type, short_description, discovered,
                   reliability, related_character_ids, public_data, internal_truth, created_at, updated_at)
SELECT e.id, e.case_id,
       CASE WHEN EXISTS (SELECT 1 FROM map_locations ml WHERE ml.id = (e.related_location_ids->>0)::uuid)
            THEN (e.related_location_ids->>0)::uuid END,
       e.title, e.type, e.description, e.discovered,
       e.reliability, e.related_suspect_ids, e.public_data, e.internal_truth,
       e.created_at, e.updated_at
FROM evidence e;

-- Conversations become interactions.
INSERT INTO interactions (id, mission_id, user_id, character_id, interaction_type, created_at, updated_at)
SELECT c.id, c.case_id, cs.user_id, c.suspect_id, 'character_chat', c.started_at, c.last_message_at
FROM conversations c
JOIN cases cs ON cs.id = c.case_id;

INSERT INTO interaction_messages (id, interaction_id, sender, content, metadata, created_at)
SELECT id, conversation_id, role, content,
       CASE WHEN emotion <> '' THEN jsonb_build_object('emotion', emotion) ELSE '{}'::jsonb END,
       created_at
FROM conversation_messages;

-- Player notes become journal notes.
INSERT INTO journal_notes (id, mission_id, title, content, created_at, updated_at)
SELECT id, case_id, title, content, created_at, updated_at
FROM player_notes;

-- Case events become mission events.
INSERT INTO mission_events (id, mission_id, type, payload, created_at)
SELECT id, case_id, type, payload, created_at
FROM case_events;

-- Detective profiles become player profiles.
INSERT INTO player_profiles (user_id, display_name, rank, level, xp, total_missions,
                             completed_missions, failed_missions, success_rate,
                             favorite_mission_type, created_at, updated_at)
SELECT dp.user_id, u.display_name, dp.rank, GREATEST(1, dp.xp / 300 + 1), dp.xp,
       dp.total_cases, dp.solved_cases, dp.failed_cases, dp.accuracy_rate,
       CASE WHEN dp.total_cases > 0 THEN 'detective' ELSE '' END,
       dp.created_at, dp.updated_at
FROM detective_profiles dp
JOIN users u ON u.id = dp.user_id;

-- Backfill agent_runs.mission_id (case IDs were preserved as mission IDs).
UPDATE agent_runs SET mission_id = case_id WHERE case_id IS NOT NULL;

-- Every existing user gets a wallet with a starting balance.
INSERT INTO wallets (user_id, balance)
SELECT id, 500 FROM users;

INSERT INTO wallet_transactions (wallet_id, user_id, type, amount, balance_after, metadata)
SELECT w.id, w.user_id, 'admin_adjustment', 500, 500, '{"reason":"agentverse_migration_starting_balance"}'::jsonb
FROM wallets w;
