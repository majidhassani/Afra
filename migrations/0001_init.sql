-- CaseMind initial schema.

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE detective_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    rank TEXT NOT NULL DEFAULT 'Rookie',
    xp INT NOT NULL DEFAULT 0,
    solved_cases INT NOT NULL DEFAULT 0,
    failed_cases INT NOT NULL DEFAULT 0,
    total_cases INT NOT NULL DEFAULT 0,
    accuracy_rate REAL NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL,
    difficulty TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'generating',
    summary TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    solved_at TIMESTAMPTZ
);
CREATE INDEX idx_cases_user_id ON cases(user_id);
CREATE INDEX idx_cases_status ON cases(status);

CREATE TABLE case_bibles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL UNIQUE REFERENCES cases(id) ON DELETE CASCADE,
    culprit_id UUID,
    motive TEXT NOT NULL DEFAULT '',
    truth JSONB NOT NULL DEFAULT '{}',
    real_timeline JSONB NOT NULL DEFAULT '[]',
    hidden_facts JSONB NOT NULL DEFAULT '[]',
    false_leads JSONB NOT NULL DEFAULT '[]',
    evidence_truth JSONB NOT NULL DEFAULT '{}',
    suspect_secrets JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE suspects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    age INT NOT NULL DEFAULT 0,
    job TEXT NOT NULL DEFAULT '',
    relation_to_victim TEXT NOT NULL DEFAULT '',
    public_profile JSONB NOT NULL DEFAULT '{}',
    personality JSONB NOT NULL DEFAULT '{}',
    known_facts JSONB NOT NULL DEFAULT '[]',
    stress_level INT NOT NULL DEFAULT 10,
    trust_level INT NOT NULL DEFAULT 50,
    interrogation_count INT NOT NULL DEFAULT 0,
    is_culprit BOOLEAN NOT NULL DEFAULT false,
    secrets JSONB NOT NULL DEFAULT '[]',
    lie_profile JSONB NOT NULL DEFAULT '{}',
    private_memory JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_suspects_case_id ON suspects(case_id);

CREATE TABLE evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    type TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    public_data JSONB NOT NULL DEFAULT '{}',
    internal_truth JSONB NOT NULL DEFAULT '{}',
    discovered BOOLEAN NOT NULL DEFAULT false,
    reliability INT NOT NULL DEFAULT 50,
    related_suspect_ids JSONB NOT NULL DEFAULT '[]',
    related_location_ids JSONB NOT NULL DEFAULT '[]',
    related_timeline_event_ids JSONB NOT NULL DEFAULT '[]',
    attachment_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_evidence_case_id ON evidence(case_id);
CREATE INDEX idx_evidence_discovered ON evidence(discovered);

CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT '',
    latitude DOUBLE PRECISION NOT NULL DEFAULT 0,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    discovered BOOLEAN NOT NULL DEFAULT false,
    related_evidence_ids JSONB NOT NULL DEFAULT '[]',
    related_suspect_ids JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_locations_case_id ON locations(case_id);

CREATE TABLE real_timeline_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    occurred_at TIMESTAMPTZ NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    participants JSONB NOT NULL DEFAULT '[]',
    revealed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_real_timeline_events_case_id ON real_timeline_events(case_id);

CREATE TABLE player_timeline_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    occurred_at TIMESTAMPTZ NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    linked_evidence_ids JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_player_timeline_events_case_id ON player_timeline_events(case_id);

CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    suspect_id UUID NOT NULL REFERENCES suspects(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_conversations_case_id ON conversations(case_id);
CREATE UNIQUE INDEX idx_conversations_case_suspect ON conversations(case_id, suspect_id);

CREATE TABLE conversation_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    emotion TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_conversation_messages_conversation_id ON conversation_messages(conversation_id);

CREATE TABLE player_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_player_notes_case_id ON player_notes(case_id);

CREATE TABLE discovered_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    fact TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_discovered_facts_case_id ON discovered_facts(case_id);

CREATE TABLE solve_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    accused_suspect_id UUID NOT NULL,
    motive TEXT NOT NULL DEFAULT '',
    reasoning TEXT NOT NULL DEFAULT '',
    correct BOOLEAN NOT NULL DEFAULT false,
    score INT NOT NULL DEFAULT 0,
    feedback TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_solve_attempts_case_id ON solve_attempts(case_id);

CREATE TABLE agent_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID,
    agent_name TEXT NOT NULL,
    task_type TEXT NOT NULL,
    input_hash TEXT NOT NULL DEFAULT '',
    input JSONB NOT NULL DEFAULT '{}',
    output JSONB NOT NULL DEFAULT '{}',
    model TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    latency_ms INT NOT NULL DEFAULT 0,
    token_usage JSONB NOT NULL DEFAULT '{}',
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_agent_runs_case_id ON agent_runs(case_id);

CREATE TABLE case_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_case_events_case_id ON case_events(case_id);

CREATE TABLE evidence_inspections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evidence_id UUID NOT NULL REFERENCES evidence(id) ON DELETE CASCADE,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    question TEXT NOT NULL DEFAULT '',
    analysis TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_evidence_inspections_case_id ON evidence_inspections(case_id);
CREATE INDEX idx_evidence_inspections_evidence_id ON evidence_inspections(evidence_id);
