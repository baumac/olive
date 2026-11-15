package publicapi

import (
	"strings"
	"testing"
)

func TestEmailHTMLLinksOnlyAURLParagraph(t *testing.T) {
	got := emailHTML([]string{
		"https://evil.example/login (x@example.com) confirmed the email address.",
		"https://reviews.example.com/confirm-email#token=a.b",
		"Tom & <b>Jerry</b>",
	})
	want := "<p>https://evil.example/login (x@example.com) confirmed the email address.</p>\n" +
		`<p><a href="https://reviews.example.com/confirm-email#token=a.b">https://reviews.example.com/confirm-email#token=a.b</a></p>` + "\n" +
		"<p>Tom &amp; &lt;b&gt;Jerry&lt;/b&gt;</p>\n"
	if got != want {
		t.Errorf("HTML:\n%s\nwant:\n%s", got, want)
	}
	if strings.Count(got, "<a ") != 1 {
		t.Errorf("%d links, want 1", strings.Count(got, "<a "))
	}
}
