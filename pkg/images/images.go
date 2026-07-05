// Package images validates user-uploaded images before they are forwarded
// to a vision-capable model: MIME allowlist, decoded size, and pixel
// dimensions. PNG, JPEG and WEBP are supported.
package images

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"

	"casemind/internal/llm"
	apperrors "casemind/pkg/errors"
)

const (
	// MaxBytes is the maximum decoded image size (per image).
	MaxBytes = 4 << 20 // 4 MiB
	// MaxPixels is the maximum width/height in pixels.
	MaxPixels = 4096
	// MaxPerRequest caps how many images one request may attach.
	MaxPerRequest = 4
)

// Payload is the wire format for an uploaded image: base64 data + MIME.
type Payload struct {
	Data string `json:"data"` // base64 (raw, no data: prefix)
	MIME string `json:"mime"`
}

// DecodeAndValidate turns uploaded payloads into llm.Image attachments,
// enforcing count, MIME, byte-size and pixel-dimension limits.
func DecodeAndValidate(payloads []Payload) ([]llm.Image, error) {
	if len(payloads) == 0 {
		return nil, nil
	}
	if len(payloads) > MaxPerRequest {
		return nil, apperrors.Invalid("too_many_images", fmt.Sprintf("at most %d images per request", MaxPerRequest))
	}
	out := make([]llm.Image, 0, len(payloads))
	for i, p := range payloads {
		img, err := decodeOne(p)
		if err != nil {
			return nil, apperrors.Invalid("invalid_image", fmt.Sprintf("image %d: %v", i+1, err))
		}
		out = append(out, img)
	}
	return out, nil
}

func decodeOne(p Payload) (llm.Image, error) {
	mime := strings.ToLower(strings.TrimSpace(p.MIME))
	if mime == "image/jpg" {
		mime = "image/jpeg"
	}
	if !llm.SupportedImageMIMEs[mime] {
		return llm.Image{}, fmt.Errorf("unsupported MIME %q (allowed: png, jpeg, webp)", p.MIME)
	}
	// Strip an optional data-URL prefix so both raw and data: payloads work.
	data := p.Data
	if idx := strings.Index(data, ";base64,"); idx >= 0 {
		data = data[idx+len(";base64,"):]
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return llm.Image{}, fmt.Errorf("invalid base64 data")
	}
	if len(raw) == 0 {
		return llm.Image{}, fmt.Errorf("empty image")
	}
	if len(raw) > MaxBytes {
		return llm.Image{}, fmt.Errorf("image exceeds %d MiB limit", MaxBytes>>20)
	}
	// The declared MIME must match the actual content.
	sniffed := http.DetectContentType(raw)
	if sniffed != mime {
		return llm.Image{}, fmt.Errorf("content type %s does not match declared %s", sniffed, mime)
	}
	w, h, err := dimensions(raw, mime)
	if err != nil {
		return llm.Image{}, err
	}
	if w > MaxPixels || h > MaxPixels {
		return llm.Image{}, fmt.Errorf("dimensions %dx%d exceed %dpx limit", w, h, MaxPixels)
	}
	return llm.Image{Data: raw, MIME: mime}, nil
}

func dimensions(raw []byte, mime string) (int, int, error) {
	if mime == "image/webp" {
		return webpDimensions(raw)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return 0, 0, fmt.Errorf("undecodable image: %v", err)
	}
	return cfg.Width, cfg.Height, nil
}

// webpDimensions parses the RIFF/WEBP header (VP8, VP8L and VP8X variants)
// without pulling in a full webp decoder dependency.
func webpDimensions(raw []byte) (int, int, error) {
	if len(raw) < 30 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WEBP" {
		return 0, 0, fmt.Errorf("not a valid WEBP file")
	}
	switch string(raw[12:16]) {
	case "VP8X":
		// 24-bit little-endian canvas size minus one at offsets 24 and 27.
		w := int(raw[24]) | int(raw[25])<<8 | int(raw[26])<<16
		h := int(raw[27]) | int(raw[28])<<8 | int(raw[29])<<16
		return w + 1, h + 1, nil
	case "VP8L":
		if len(raw) < 25 || raw[20] != 0x2f {
			return 0, 0, fmt.Errorf("invalid VP8L header")
		}
		bits := binary.LittleEndian.Uint32(raw[21:25])
		return int(bits&0x3fff) + 1, int((bits>>14)&0x3fff) + 1, nil
	case "VP8 ":
		if len(raw) < 30 {
			return 0, 0, fmt.Errorf("invalid VP8 header")
		}
		w := int(binary.LittleEndian.Uint16(raw[26:28])) & 0x3fff
		h := int(binary.LittleEndian.Uint16(raw[28:30])) & 0x3fff
		return w, h, nil
	}
	return 0, 0, fmt.Errorf("unknown WEBP variant")
}
