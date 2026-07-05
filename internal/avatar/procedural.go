package avatar

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
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

// Procedural renders a deterministic identicon-style avatar: a square
// transparent PNG with a circular mirrored pattern derived from the spec.
// It needs no external service, making avatars always available.
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

	const grid = 8 // 8x8 cells, left half mirrored to the right
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	cell := size / grid
	cx, cy := size/2, size/2
	radius := size / 2

	for row := 0; row < grid; row++ {
		for col := 0; col < grid/2; col++ {
			bitIndex := uint(row*(grid/2) + col)
			// Two bits per cell: off / palette[0..2].
			v := (bits >> ((bitIndex * 2) % 62)) & 0b11
			if v == 0 {
				continue
			}
			c := palette[v-1]
			fillCell(img, col, row, cell, c, cx, cy, radius)
			fillCell(img, grid-1-col, row, cell, c, cx, cy, radius) // mirror
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fillCell paints one grid cell, clipped to the avatar's circular mask so
// the corners of the square canvas stay fully transparent.
func fillCell(img *image.NRGBA, col, row, cell int, c color.NRGBA, cx, cy, radius int) {
	r2 := radius * radius
	for y := row * cell; y < (row+1)*cell; y++ {
		for x := col * cell; x < (col+1)*cell; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}
