package publicapi

import (
	"encoding/binary"
	"image"

	"golang.org/x/image/draw"
)

// exifOrientation returns the EXIF orientation of a JPEG file, from 1 to 8.
// It returns 1 for a file without the tag and for a file it cannot read.
func exifOrientation(data []byte) int {
	const (
		markerSOS      = 0xDA
		markerAPP1     = 0xE1
		tagOrientation = 0x0112
	)
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		// Fill bytes can come before a marker.
		if marker == 0xFF {
			i++
			continue
		}
		if marker == markerSOS {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		segment := data[i+4 : i+2+size]
		if marker == markerAPP1 && len(segment) >= 6 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:], tagOrientation)
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(tiff []byte, tag uint16) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	offset := int(order.Uint32(tiff[4:]))
	if offset < 8 || offset+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[offset:]))
	for n := range count {
		entry := offset + 2 + n*12
		if entry+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[entry:]) != tag {
			continue
		}
		if v := int(order.Uint16(tiff[entry+8:])); v >= 1 && v <= 8 {
			return v
		}
		return 1
	}
	return 1
}

// orient returns the image as the EXIF orientation tells to show it.
func orient(img *image.RGBA, orientation int) *image.RGBA {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	// The orientations 5 to 8 swap the width and the height.
	swap := orientation >= 5
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	if swap {
		out = image.NewRGBA(image.Rect(0, 0, h, w))
	}
	for y := range h {
		for x := range w {
			var dx, dy int
			switch orientation {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			copy(out.Pix[out.PixOffset(dx, dy):][:4], img.Pix[img.PixOffset(x, y):][:4])
		}
	}
	return out
}

// fit makes the image smaller until it fits in a square of side pixels. It
// keeps the aspect ratio. It does not make a small image larger.
func fit(img image.Image, side int) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > side || h > side {
		if w >= h {
			w, h = side, max(1, h*side/w)
		} else {
			w, h = max(1, w*side/h), side
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	if w == b.Dx() && h == b.Dy() {
		draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	} else {
		draw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Src, nil)
	}
	return out
}
