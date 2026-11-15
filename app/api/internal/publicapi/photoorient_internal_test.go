package publicapi

import (
	"image"
	"image/color"
	"testing"
)

func TestOrient(t *testing.T) {
	// corners has the new place of the top left pixel of a 3 by 2 image.
	corners := map[int]image.Point{
		1: {0, 0}, 2: {2, 0}, 3: {2, 1}, 4: {0, 1},
		5: {0, 0}, 6: {1, 0}, 7: {1, 2}, 8: {0, 2},
	}
	// seconds has the new place of the pixel to the right of it.
	seconds := map[int]image.Point{
		1: {1, 0}, 2: {1, 0}, 3: {1, 1}, 4: {1, 1},
		5: {0, 1}, 6: {1, 1}, 7: {1, 1}, 8: {0, 1},
	}
	for orientation := 1; orientation <= 8; orientation++ {
		img := image.NewRGBA(image.Rect(0, 0, 3, 2))
		img.Set(0, 0, color.RGBA{255, 0, 0, 255})
		img.Set(1, 0, color.RGBA{0, 255, 0, 255})
		out := orient(img, orientation)
		wantSize := image.Pt(3, 2)
		if orientation >= 5 {
			wantSize = image.Pt(2, 3)
		}
		if out.Bounds().Size() != wantSize {
			t.Errorf("orientation %d: size %v", orientation, out.Bounds().Size())
			continue
		}
		if c := corners[orientation]; out.RGBAAt(c.X, c.Y) != (color.RGBA{255, 0, 0, 255}) {
			t.Errorf("orientation %d: the first pixel is not at %v", orientation, c)
		}
		if c := seconds[orientation]; out.RGBAAt(c.X, c.Y) != (color.RGBA{0, 255, 0, 255}) {
			t.Errorf("orientation %d: the second pixel is not at %v", orientation, c)
		}
	}
}

// littleEndianExif makes an APP1 segment with one IFD entry for the
// orientation.
func littleEndianExif(orientation uint16) []byte {
	tiff := []byte{'I', 'I', 0x2a, 0, 8, 0, 0, 0, 1, 0}
	tiff = append(tiff, 0x12, 0x01, 3, 0, 1, 0, 0, 0, byte(orientation), byte(orientation>>8), 0, 0)
	tiff = append(tiff, 0, 0, 0, 0)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	return append([]byte{0xFF, 0xE1, 0, byte(len(payload) + 2)}, payload...)
}

func TestExifOrientation(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want int
	}{
		{"no EXIF", []byte{0xFF, 0xD8, 0xFF, 0xDA, 0, 2}, 1},
		{"not a JPEG", []byte("\x89PNG\r\n\x1a\n"), 1},
		{"orientation 6", append([]byte{0xFF, 0xD8}, littleEndianExif(6)...), 6},
		{"orientation 9", append([]byte{0xFF, 0xD8}, littleEndianExif(9)...), 1},
		{"cut segment", append([]byte{0xFF, 0xD8}, littleEndianExif(6)[:20]...), 1},
		{"segment size 0", []byte{0xFF, 0xD8, 0xFF, 0xE1, 0, 0, 0, 0}, 1},
	} {
		if got := exifOrientation(tc.data); got != tc.want {
			t.Errorf("%s: orientation %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestFit(t *testing.T) {
	for _, tc := range []struct{ w, h, side, wantW, wantH int }{
		{4000, 1000, 2048, 2048, 512},
		{1000, 4000, 2048, 512, 2048},
		{100, 50, 2048, 100, 50},
		{5000, 1, 480, 480, 1},
	} {
		out := fit(image.NewRGBA(image.Rect(0, 0, tc.w, tc.h)), tc.side)
		if out.Bounds().Dx() != tc.wantW || out.Bounds().Dy() != tc.wantH {
			t.Errorf("%dx%d: %v", tc.w, tc.h, out.Bounds().Size())
		}
	}
}

func BenchmarkFitLargest(b *testing.B) {
	img := image.NewYCbCr(image.Rect(0, 0, 6000, 4000), image.YCbCrSubsampleRatio420)
	for b.Loop() {
		fit(img, fullSide)
	}
}
