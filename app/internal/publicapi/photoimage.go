package publicapi

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

const (
	maxUploadBytes  = 10 << 20
	maxUploadPixels = 25_000_000
	maxUploadSide   = 8192
	fullSide        = 2048
	thumbSide       = 480
	jpegQuality     = 85
)

// uploadTypes maps each upload content type to its name in the 422 text.
var uploadTypes = map[string]string{"image/jpeg": "JPEG", "image/png": "PNG", "image/webp": "WebP"}

// processedPhoto has the two JPEG files of a photo. width and height are
// the size of full.
type processedPhoto struct {
	full, thumb   []byte
	width, height int
}

// processPhoto makes new JPEG files from an upload. The JPEG encoder writes
// no metadata, such as EXIF or GPS data. A non-empty detail is the text of a
// 422 response.
func processPhoto(data []byte, contentType string) (processedPhoto, string, error) {
	if !hasMagic(data, contentType) {
		return processedPhoto{}, fmt.Sprintf("The body is not a %s image.", uploadTypes[contentType]), nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return processedPhoto{}, "The image cannot be read.", nil
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxUploadSide || cfg.Height > maxUploadSide ||
		cfg.Width*cfg.Height > maxUploadPixels {
		return processedPhoto{}, "The image must have at most 25 megapixels and at most 8192 pixels on each side.", nil
	}
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return processedPhoto{}, "The image cannot be read.", nil
	}

	full := imaging.Fit(img, fullSide, fullSide, imaging.Lanczos)
	thumb := imaging.Fit(full, thumbSide, thumbSide, imaging.Lanczos)
	out := processedPhoto{width: full.Bounds().Dx(), height: full.Bounds().Dy()}
	if out.full, err = encodeJPEG(full); err != nil {
		return processedPhoto{}, "", err
	}
	if out.thumb, err = encodeJPEG(thumb); err != nil {
		return processedPhoto{}, "", err
	}
	return out, "", nil
}

func hasMagic(data []byte, contentType string) bool {
	switch contentType {
	case "image/jpeg":
		return bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF})
	case "image/png":
		return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n"))
	case "image/webp":
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	}
	return false
}

// encodeJPEG puts img on a white background. JPEG has no transparency.
func encodeJPEG(img image.Image) ([]byte, error) {
	b := img.Bounds()
	flat := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(flat, flat.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, b.Min, draw.Over)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flat, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
