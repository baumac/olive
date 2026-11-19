package publicapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestPhotoMemoryRead(t *testing.T) {
	t.Parallel()
	m := &photoMemory{limit: 1 << 20}
	data, held, err := m.read(bytes.NewReader(make([]byte, 300<<10)))
	if err != nil || len(data) != 300<<10 {
		t.Fatalf("%d bytes, error %v", len(data), err)
	}
	if held != int64(cap(data)) || m.used.Load() != held {
		t.Errorf("held %d, used %d, capacity %d", held, m.used.Load(), cap(data))
	}
	m.give(held)

	if _, _, err := m.read(bytes.NewReader(make([]byte, 2<<20))); !errors.Is(err, errPhotoMemoryFull) {
		t.Errorf("error %v", err)
	}
	if m.used.Load() != 0 {
		t.Errorf("used %d after a full read", m.used.Load())
	}
}

func TestPhotoMemoryReadTooLarge(t *testing.T) {
	t.Parallel()
	m := &photoMemory{limit: 64 << 20}
	var tooLarge *http.MaxBytesError
	if _, _, err := m.read(io.LimitReader(zeros{}, maxUploadBytes+10)); !errors.As(err, &tooLarge) {
		t.Errorf("error %v", err)
	}
	if m.used.Load() != 0 {
		t.Errorf("used %d", m.used.Load())
	}
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

// An upload that waits for a decode stops with its request.
func TestReadPhotoGivesBackMemoryOnCancel(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	for range maxParallelDecodes {
		s.decodes <- struct{}{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.readPhoto(ctx, bytes.NewReader([]byte("image")), "image/png"); err == nil {
		t.Fatal("no error")
	}
	if s.photoMemory.used.Load() != 0 {
		t.Errorf("used %d", s.photoMemory.used.Load())
	}
}
