package publicapi

import (
	"container/list"
	"math"
	"net/netip"
	"strconv"
	"sync"
	"time"
)

// rateLimiter counts requests per key in fixed time windows.
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	now    func() time.Time
	keys   map[string]*list.Element
	// order has the windows from the oldest start to the newest start.
	order *list.List
}

type rateWindow struct {
	key   string
	start time.Time
	count int
}

// maxWindows is the most windows that the limiter keeps. A new key then
// removes the oldest window, also if that window did not end.
const maxWindows = 10000

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, now: time.Now, keys: map[string]*list.Element{}, order: list.New()}
}

// allow returns 0 and the start of the window that counts the request if the
// request can continue. Otherwise it returns the time until the window ends.
func (l *rateLimiter) allow(key string) (time.Duration, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for e := l.order.Front(); e != nil && now.Sub(e.Value.(*rateWindow).start) >= l.window; e = l.order.Front() {
		l.remove(e)
	}

	if e, ok := l.keys[key]; ok {
		w := e.Value.(*rateWindow)
		if w.count >= l.limit {
			return w.start.Add(l.window).Sub(now), time.Time{}
		}
		w.count++
		return 0, w.start
	}
	if l.order.Len() >= maxWindows {
		l.remove(l.order.Front())
	}
	l.keys[key] = l.order.PushBack(&rateWindow{key: key, start: now, count: 1})
	return 0, now
}

// giveBack removes one request of key from the window that started at start.
// A later window of key stays the same.
func (l *rateLimiter) giveBack(key string, start time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.keys[key]; ok {
		if w := e.Value.(*rateWindow); w.start.Equal(start) && w.count > 0 {
			w.count--
		}
	}
}

func (l *rateLimiter) remove(e *list.Element) {
	delete(l.keys, l.order.Remove(e).(*rateWindow).key)
}

// clientKey changes an IPv6 address to its /64 network. One host often gets
// a full /64 network.
func clientKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	if addr = addr.Unmap(); addr.Is4() {
		return addr.String()
	}
	prefix, err := addr.Prefix(64)
	if err != nil {
		return ip
	}
	return prefix.String()
}

// retryAfter changes wait to whole seconds for the Retry-After header. The
// minimum is 1.
func retryAfter(wait time.Duration) string {
	return strconv.Itoa(max(1, int(math.Ceil(wait.Seconds()))))
}
