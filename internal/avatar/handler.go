package avatar

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Options returns the supported generation parameters so the frontend can
// render pickers without hardcoding them.
func (h *Handler) Options(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{
		"styles":     Styles,
		"genders":    Genders,
		"age_groups": AgeGroups,
	})
}

// Generate renders an avatar and returns it as base64 PNG. The response is
// JSON (not raw bytes) to fit the app-wide {data}/{error} envelope.
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	var spec Spec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		response.Err(w, apperrors.Invalid("invalid_body", "invalid JSON body"))
		return
	}
	result, err := h.svc.Generate(r.Context(), spec)
	if err != nil {
		response.Err(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"avatar": map[string]any{
			"png_base64": base64.StdEncoding.EncodeToString(result.PNG),
			"mime":       "image/png",
			"provider":   result.Provider,
			"style":      spec.Style,
		},
	})
}
