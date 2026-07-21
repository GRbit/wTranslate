package icons

import (
	"bytes"
	"image/png"
	"testing"
)

// TestAppIconSizes verifies App renders a decodable PNG with the exact
// requested dimensions for every size installed into the hicolor theme
// (plus 512 for build/appicon.png and 32 for the tray pixmap).
func TestAppIconSizes(t *testing.T) {
	for _, size := range []int{16, 22, 24, 32, 48, 64, 128, 256, 512} {
		data := App(size)
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("App(%d): not a valid PNG: %v", size, err)
		}
		b := img.Bounds()
		if b.Dx() != size || b.Dy() != size {
			t.Errorf("App(%d): got %dx%d, want %dx%d", size, b.Dx(), b.Dy(), size, size)
		}
	}
}
