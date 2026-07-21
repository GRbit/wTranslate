// Package icons renders the application's icons at runtime from simple
// shape descriptions - no image files are committed or embedded. The gen
// subcommand writes PNGs to build/ for desktop integration and packaging;
// it runs as a wails pre-build hook. The tray icon is the app icon at
// StatusNotifierItem pixmap size (see App).
package icons

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// App draws the window/taskbar/tray icon at the given size: a white
// capital T on a blue rounded-square background. The geometry is authored
// in 512x512 coordinates and scaled.
func App(size int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := float64(size) / 512

	bg := color.NRGBA{0x2b, 0x6c, 0xb0, 0xff} // translator blue
	fg := color.NRGBA{0xf5, 0xf7, 0xfa, 0xff} // near-white

	// coverage from a signed distance in authoring units (1 device px soft edge)
	cov := func(d float64) float64 {
		return math.Min(1, math.Max(0, 0.5-d*scale))
	}
	// signed distance to a rounded rectangle centered at (cx,cy)
	roundRect := func(px, py, cx, cy, hw, hh, r float64) float64 {
		dx := math.Abs(px-cx) - (hw - r)
		dy := math.Abs(py-cy) - (hh - r)
		ax, ay := math.Max(dx, 0), math.Max(dy, 0)
		return math.Hypot(ax, ay) + math.Min(math.Max(dx, dy), 0) - r
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// sample position in 512x512 authoring coordinates
			px, py := (float64(x)+0.5)/scale, (float64(y)+0.5)/scale

			// background rounded square
			a := cov(roundRect(px, py, 256, 256, 240, 240, 96))
			if a <= 0 {
				continue
			}
			c := bg

			// capital T: horizontal bar plus stem, centered as a glyph
			t := roundRect(px, py, 256, 150, 140, 36, 18)
			t = math.Min(t, roundRect(px, py, 256, 274, 34, 124, 18))
			if ta := cov(t); ta > 0 {
				c = blend(c, fg, ta)
			}

			c.A = uint8(a * 255)
			img.SetNRGBA(x, y, c)
		}
	}
	return encode(img)
}

// blend mixes b over a with opacity t (0..1), ignoring alpha.
func blend(a, b color.NRGBA, t float64) color.NRGBA {
	mix := func(x, y uint8) uint8 {
		return uint8(float64(x)*(1-t) + float64(y)*t)
	}
	return color.NRGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), a.A}
}

func encode(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		// Cannot fail for a valid in-memory NRGBA image.
		panic(err)
	}
	return buf.Bytes()
}
