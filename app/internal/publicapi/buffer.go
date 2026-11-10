package publicapi

import (
	"bytes"
	"maps"
	"net/http"
)

// bufferedWriter keeps the status, the headers, and the body of a response.
// Only Flush writes them to the underlying writer. The response can then
// change after the handler, for example to 500 for an invalid response.
type bufferedWriter struct {
	w      http.ResponseWriter
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedWriter(w http.ResponseWriter) *bufferedWriter {
	return &bufferedWriter{w: w, header: http.Header{}}
}

func (b *bufferedWriter) Header() http.Header {
	return b.header
}

// WriteHeader keeps the first status, like an http.ResponseWriter.
func (b *bufferedWriter) WriteHeader(status int) {
	if b.status == 0 {
		b.status = status
	}
}

func (b *bufferedWriter) Write(p []byte) (int, error) {
	b.WriteHeader(http.StatusOK)
	return b.body.Write(p)
}

// Status returns 200 for a handler that did not write a status.
func (b *bufferedWriter) Status() int {
	if b.status == 0 {
		return http.StatusOK
	}
	return b.status
}

// Flush writes the response to the underlying writer. The headers of the
// underlying writer stay, such as the CORS headers of PocketBase. A buffered
// header replaces a header with the same name.
func (b *bufferedWriter) Flush() error {
	maps.Copy(b.w.Header(), b.header)
	b.w.WriteHeader(b.Status())
	_, err := b.body.WriteTo(b.w)
	return err
}

// headerCheckWriter checks the status and the headers of a response before
// it writes them. The body then goes to the client without a buffer. After
// a failed check, the writer drops the headers and the body.
type headerCheckWriter struct {
	http.ResponseWriter
	check func(status int, header http.Header) error
	wrote bool
	err   error
}

func (h *headerCheckWriter) WriteHeader(status int) {
	if h.wrote {
		return
	}
	h.wrote = true
	if h.err = h.check(status, h.Header()); h.err != nil {
		clear(h.Header())
		return
	}
	h.ResponseWriter.WriteHeader(status)
}

func (h *headerCheckWriter) Write(p []byte) (int, error) {
	h.WriteHeader(http.StatusOK)
	if h.err != nil {
		return len(p), nil
	}
	return h.ResponseWriter.Write(p)
}
