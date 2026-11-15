package publicapi

import (
	"strconv"
	"testing"
	"time"
)

func TestRateLimiterWindows(t *testing.T) {
	now := time.Unix(0, 0)
	l := newRateLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	for i := range 2 {
		if wait, _ := l.allow("a"); wait != 0 {
			t.Fatalf("request %d waits %v", i, wait)
		}
	}
	now = now.Add(20 * time.Second)
	if wait, _ := l.allow("a"); wait != 40*time.Second {
		t.Errorf("third request waits %v, want 40s", wait)
	}
	if wait, _ := l.allow("b"); wait != 0 {
		t.Errorf("another key waits %v", wait)
	}
	now = now.Add(40 * time.Second)
	if wait, _ := l.allow("a"); wait != 0 {
		t.Errorf("request after the window waits %v", wait)
	}
	if len(l.keys) != 2 || l.order.Len() != 2 {
		t.Errorf("%d keys and %d windows, want 2 and 2", len(l.keys), l.order.Len())
	}
}

func TestRateLimiterWithoutCap(t *testing.T) {
	l := newRateLimiter(1, time.Minute)
	l.maxWindows = 0
	l.allow("victim")
	for i := range maxWindows + 5 {
		l.allow(strconv.Itoa(i))
	}
	if wait, _ := l.allow("victim"); wait == 0 {
		t.Error("other keys removed the window of victim")
	}
}

func TestRateLimiterGiveBack(t *testing.T) {
	now := time.Unix(0, 0)
	l := newRateLimiter(1, time.Minute)
	l.now = func() time.Time { return now }

	_, start := l.allow("a")
	l.giveBack("a", start)
	if wait, _ := l.allow("a"); wait != 0 {
		t.Fatalf("request after give back waits %v", wait)
	}
	// A give back from the old window does not change the new window.
	now = now.Add(time.Minute)
	l.allow("a")
	l.giveBack("a", start)
	if wait, _ := l.allow("a"); wait == 0 {
		t.Error("the old give back changed the new window")
	}
}

func TestRateLimiterKeepsAtMostMaxWindows(t *testing.T) {
	now := time.Unix(0, 0)
	l := newRateLimiter(1, time.Minute)
	l.now = func() time.Time { return now }

	for i := range maxWindows + 5 {
		l.allow(strconv.Itoa(i))
	}
	if len(l.keys) != maxWindows || l.order.Len() != maxWindows {
		t.Fatalf("%d keys and %d windows, want %d", len(l.keys), l.order.Len(), maxWindows)
	}
	if _, ok := l.keys["0"]; ok {
		t.Error("the oldest window stays")
	}
	if wait, _ := l.allow(strconv.Itoa(maxWindows + 4)); wait == 0 {
		t.Error("the newest window was removed")
	}
}

func TestClientKey(t *testing.T) {
	for ip, want := range map[string]string{
		"192.0.2.1":            "192.0.2.1",
		"::ffff:192.0.2.1":     "192.0.2.1",
		"2001:db8:1:2:3:4:5:6": "2001:db8:1:2::/64",
		"2001:db8:1:2:ffff::1": "2001:db8:1:2::/64",
		"fe80::1%eth0":         "fe80::/64",
		"not an address":       "not an address",
		"2001:db8:1:3:3:4:5:6": "2001:db8:1:3::/64",
	} {
		if got := clientKey(ip); got != want {
			t.Errorf("clientKey(%q) = %q, want %q", ip, got, want)
		}
	}
}
