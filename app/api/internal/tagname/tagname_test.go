package tagname

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestOf(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../../../specs/api/tag-names.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Text, Tag string }
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if got := Of(c.Text); got != c.Tag {
			t.Errorf("Of(%+q) = %+q, want %+q", c.Text, got, c.Tag)
		}
		if got := Of(c.Tag); got != c.Tag {
			t.Errorf("Of(%+q) = %+q, want the same tag", c.Tag, got)
		}
	}
}

func TestOfLongText(t *testing.T) {
	t.Parallel()
	if got := Of(strings.Repeat("a ", 40)); len([]rune(got)) != 79 {
		t.Errorf("Of cut the text to %q", got)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	got := List([]string{"Pizza", "pasta", "!!!", "pizza", "PASTA", "wood fired"})
	if want := []string{"pizza", "pasta", "wood-fired"}; !slices.Equal(got, want) {
		t.Errorf("List = %v, want %v", got, want)
	}
	if got := List(nil); got == nil || len(got) != 0 {
		t.Errorf("List(nil) = %#v, want an empty list", got)
	}
}

func TestKey(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{"café": "cafe", "crème-brûlée": "creme-brulee", "ラーメン": "ラーメン", "ψάρι": "ψαρι"} {
		if got := Key(name); got != want {
			t.Errorf("Key(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestSearchKey(t *testing.T) {
	t.Parallel()
	for search, want := range map[string]string{
		"Wood":      "wood",
		"wood ":     "wood-",
		"Wood_F":    "wood-f",
		"Café":      "cafe",
		"οδοσ":      "οδοσ",
		"οδοσ ":     "οδος-",
		"ΟΔΟΣ ΜΑΣ":  "οδος-μασ",
		"!!!":       "",
		"":          "",
		"  pasta  ": "pasta-",
	} {
		if got := SearchKey(search); got != want {
			t.Errorf("SearchKey(%q) = %q, want %q", search, got, want)
		}
	}
}
