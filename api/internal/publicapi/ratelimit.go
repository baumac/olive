package publicapi

import (
	"math"
	"strconv"
	"sync"
	"time"
)

// rateLimiter counts requests per key in fixed time windows.
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	windows map[string]*rateWindow
	swept   time.Time
}

type rateWindow struct {
	start time.Time
	count int
}

// maxWindows is the map size at which allow removes the expired windows.
// allow does this at most one time per window. Many clients then do not
// make each request walk the full map.
const maxWindows = 10000

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, now: time.Now, windows: map[string]*rateWindow{}}
}

// allow returns 0 if the request can continue. Otherwise it returns the
// time until the window of key ends.
func (l *rateLimiter) allow(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.windows) >= maxWindows && now.Sub(l.swept) >= l.window {
		l.swept = now
		for k, w := range l.windows {
			if now.Sub(w.start) >= l.window {
				delete(l.windows, k)
			}
		}
	}

	w, ok := l.windows[key]
	if !ok || now.Sub(w.start) >= l.window {
		l.windows[key] = &rateWindow{start: now, count: 1}
		return 0
	}
	if w.count >= l.limit {
		return w.start.Add(l.window).Sub(now)
	}
	w.count++
	return 0
}

// retryAfter changes wait to whole seconds for the Retry-After header. The
// minimum is 1.
func retryAfter(wait time.Duration) string {
	return strconv.Itoa(max(1, int(math.Ceil(wait.Seconds()))))
}
