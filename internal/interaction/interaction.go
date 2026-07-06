// Package interaction persists dialogue threads: NPC chats and AI guidance
// conversations, generalized from the old interrogation conversations.
package interaction

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

// Interaction types.
const (
	TypeCharacterChat = "character_chat"
	TypeGuidance      = "guidance"
)

// Message senders.
const (
	SenderPlayer    = "player"
	SenderCharacter = "character"
	SenderAI        = "ai"
)

type Interaction struct {
	ID          uuid.UUID  `json:"id"`
	MissionID   uuid.UUID  `json:"mission_id"`
	UserID      uuid.UUID  `json:"user_id"`
	CharacterID *uuid.UUID `json:"character_id,omitempty"`
	Type        string     `json:"interaction_type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Message struct {
	ID            uuid.UUID       `json:"id"`
	InteractionID uuid.UUID       `json:"interaction_id"`
	Sender        string          `json:"sender"`
	Content       string          `json:"content"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
}

type Repository interface {
	// FindOrCreate returns the single thread for (mission, type, character).
	FindOrCreate(ctx context.Context, missionID, userID uuid.UUID, interactionType string, characterID *uuid.UUID) (*Interaction, error)
	AddMessage(ctx context.Context, m *Message) error
	// Recent returns the last n messages in chronological order.
	Recent(ctx context.Context, interactionID uuid.UUID, n int) ([]Message, error)
	CountByMission(ctx context.Context, missionID uuid.UUID) (int, error)
	// CountInteractedCharacters returns how many distinct characters the player
	// has actually exchanged messages with (character chats with at least one
	// player message).
	CountInteractedCharacters(ctx context.Context, missionID uuid.UUID) (int, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) FindOrCreate(ctx context.Context, missionID, userID uuid.UUID, interactionType string, characterID *uuid.UUID) (*Interaction, error) {
	in := &Interaction{MissionID: missionID, UserID: userID, CharacterID: characterID, Type: interactionType}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO interactions (mission_id, user_id, character_id, interaction_type)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (mission_id, interaction_type, COALESCE(character_id, '00000000-0000-0000-0000-000000000000'::uuid))
		 DO UPDATE SET updated_at = now()
		 RETURNING id, created_at, updated_at`,
		missionID, userID, characterID, interactionType,
	).Scan(&in.ID, &in.CreatedAt, &in.UpdatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "find or create interaction")
	}
	return in, nil
}

func (r *PGRepository) AddMessage(ctx context.Context, m *Message) error {
	metadata := m.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO interaction_messages (interaction_id, sender, content, metadata)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		m.InteractionID, m.Sender, m.Content, metadata,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "insert interaction message")
	}
	return nil
}

func (r *PGRepository) Recent(ctx context.Context, interactionID uuid.UUID, n int) ([]Message, error) {
	if n <= 0 || n > 100 {
		n = 30
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, interaction_id, sender, content, metadata, created_at FROM (
		     SELECT id, interaction_id, sender, content, metadata, created_at
		     FROM interaction_messages WHERE interaction_id = $1
		     ORDER BY created_at DESC LIMIT $2
		 ) recent ORDER BY created_at ASC`, interactionID, n)
	if err != nil {
		return nil, apperrors.Internal(err, "list interaction messages")
	}
	defer rows.Close()
	messages := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.InteractionID, &m.Sender, &m.Content, &m.Metadata, &m.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan interaction message")
		}
		messages = append(messages, m)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate interaction messages")
	}
	return messages, nil
}

func (r *PGRepository) CountByMission(ctx context.Context, missionID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM interaction_messages im
		 JOIN interactions i ON i.id = im.interaction_id
		 WHERE i.mission_id = $1 AND im.sender = $2`, missionID, SenderPlayer).Scan(&n)
	if err != nil {
		return 0, apperrors.Internal(err, "count mission interactions")
	}
	return n, nil
}

func (r *PGRepository) CountInteractedCharacters(ctx context.Context, missionID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(DISTINCT i.character_id) FROM interactions i
		 WHERE i.mission_id = $1 AND i.interaction_type = $2 AND i.character_id IS NOT NULL
		   AND EXISTS (
		     SELECT 1 FROM interaction_messages im
		     WHERE im.interaction_id = i.id AND im.sender = $3
		   )`, missionID, TypeCharacterChat, SenderPlayer).Scan(&n)
	if err != nil {
		return 0, apperrors.Internal(err, "count interacted characters")
	}
	return n, nil
}
