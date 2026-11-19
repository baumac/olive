package publicapi

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestProcessPhotoRefusesBrokenImages(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	cut := buf.Bytes()[:buf.Len()-20]
	for _, tc := range []struct {
		name, contentType string
		data              []byte
	}{
		{"cut PNG", "image/png", cut},
		{"unknown type", "image/gif", buf.Bytes()},
	} {
		_, detail, err := processPhoto(tc.data, tc.contentType)
		if err != nil || detail == "" {
			t.Errorf("%s: detail %q, error %v", tc.name, detail, err)
		}
	}
}
