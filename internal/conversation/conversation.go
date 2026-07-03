// Package conversation persists interrogation transcripts.
package conversation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Conversation struct {
	ID            uuid.UUID `json:"id"`
	CaseID        uuid.UUID `json:"case_id"`
	SuspectID     uuid.UUID `json:"suspect_id"`
	StartedAt     time.Time `json:"started_at"`
	LastMessageAt time.Time `json:"last_message_at"`
}

type Message struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Role           string    `json:"role"` // "detective" or "suspect"
	Content        string    `json:"content"`
	Emotion        string    `json:"emotion,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

const (
	RoleDetective = "detective"
	RoleSuspect   = "suspect"
)

type Repository interface {
	FindOrCreate(ctx context.Context, caseID, suspectID uuid.UUID) (*Conversation, error)
	AddMessage(ctx context.Context, m *Message) error
	Recent(ctx context.Context, conversationID uuid.UUID, limit int) ([]Message, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) FindOrCreate(ctx context.Context, caseID, suspectID uuid.UUID) (*Conversation, error) {
	c := &Conversation{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO conversations (case_id, suspect_id) VALUES ($1, $2)
		 ON CONFLICT (case_id, suspect_id) DO UPDATE SET last_message_at = conversations.last_message_at
		 RETURNING id, case_id, suspect_id, started_at, last_message_at`,
		caseID, suspectID,
	).Scan(&c.ID, &c.CaseID, &c.SuspectID, &c.StartedAt, &c.LastMessageAt)
	if err != nil {
		return nil, apperrors.Internal(err, "find or create conversation")
	}
	return c, nil
}

func (r *PGRepository) AddMessage(ctx context.Context, m *Message) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO conversation_messages (conversation_id, role, content, emotion)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		m.ConversationID, m.Role, m.Content, m.Emotion,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return apperrors.Internal(err, "add conversation message")
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE conversations SET last_message_at = now() WHERE id = $1`, m.ConversationID)
	if err != nil {
		return apperrors.Internal(err, "touch conversation")
	}
	return nil
}

// Recent returns the last `limit` messages in chronological order.
func (r *PGRepository) Recent(ctx context.Context, conversationID uuid.UUID, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, conversation_id, role, content, emotion, created_at FROM (
		    SELECT id, conversation_id, role, content, emotion, created_at
		    FROM conversation_messages WHERE conversation_id = $1
		    ORDER BY created_at DESC LIMIT $2
		 ) sub ORDER BY created_at ASC`, conversationID, limit)
	if err != nil {
		return nil, apperrors.Internal(err, "list recent messages")
	}
	defer rows.Close()
	messages := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.Emotion, &m.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan message")
		}
		messages = append(messages, m)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate messages")
	}
	return messages, nil
}
