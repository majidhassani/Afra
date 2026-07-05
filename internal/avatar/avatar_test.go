package avatar

import (
	"bytes"
	"context"
	"image/png"
	"io"
	"log/slog"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestProceduralAvatarIsSquareTransparentPNG(t *testing.T) {
	data, err := Procedural(Spec{Style: "cyberpunk", Seed: "user-1", Size: 256})
	if err != nil {
		t.Fatalf("Procedural: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not valid PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 256 || b.Dy() != 256 {
		t.Errorf("expected 256x256, got %dx%d", b.Dx(), b.Dy())
	}
	// Corners lie outside the circular mask and must be fully transparent.
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Errorf("corner pixel is not transparent (alpha=%d)", a)
	}
}

func TestProceduralAvatarIsDeterministic(t *testing.T) {
	a, _ := Procedural(Spec{Style: "anime", Seed: "same"})
	b, _ := Procedural(Spec{Style: "anime", Seed: "same"})
	if !bytes.Equal(a, b) {
		t.Error("same spec must produce identical avatars")
	}
	c, _ := Procedural(Spec{Style: "anime", Seed: "different"})
	if bytes.Equal(a, c) {
		t.Error("different seeds must produce different avatars")
	}
}

func TestServiceRejectsInvalidStyle(t *testing.T) {
	svc := NewService(nil, discardLogger())
	if _, err := svc.Generate(context.Background(), Spec{Style: "vaporwave"}); err == nil {
		t.Error("expected invalid style error")
	}
}

func TestServiceFallsBackToProcedural(t *testing.T) {
	svc := NewService(nil, discardLogger())
	res, err := svc.Generate(context.Background(), Spec{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.Provider != "procedural" {
		t.Errorf("provider = %q", res.Provider)
	}
	if _, err := png.Decode(bytes.NewReader(res.PNG)); err != nil {
		t.Errorf("not a PNG: %v", err)
	}
}
