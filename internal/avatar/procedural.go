package avatar

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// stylePalettes maps each style to foreground tones used by the
// deterministic local generator.
var stylePalettes = map[string][3]color.NRGBA{
	"realistic":      {{0x6b, 0x4f, 0x3a, 0xff}, {0xa8, 0x8b, 0x6f, 0xff}, {0x3e, 0x2f, 0x23, 0xff}},
	"semi-realistic": {{0x5a, 0x6e, 0x8c, 0xff}, {0x9d, 0xb2, 0xc9, 0xff}, {0x2f, 0x3e, 0x54, 0xff}},
	"cartoon":        {{0xff, 0x8a, 0x3d, 0xff}, {0xff, 0xc4, 0x5c, 0xff}, {0xe8, 0x54, 0x2f, 0xff}},
	"anime":          {{0xc0, 0x5c, 0xd6, 0xff}, {0xf2, 0xa6, 0xe8, 0xff}, {0x6e, 0x2f, 0x8c, 0xff}},
	"fantasy":        {{0x3d, 0x8c, 0x5a, 0xff}, {0x8c, 0xc9, 0x7a, 0xff}, {0x1f, 0x54, 0x38, 0xff}},
	"business":       {{0x37, 0x47, 0x5a, 0xff}, {0x7a, 0x8c, 0xa0, 0xff}, {0x1c, 0x26, 0x33, 0xff}},
	"modern-minimal": {{0x4a, 0x4a, 0x4a, 0xff}, {0xb0, 0xb0, 0xb0, 0xff}, {0x21, 0x21, 0x21, 0xff}},
	"gaming":         {{0x2f, 0xd6, 0x5c, 0xff}, {0x1f, 0x8a, 0xe8, 0xff}, {0x12, 0x2e, 0x1c, 0xff}},
	"cyberpunk":      {{0xff, 0x2e, 0x88, 0xff}, {0x2e, 0xe6, 0xff, 0xff}, {0x38, 0x12, 0x54, 0xff}},
}

// Procedural renders a deterministic, smooth tactical portrait placeholder.
// It needs no external service, making avatars always available without
// presenting pixel-art geometry as a generated portrait.
func Procedural(spec Spec) ([]byte, error) {
	size := spec.Size
	if size <= 0 {
		size = 512
	}

	h := fnv.New64a()
	h.Write([]byte(spec.Seed + "|" + spec.Style + "|" + spec.Gender + "|" + spec.AgeGroup + "|" + spec.Ethnicity))
	bits := h.Sum64()

	palette, ok := stylePalettes[spec.Style]
	if !ok {
		palette = stylePalettes["modern-minimal"]
	}
	// Preserve the palette identity while introducing a deterministic tonal
	// variation so different seeds never collapse into the same placeholder.
	palette[1].R ^= byte(bits >> 8)
	palette[1].G ^= byte(bits >> 16)
	palette[1].B ^= byte(bits >> 24)

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	cx := size / 2
	face := size * 29 / 100
	shoulder := size * 46 / 100
	variant := int(bits % 13)
	fillCircle(img, cx, size*42/100, face, palette[1])
	fillCircle(img, cx, size*25/100, face*62/100, palette[2])
	fillCircle(img, cx-face/2, size*42/100, face*16/100, palette[0])
	fillCircle(img, cx+face/2, size*42/100, face*16/100, palette[0])
	fillEllipse(img, cx, size*91/100, shoulder+(shoulder*variant/130), shoulder*42/100, palette[0])
	fillEllipse(img, cx, size*98/100, shoulder*72/100, shoulder*18/100, palette[2])
	// A quiet rim light keeps the placeholder legible inside dark HUD panels.
	fillCircle(img, cx, size*42/100, face*102/100, color.NRGBA{0x5f, 0xff, 0xb2, 0x22})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fillCell paints one grid cell, clipped to the avatar's circular mask so
// the corners of the square canvas stay fully transparent.
func fillCircle(img *image.NRGBA, cx, cy, radius int, c color.NRGBA) {
	fillEllipse(img, cx, cy, radius, radius, c)
}

func fillEllipse(img *image.NRGBA, cx, cy, rx, ry int, c color.NRGBA) {
	if rx <= 0 || ry <= 0 {
		return
	}
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if dx*dx+dy*dy <= 1 {
				if c.A == 0xff {
					img.SetNRGBA(x, y, c)
				} else {
					draw.Draw(img, image.Rect(x, y, x+1, y+1), image.NewUniform(c), image.Point{}, draw.Over)
				}
			}
		}
	}
}
