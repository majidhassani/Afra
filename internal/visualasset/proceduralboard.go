package visualasset

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// boardPalettes gives each mission type a day/night gradient so the fallback
// board still carries the scenario's mood.
var boardPalettes = map[string][2]color.NRGBA{
	"detective":         {{R: 0x2b, G: 0x34, B: 0x4a, A: 0xff}, {R: 0x0f, G: 0x12, B: 0x1c, A: 0xff}},
	"wildlife_rescue":   {{R: 0x2e, G: 0x54, B: 0x39, A: 0xff}, {R: 0x10, G: 0x22, B: 0x16, A: 0xff}},
	"disaster_response": {{R: 0x6b, G: 0x45, B: 0x26, A: 0xff}, {R: 0x2a, G: 0x18, B: 0x0c, A: 0xff}},
	"exploration":       {{R: 0x1d, G: 0x2b, B: 0x53, A: 0xff}, {R: 0x07, G: 0x0a, B: 0x1a, A: 0xff}},
	"survival":          {{R: 0x4d, G: 0x5d, B: 0x6e, A: 0xff}, {R: 0x1a, G: 0x22, B: 0x2c, A: 0xff}},
	"diplomacy":         {{R: 0x3a, G: 0x3f, B: 0x58, A: 0xff}, {R: 0x14, G: 0x16, B: 0x24, A: 0xff}},
	"medical_mystery":   {{R: 0x3f, G: 0x2b, B: 0x45, A: 0xff}, {R: 0x15, G: 0x0d, B: 0x1a, A: 0xff}},
}

// ProceduralBoard renders a deterministic gradient board (1024×576) so the
// scene shell always has a themed backdrop even without an image API.
// Night/dusk darkens the palette; high risk warms the horizon.
func ProceduralBoard(missionType, timeOfDay, weather string, risk int) []byte {
	pal, ok := boardPalettes[missionType]
	if !ok {
		pal = boardPalettes["detective"]
	}
	top, bottom := pal[0], pal[1]

	dark := 0
	switch timeOfDay {
	case "night":
		dark = 60
	case "dusk":
		dark = 30
	}
	if weather == "storm" || weather == "fog" {
		dark += 15
	}
	top = darken(top, dark)
	bottom = darken(bottom, dark)
	if risk >= 60 {
		// A warm danger tint low on the horizon.
		bottom.R = clamp8(int(bottom.R) + 40)
	}

	const w, h = 1024, 576
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		c := lerp(top, bottom, float64(y)/float64(h-1))
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

func lerp(a, b color.NRGBA, t float64) color.NRGBA {
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: 0xff}
}

func darken(c color.NRGBA, amount int) color.NRGBA {
	sub := func(v uint8) uint8 { return clamp8(int(v) - amount) }
	return color.NRGBA{R: sub(c.R), G: sub(c.G), B: sub(c.B), A: 0xff}
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
