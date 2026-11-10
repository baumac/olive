package publicapi_test

import (
	"bytes"
	"cmp"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/baumac/reviews-app/app/internal/publicapi"
)

// webp1x1 is a lossless WebP image of one pixel. The Go libraries cannot
// encode WebP.
const webp1x1 = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

const reviewBody = `{"space":7,"service":8,"food":9,"price":6,"cleanliness":8,"title":"Good"}`

// testJPEG encodes an image of w by h pixels. A non-zero orientation adds
// an EXIF segment with that orientation and a GPS latitude.
func testJPEG(t *testing.T, w, h int, orientation uint16) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	if orientation == 0 {
		return data
	}
	return append(append([]byte{0xFF, 0xD8}, exifSegment(orientation)...), data[2:]...)
}

// exifSegment makes an APP1 segment with a big-endian TIFF header. IFD0 has
// the orientation and a pointer to a GPS IFD with a latitude reference.
func exifSegment(orientation uint16) []byte {
	var tiff bytes.Buffer
	be := binary.BigEndian
	tiff.WriteString("MM\x00\x2a")
	_ = binary.Write(&tiff, be, uint32(8))
	entry := func(tag, typ uint16, count, value uint32) {
		_ = binary.Write(&tiff, be, tag)
		_ = binary.Write(&tiff, be, typ)
		_ = binary.Write(&tiff, be, count)
		_ = binary.Write(&tiff, be, value)
	}
	_ = binary.Write(&tiff, be, uint16(2))
	entry(0x0112, 3, 1, uint32(orientation)<<16)
	entry(0x8825, 4, 1, 8+2+2*12+4)
	_ = binary.Write(&tiff, be, uint32(0))
	_ = binary.Write(&tiff, be, uint16(1))
	entry(0x0001, 2, 2, uint32('N')<<24)
	_ = binary.Write(&tiff, be, uint32(0))

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	segment := []byte{0xFF, 0xE1, 0, 0}
	be.PutUint16(segment[2:], uint16(len(payload)+2))
	return append(segment, payload...)
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.NRGBA{200, 10, 10, 128})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// pngBomb is a small PNG whose header claims 50,000 by 50,000 pixels.
func pngBomb(t *testing.T) []byte {
	t.Helper()
	data := testPNG(t, 1, 1)
	binary.BigEndian.PutUint32(data[16:], 50000)
	binary.BigEndian.PutUint32(data[20:], 50000)
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	return data
}

type photoEnv struct {
	*env
	ana, joe authToken
	reviews  string
}

func newPhotoEnv(t *testing.T, opts publicapi.Options) photoEnv {
	t.Helper()
	e := newEnv(t, opts)
	ana := e.signUp("ana@example.com", "Ana")
	joe := e.signUp("joe@example.com", "Joe")
	restaurant := e.createRestaurant(ana.Token, `{"name":"Luigi's"}`)["id"].(string)
	review := e.createReview(ana.Token, restaurant, reviewBody)["id"].(string)
	return photoEnv{env: e, ana: ana, joe: joe, reviews: "/v1/restaurants/" + restaurant + "/reviews/" + review}
}

func (p photoEnv) upload(token string, data []byte, contentType string, status int) response {
	p.t.Helper()
	return p.expect(request{method: "POST", path: p.reviews + "/photos", body: string(data), contentType: contentType, token: token}, status)
}

func (p photoEnv) addPhoto() map[string]any {
	p.t.Helper()
	return p.upload(p.ana.Token, testJPEG(p.t, 40, 20, 0), "image/jpeg", http.StatusCreated).object(p.t)
}

func urlOf(photo map[string]any, variant string) string {
	return photo["urls"].(map[string]any)[variant].(string)
}

func TestAddReviewPhoto(t *testing.T) {
	p := newPhotoEnv(t, testOptions)
	res := p.upload(p.ana.Token, testJPEG(t, 40, 20, 6), "image/jpeg", http.StatusCreated)
	photo := res.object(t)
	id := photo["id"].(string)
	if loc := res.header.Get("Location"); loc != p.reviews+"/photos/"+id {
		t.Errorf("Location %q", loc)
	}
	// Orientation 6 turns the image by 90 degrees.
	if photo["width"] != 20.0 || photo["height"] != 40.0 || photo["alt"] != nil {
		t.Errorf("photo %v", photo)
	}
	if urlOf(photo, "full") != p.reviews+"/photos/"+id+"/full" || urlOf(photo, "thumb") != p.reviews+"/photos/"+id+"/thumb" {
		t.Errorf("urls %v", photo["urls"])
	}

	review := p.expect(request{method: "GET", path: p.reviews}, http.StatusOK).object(t)
	if photos := review["photos"].([]any); len(photos) != 1 || photos[0].(map[string]any)["id"] != id {
		t.Errorf("review photos %v", review["photos"])
	}

	for _, variant := range []string{"full", "thumb"} {
		file := p.expect(request{method: "GET", path: urlOf(photo, variant)}, http.StatusOK)
		for name, want := range map[string]string{
			"Content-Type":                 "image/jpeg",
			"Cache-Control":                "public, max-age=31536000, immutable",
			"Content-Security-Policy":      "default-src 'none'; sandbox",
			"X-Content-Type-Options":       "nosniff",
			"Cross-Origin-Resource-Policy": "same-origin",
		} {
			if got := file.header.Get(name); got != want {
				t.Errorf("%s %s: %q, want %q", variant, name, got, want)
			}
		}
		if bytes.Contains(file.body, []byte("Exif")) {
			t.Errorf("%s has EXIF data", variant)
		}
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(file.body))
		if err != nil || cfg.Width != 20 || cfg.Height != 40 {
			t.Errorf("%s: %+v, %v", variant, cfg, err)
		}
	}

	full := urlOf(photo, "full")
	etag := p.expect(request{method: "GET", path: full}, http.StatusOK).header.Get("ETag")
	if !strings.HasPrefix(etag, `"full_`) {
		t.Errorf("ETag %q", etag)
	}
	p.expect(request{method: "GET", path: full, header: map[string]string{"If-None-Match": etag}}, http.StatusNotModified)
	if r := p.expect(request{method: "GET", path: full, header: map[string]string{"Range": "bytes=0-9"}}, http.StatusPartialContent); len(r.body) != 10 {
		t.Errorf("range: %d bytes", len(r.body))
	}
	if r := p.expect(request{method: "HEAD", path: full}, http.StatusOK); len(r.body) != 0 {
		t.Errorf("HEAD has a body of %d bytes", len(r.body))
	}
	p.expect(request{method: "GET", path: p.reviews + "/photos/" + id + "/large"}, http.StatusBadRequest)

	// http.ServeContent sends its 416 as text, not as Problem Details.
	outside := p.do(request{method: "GET", path: full, header: map[string]string{"Range": "bytes=99999999-"}})
	if outside.status != http.StatusRequestedRangeNotSatisfiable || outside.header.Get("Cache-Control") != "" {
		t.Errorf("416: status %d, Cache-Control %q", outside.status, outside.header.Get("Cache-Control"))
	}
}

func TestAddReviewPhotoTypes(t *testing.T) {
	p := newPhotoEnv(t, testOptions)
	webp, err := base64.StdEncoding.DecodeString(webp1x1)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, contentType string
		data              []byte
	}{
		{"png with transparency", "image/png", testPNG(t, 30, 10)},
		{"webp", "image/webp", webp},
		{"large jpeg", "image/jpeg", testJPEG(t, 3000, 1000, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			photo := p.upload(p.ana.Token, tc.data, tc.contentType, http.StatusCreated).object(t)
			if w := photo["width"].(float64); w < 1 || w > 2048 {
				t.Errorf("width %v", w)
			}
			thumb := p.expect(request{method: "GET", path: urlOf(photo, "thumb")}, http.StatusOK)
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(thumb.body))
			if err != nil || max(cfg.Width, cfg.Height) > 480 {
				t.Errorf("thumb %+v, %v", cfg, err)
			}
		})
	}
}

func TestAddReviewPhotoErrors(t *testing.T) {
	p := newPhotoEnv(t, testOptions)
	jpg := testJPEG(t, 40, 20, 0)
	for _, tc := range []struct {
		name, path, contentType, token string
		data                           []byte
		status                         int
	}{
		{"no token", "", "image/jpeg", "", jpg, http.StatusUnauthorized},
		{"other user", "", "image/jpeg", p.joe.Token, jpg, http.StatusForbidden},
		{"unknown review", "/v1/restaurants/aaaaaaaaaaaaaaa/reviews/aaaaaaaaaaaaaaa/photos", "image/jpeg", p.ana.Token, jpg, http.StatusNotFound},
		{"gif", "", "image/gif", p.ana.Token, []byte("GIF89a"), http.StatusUnsupportedMediaType},
		{"svg", "", "image/svg+xml", p.ana.Token, []byte("<svg/>"), http.StatusUnsupportedMediaType},
		{"text as jpeg", "", "image/jpeg", p.ana.Token, []byte("not an image"), http.StatusUnprocessableEntity},
		{"png as jpeg", "", "image/jpeg", p.ana.Token, testPNG(t, 2, 2), http.StatusUnprocessableEntity},
		{"broken jpeg", "", "image/jpeg", p.ana.Token, jpg[:40], http.StatusUnprocessableEntity},
		{"too many pixels", "", "image/png", p.ana.Token, pngBomb(t), http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := cmp.Or(tc.path, p.reviews+"/photos")
			p.expect(request{method: "POST", path: path, body: string(tc.data), contentType: tc.contentType, token: tc.token}, tc.status)
		})
	}

	t.Run("too large", func(t *testing.T) {
		for _, withLength := range []bool{true, false} {
			body := io.MultiReader(bytes.NewReader(jpg), io.LimitReader(filler{}, 11<<20))
			r := httptest.NewRequest(http.MethodPost, p.reviews+"/photos", body)
			r.ContentLength = -1
			if withLength {
				r.ContentLength = int64(len(jpg)) + 11<<20
			}
			r.Header.Set("Content-Type", "image/jpeg")
			r.Header.Set("Authorization", "Bearer "+p.ana.Token)
			w := httptest.NewRecorder()
			p.h.ServeHTTP(w, r)
			if w.Code != http.StatusRequestEntityTooLarge || w.Header().Get("Content-Type") != "application/problem+json" {
				t.Errorf("length %v: %d %s", withLength, w.Code, w.Body)
			}
		}
	})

	t.Run("sixth photo", func(t *testing.T) {
		for range 5 {
			p.addPhoto()
		}
		p.upload(p.ana.Token, jpg, "image/jpeg", http.StatusConflict)
	})
}

func TestParallelUploadsAtTheLastPlace(t *testing.T) {
	p := newPhotoEnv(t, testOptions)
	for range 4 {
		p.addPhoto()
	}
	jpg := string(testJPEG(t, 40, 20, 0))
	var wg sync.WaitGroup
	statuses := make([]int, 2)
	for i := range statuses {
		wg.Go(func() {
			statuses[i] = p.do(request{method: "POST", path: p.reviews + "/photos", body: jpg, contentType: "image/jpeg", token: p.ana.Token}).status
		})
	}
	wg.Wait()
	created, conflicts := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		}
	}
	if created != 1 || conflicts != 1 {
		t.Errorf("statuses %v, want one 201 and one 409", statuses)
	}
}

func TestUpdateReviewPhoto(t *testing.T) {
	p := newPhotoEnv(t, testOptions)
	photo := p.addPhoto()
	path := p.reviews + "/photos/" + photo["id"].(string)

	got := p.expect(request{method: "PATCH", path: path, token: p.ana.Token, body: `{"alt":"A plate of pasta"}`}, http.StatusOK).object(t)
	if got["alt"] != "A plate of pasta" {
		t.Errorf("alt %v", got["alt"])
	}
	review := p.expect(request{method: "GET", path: p.reviews}, http.StatusOK).object(t)
	if alt := review["photos"].([]any)[0].(map[string]any)["alt"]; alt != "A plate of pasta" {
		t.Errorf("review photo alt %v", alt)
	}
	if got := p.expect(request{method: "PATCH", path: path, token: p.ana.Token, body: `{}`}, http.StatusOK).object(t); got["alt"] != "A plate of pasta" {
		t.Errorf("empty patch: alt %v", got["alt"])
	}
	if got := p.expect(request{method: "PATCH", path: path, token: p.ana.Token, body: `{"alt":null}`}, http.StatusOK).object(t); got["alt"] != nil {
		t.Errorf("null: alt %v", got["alt"])
	}

	res := p.expect(request{method: "PATCH", path: path, token: p.ana.Token, body: `{"alt":"Pasta​"}`}, http.StatusUnprocessableEntity)
	if props := errorProperties(t, res); len(props) != 1 || props[0] != "alt" {
		t.Errorf("422 properties %v", props)
	}
	p.expect(request{method: "PATCH", path: path, token: p.ana.Token, body: `{"width":10}`}, http.StatusBadRequest)
	p.expect(request{method: "PATCH", path: path, token: p.joe.Token, body: `{"alt":"Mine"}`}, http.StatusForbidden)
	p.expect(request{method: "PATCH", path: p.reviews + "/photos/aaaaaaaaaaaaaaa", token: p.ana.Token, body: `{}`}, http.StatusNotFound)
}

func TestDeleteReviewPhoto(t *testing.T) {
	p := newPhotoEnv(t, testOptions)
	photo := p.addPhoto()
	path := p.reviews + "/photos/" + photo["id"].(string)

	p.expect(request{method: "DELETE", path: path, token: p.joe.Token}, http.StatusForbidden)
	p.expect(request{method: "DELETE", path: path, token: p.ana.Token}, http.StatusNoContent)
	p.expect(request{method: "GET", path: urlOf(photo, "full")}, http.StatusNotFound)
	p.expect(request{method: "DELETE", path: path, token: p.ana.Token}, http.StatusNotFound)
	review := p.expect(request{method: "GET", path: p.reviews}, http.StatusOK).object(t)
	if photos := review["photos"].([]any); len(photos) != 0 {
		t.Errorf("photos %v", photos)
	}
	p.waitForNoFiles(photo["id"].(string))
}

// waitForNoFiles waits for the background file delete of PocketBase.
func (p photoEnv) waitForNoFiles(photoID string) {
	p.t.Helper()
	photos, err := p.app.FindCollectionByNameOrId("review_photos")
	if err != nil {
		p.t.Fatal(err)
	}
	dir := filepath.Join(p.app.DataDir(), "storage", photos.Id, photoID)
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return
		}
	}
	p.t.Errorf("the files of photo %s are still in %s", photoID, dir)
}

func TestPhotosOfDeletedReviewsAndUsers(t *testing.T) {
	t.Run("review", func(t *testing.T) {
		p := newPhotoEnv(t, testOptions)
		photo := p.addPhoto()
		p.expect(request{method: "DELETE", path: p.reviews, token: p.ana.Token}, http.StatusNoContent)
		if _, err := p.app.FindRecordById("review_photos", photo["id"].(string)); err == nil {
			t.Error("the photo record stays")
		}
		p.waitForNoFiles(photo["id"].(string))
	})
	t.Run("user", func(t *testing.T) {
		p := newPhotoEnv(t, testOptions)
		photo := p.addPhoto()
		user, err := p.app.FindRecordById("users", p.ana.User.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.app.Delete(user); err != nil {
			t.Fatal(err)
		}
		if _, err := p.app.FindRecordById("review_photos", photo["id"].(string)); err == nil {
			t.Error("the photo record stays")
		}
		p.waitForNoFiles(photo["id"].(string))
	})
}

func TestPocketBaseDoesNotServePhotoFiles(t *testing.T) {
	p := newPhotoEnv(t, testOptions)
	id := p.addPhoto()["id"].(string)
	record, err := p.app.FindRecordById("review_photos", id)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"full", "thumb"} {
		path := fmt.Sprintf("/api/files/review_photos/%s/%s", id, record.GetString(field))
		if res := p.do(request{method: "GET", path: path, token: p.ana.Token}); res.status == http.StatusOK {
			t.Errorf("%s: 200", path)
		}
	}
}

func TestPhotoUploadRateLimit(t *testing.T) {
	opts := testOptions
	opts.PhotoUploadLimit = 1
	p := newPhotoEnv(t, opts)
	p.addPhoto()
	res := p.upload(p.ana.Token, testJPEG(t, 4, 4, 0), "image/jpeg", http.StatusTooManyRequests)
	if res.header.Get("Retry-After") == "" {
		t.Error("no Retry-After")
	}
	// Each user has an own count.
	other := p.createReview(p.joe.Token, strings.Split(p.reviews, "/")[3], reviewBody)["id"].(string)
	p.expect(request{method: "POST", path: "/v1/restaurants/" + strings.Split(p.reviews, "/")[3] + "/reviews/" + other + "/photos",
		body: string(testJPEG(t, 4, 4, 0)), contentType: "image/jpeg", token: p.joe.Token}, http.StatusCreated)
}
