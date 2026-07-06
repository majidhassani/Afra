// Package visualasset generates and persists character avatars and clue
// images on demand, then stores the resulting image as a self-contained data
// URL so the client can render it with no external storage dependency.
//
// Privacy rule: generation prompts are internal inputs. This package returns
// only *_url and *_status; it never echoes a prompt to the client.
package visualasset

import (
	"context"
	"encoding/base64"
	"log/slog"

	"github.com/google/uuid"

	"casemind/internal/avatar"
	"casemind/internal/character"
	"casemind/internal/clue"
)

// Asset statuses shared by characters and clues.
const (
	StatusReady       = "ready"
	StatusUnavailable = "unavailable"
)

// Generator renders an image from an avatar spec (image API or procedural
// fallback). Implemented by *avatar.Service.
type Generator interface {
	Generate(ctx context.Context, spec avatar.Spec) (*avatar.Result, error)
}

// Ownership guards access to a mission's assets. Implemented by mission.Service.
type Ownership interface {
	EnsureOwned(ctx context.Context, userID, missionID uuid.UUID) error
}

type Service struct {
	chars character.Repository
	clues clue.Repository
	gen   Generator
	owner Ownership
	log   *slog.Logger
}

func NewService(chars character.Repository, clues clue.Repository, gen Generator, owner Ownership, log *slog.Logger) *Service {
	return &Service{chars: chars, clues: clues, gen: gen, owner: owner, log: log}
}

// CharacterAvatar generates (or regenerates) a character's avatar and persists
// it. It returns the public character with avatar_url/avatar_status set. On
// generation failure it records "unavailable" so the UI shows a placeholder
// instead of blocking gameplay.
func (s *Service) CharacterAvatar(ctx context.Context, userID, missionID, characterID uuid.UUID) (*character.PublicCharacter, error) {
	if err := s.owner.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.chars.GetByID(ctx, missionID, characterID)
	if err != nil {
		return nil, err
	}
	dataURL, ok := s.render(ctx, avatar.Spec{
		Style:    "semi-realistic",
		AgeGroup: ageBand(c.Age),
		Seed:     c.ID.String(),
	})
	status := StatusReady
	if !ok {
		status = StatusUnavailable
		dataURL = ""
	}
	if err := s.chars.UpdateAvatar(ctx, characterID, dataURL, status); err != nil {
		return nil, err
	}
	c.AvatarURL = dataURL
	c.AvatarStatus = status
	pub := c.Public()
	return &pub, nil
}

// ClueImage generates (or regenerates) a clue's evidence image and persists it.
func (s *Service) ClueImage(ctx context.Context, userID, missionID, clueID uuid.UUID) (*clue.PublicClue, error) {
	if err := s.owner.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	c, err := s.clues.GetByID(ctx, missionID, clueID)
	if err != nil {
		return nil, err
	}
	dataURL, ok := s.render(ctx, avatar.Spec{
		Style: "realistic",
		Seed:  c.ID.String(),
		Size:  512,
	})
	status := StatusReady
	if !ok {
		status = StatusUnavailable
		dataURL = ""
	}
	if err := s.clues.UpdateImage(ctx, clueID, dataURL, status); err != nil {
		return nil, err
	}
	c.ImageURL = dataURL
	c.ImageStatus = status
	pub := c.Public()
	return &pub, nil
}

// render produces a base64 PNG data URL for the spec. The procedural fallback
// always succeeds, so ok is false only on an unexpected internal error.
func (s *Service) render(ctx context.Context, spec avatar.Spec) (string, bool) {
	res, err := s.gen.Generate(ctx, spec)
	if err != nil || res == nil || len(res.PNG) == 0 {
		if err != nil {
			s.log.Warn("visual asset generation failed", "error", err)
		}
		return "", false
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(res.PNG), true
}

func ageBand(age int) string {
	switch {
	case age <= 0:
		return "adult"
	case age < 18:
		return "child"
	case age < 35:
		return "young-adult"
	case age < 50:
		return "adult"
	case age < 65:
		return "middle-aged"
	default:
		return "senior"
	}
}
