// Package avatar generates square profile avatars with transparent
// backgrounds. Generation is provider-agnostic: an image-capable API
// (llm.ImageGenerator) renders the avatar when configured, and a
// deterministic local generator is the guaranteed fallback so the feature
// works even when no image model is available (the current Gemini chat
// gateway cannot generate images).
package avatar

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"casemind/internal/llm"
	apperrors "casemind/pkg/errors"
)

// Styles supported for avatar generation. The prompt fragments steer
// API-based providers; the procedural fallback maps them to palettes.
var Styles = []string{
	"realistic", "semi-realistic", "cartoon", "anime", "fantasy",
	"business", "modern-minimal", "gaming", "cyberpunk",
}

var Genders = []string{"male", "female", "unspecified"}

var AgeGroups = []string{"child", "young-adult", "adult", "middle-aged", "senior"}

// Spec describes the requested avatar.
type Spec struct {
	Style     string `json:"style"`
	Gender    string `json:"gender"`
	AgeGroup  string `json:"age_group"`
	Ethnicity string `json:"ethnicity"`
	// Seed makes generation deterministic for the same subject (e.g. a
	// character ID or username). Optional.
	Seed string `json:"seed"`
	// Size in pixels (square). Defaults to 512, capped at 1024.
	Size int `json:"size"`
}

func (s *Spec) normalize() error {
	s.Style = strings.ToLower(strings.TrimSpace(s.Style))
	s.Gender = strings.ToLower(strings.TrimSpace(s.Gender))
	s.AgeGroup = strings.ToLower(strings.TrimSpace(s.AgeGroup))
	s.Ethnicity = strings.TrimSpace(s.Ethnicity)
	if s.Style == "" {
		s.Style = "modern-minimal"
	}
	if !contains(Styles, s.Style) {
		return apperrors.Invalid("invalid_style", fmt.Sprintf("style must be one of: %s", strings.Join(Styles, ", ")))
	}
	if s.Gender == "" {
		s.Gender = "unspecified"
	}
	if !contains(Genders, s.Gender) {
		return apperrors.Invalid("invalid_gender", fmt.Sprintf("gender must be one of: %s", strings.Join(Genders, ", ")))
	}
	if s.AgeGroup == "" {
		s.AgeGroup = "adult"
	}
	if !contains(AgeGroups, s.AgeGroup) {
		return apperrors.Invalid("invalid_age_group", fmt.Sprintf("age_group must be one of: %s", strings.Join(AgeGroups, ", ")))
	}
	if len(s.Ethnicity) > 60 {
		return apperrors.Invalid("invalid_ethnicity", "ethnicity must be at most 60 characters")
	}
	if s.Size <= 0 {
		s.Size = 512
	}
	if s.Size > 1024 {
		s.Size = 1024
	}
	return nil
}

// Prompt builds a compact, non-redundant image prompt for API providers.
// Kept short deliberately: image prompts are billed like any other tokens.
func (s Spec) Prompt() string {
	parts := []string{s.Style, "style avatar portrait"}
	if s.Gender != "unspecified" {
		parts = append(parts, s.Gender)
	}
	parts = append(parts, s.AgeGroup)
	if s.Ethnicity != "" {
		parts = append(parts, s.Ethnicity)
	}
	parts = append(parts, "head and shoulders, centered, square, transparent background, no backdrop, high quality")
	return strings.Join(parts, ", ")
}

// Result is the generated avatar.
type Result struct {
	PNG      []byte
	Provider string
}

// Service generates avatars, preferring the configured image API and
// falling back to the deterministic local generator on failure.
type Service struct {
	imageAPI llm.ImageGenerator // nil when no image API is configured
	log      *slog.Logger
}

func NewService(imageAPI llm.ImageGenerator, log *slog.Logger) *Service {
	return &Service{imageAPI: imageAPI, log: log}
}

func (s *Service) Generate(ctx context.Context, spec Spec) (*Result, error) {
	if err := spec.normalize(); err != nil {
		return nil, err
	}
	if s.imageAPI != nil {
		start := time.Now()
		png, err := s.imageAPI.GenerateImage(ctx, llm.ImageRequest{
			Prompt:      spec.Prompt(),
			Size:        fmt.Sprintf("%dx%d", spec.Size, spec.Size),
			Transparent: true,
		})
		if err == nil {
			s.log.Info("avatar generated", "provider", s.imageAPI.Name(),
				"style", spec.Style, "latency_ms", time.Since(start).Milliseconds())
			return &Result{PNG: png, Provider: s.imageAPI.Name()}, nil
		}
		s.log.Warn("image API avatar generation failed, using procedural fallback",
			"provider", s.imageAPI.Name(), "error", err)
	}
	png, err := Procedural(spec)
	if err != nil {
		return nil, apperrors.Internal(err, "procedural avatar generation")
	}
	return &Result{PNG: png, Provider: "procedural"}, nil
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
