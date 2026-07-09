package visualasset

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/google/uuid"

	"casemind/internal/llm"
	apperrors "casemind/pkg/errors"
)

// Board types a mission can generate background art for.
const (
	BoardMission = "mission_board_background"
	BoardMap     = "map_board_background"
	BoardReport  = "report_center_background"
	BoardDebrief = "debrief_background"
	BoardLoading = "loading_screen"
)

var validBoardTypes = []string{BoardMission, BoardMap, BoardReport, BoardDebrief, BoardLoading}

// BoardGateway is the slice of the mission module board generation needs:
// player-safe prompt context in, persisted board image out. Implemented by
// mission.Service. Context values are primitives so neither package imports
// the other's types.
type BoardGateway interface {
	BoardContext(ctx context.Context, userID, missionID uuid.UUID) (missionType, region, weather, timeOfDay string, risk int, err error)
	SaveBoardArt(ctx context.Context, missionID uuid.UUID, boardType, dataURL, status string) (int, error)
}

// BoardResult is the client-facing outcome (URL + status only — the prompt
// stays server-side, same privacy rule as avatars).
type BoardResult struct {
	BoardType string `json:"board_type"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	Version   int    `json:"version"`
}

// Board generates (or regenerates) one scenario board background and persists
// it. The image API renders it when configured; the procedural theme board is
// the guaranteed fallback so the scene shell always has a backdrop.
func (s *Service) Board(ctx context.Context, userID, missionID uuid.UUID, boardType string) (*BoardResult, error) {
	if !isValidBoard(boardType) {
		return nil, apperrors.Invalid("invalid_board_type",
			"board_type must be one of: "+strings.Join(validBoardTypes, ", "))
	}
	if err := s.owner.EnsureOwned(ctx, userID, missionID); err != nil {
		return nil, err
	}
	missionType, region, weather, timeOfDay, risk, err := s.boards.BoardContext(ctx, userID, missionID)
	if err != nil {
		return nil, err
	}

	png := s.renderBoard(ctx, boardPrompt(boardType, missionType, region, weather, timeOfDay, risk))
	if png == nil {
		png = ProceduralBoard(missionType, timeOfDay, weather, risk)
	}
	status := StatusReady
	dataURL := ""
	if len(png) > 0 {
		dataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	} else {
		status = StatusUnavailable
	}
	version, err := s.boards.SaveBoardArt(ctx, missionID, boardType, dataURL, status)
	if err != nil {
		return nil, err
	}
	return &BoardResult{BoardType: boardType, URL: dataURL, Status: status, Version: version}, nil
}

// renderBoard tries the image API; nil means "use the procedural fallback".
func (s *Service) renderBoard(ctx context.Context, prompt string) []byte {
	if s.imageAPI == nil {
		return nil
	}
	png, err := s.imageAPI.GenerateImage(ctx, llm.ImageRequest{
		Prompt: prompt,
		Size:   "1024x1024",
	})
	if err != nil || len(png) == 0 {
		if err != nil {
			s.log.Warn("board generation failed, using procedural fallback", "error", err)
		}
		return nil
	}
	return png
}

// boardPrompt composes the internal generation prompt from player-safe world
// state. Hard constraints keep the art HUD-friendly. Never sent to clients.
func boardPrompt(boardType, missionType, region, weather, timeOfDay string, risk int) string {
	mood := "calm, hopeful"
	switch {
	case risk >= 80:
		mood = "tense, dangerous, high stakes"
	case risk >= 60:
		mood = "uneasy, rising tension"
	case risk >= 30:
		mood = "focused, mysterious"
	}
	scene := map[string]string{
		BoardMission: "wide establishing shot of the mission area",
		BoardMap:     "stylized aerial terrain view of the operation region",
		BoardReport:  "dim command post interior with empty desk surfaces",
		BoardDebrief: "quiet aftermath scene of the mission area at a distance",
		BoardLoading: "atmospheric wide vista of the mission region",
	}[boardType]
	parts := []string{
		"Premium cinematic mission board for AgentVerse",
		strings.ReplaceAll(missionType, "_", " ") + " mission scenario",
		scene,
	}
	if region != "" {
		parts = append(parts, region)
	}
	parts = append(parts,
		timeOfDay, weather+" weather", mood+" mood",
		"high-fidelity realistic game concept art, graphite and deep emerald palette, restrained cyan practical light",
		"consistent tactical investigation art direction, sharp atmospheric depth, 1024px source image",
		"large empty sky and foreground areas kept clear for HUD overlays",
		"no pixel art, no text, no logos, no UI labels, no watermarks, no people in focus",
	)
	return strings.Join(parts, ", ")
}

func isValidBoard(t string) bool {
	for _, v := range validBoardTypes {
		if v == t {
			return true
		}
	}
	return false
}
