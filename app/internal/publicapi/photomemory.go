package publicapi

import (
	"errors"
	"io"
	"net/http"
	"sync/atomic"
)

// defaultPhotoMemoryLimit holds 3 to 6 uploads of 10 MB. A buffer can have
// up to twice the bytes of its body.
const defaultPhotoMemoryLimit = 64 << 20

// photoMemory counts the bytes that photo uploads hold in memory. The
// decodes semaphore limits the memory of the decodes.
type photoMemory struct {
	limit int64
	used  atomic.Int64
}

// take returns false and adds nothing if n bytes go over the limit.
func (m *photoMemory) take(n int64) bool {
	for {
		used := m.used.Load()
		if used+n > m.limit {
			return false
		}
		if m.used.CompareAndSwap(used, used+n) {
			return true
		}
	}
}

func (m *photoMemory) give(n int64) {
	m.used.Add(-n)
}

// resize changes a hold of from bytes to to bytes. It can go over the limit.
// The bytes are already in memory.
func (m *photoMemory) resize(from, to int64) {
	m.used.Add(to - from)
}

// errPhotoMemoryFull tells that the uploads in memory reached the limit.
var errPhotoMemoryFull = errors.New("the photo uploads in memory reached the limit")

// read reads r and takes the memory of the buffer as it grows. A slow client
// then holds only the bytes that it sent. The caller gives back held.
func (m *photoMemory) read(r io.Reader) (data []byte, held int64, err error) {
	const chunk = 64 << 10
	for {
		if len(data) == cap(data) {
			// One byte over the limit lets a reader without a limit get 413.
			size := min(max(2*cap(data), chunk), maxUploadBytes+1)
			if size == cap(data) {
				m.give(held)
				return nil, 0, &http.MaxBytesError{Limit: maxUploadBytes}
			}
			if !m.take(int64(size - cap(data))) {
				m.give(held)
				return nil, 0, errPhotoMemoryFull
			}
			held += int64(size - cap(data))
			grown := make([]byte, len(data), size)
			copy(grown, data)
			data = grown
		}
		n, err := r.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err == io.EOF {
			return data, held, nil
		}
		if err != nil {
			m.give(held)
			return nil, 0, err
		}
	}
}
