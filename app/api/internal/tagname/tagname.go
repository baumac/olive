// Package tagname changes free text to the stored name of a tag. See
// decision 00011.
package tagname

import (
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Of changes text to a tag name. The result can be empty or longer than 63
// characters. The Tag schema of the spec refuses both.
func Of(text string) string {
	words, _ := split(text)
	for i, w := range words {
		words[i] = finalSigma(w)
	}
	return strings.Join(words, "-")
}

// SearchKey changes a search to the key of the start of a tag name. The last
// word of the search can be unfinished, so it keeps its σ. A separator at the
// end gives a hyphen, so the search matches only whole words.
func SearchKey(text string) string {
	words, open := split(text)
	if len(words) == 0 {
		return ""
	}
	for i, w := range words {
		if i < len(words)-1 || !open {
			words[i] = finalSigma(w)
		}
	}
	key := Key(strings.Join(words, "-"))
	if !open {
		key += "-"
	}
	return key
}

// split gives the words of the tag name of text. open is true when the text
// ends in a letter, a mark, or a digit.
func split(text string) (words []string, open bool) {
	var b strings.Builder
	inWord, gap := false, false
	for _, r := range strings.ToLower(norm.NFKC.String(text)) {
		switch {
		case ignorable(r):
		case unicode.In(r, unicode.Ll, unicode.Lm, unicode.Lo, unicode.Nd):
			if gap && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			inWord, gap = true, false
		case unicode.In(r, unicode.Mn, unicode.Mc):
			if inWord {
				b.WriteRune(r)
			}
		default:
			inWord, gap = false, true
		}
	}
	if b.Len() == 0 {
		return nil, false
	}
	return strings.Split(norm.NFC.String(b.String()), "-"), !gap
}

// finalSigma changes a σ at the end of a word after a letter to ς. The
// lowercase of Σ does not know its place in the word. The UI has the same
// rule.
func finalSigma(word string) string {
	rs := []rune(word)
	n := len(rs)
	if n < 2 || rs[n-1] != 'σ' || !unicode.IsLetter(rs[n-2]) {
		return word
	}
	rs[n-1] = 'ς'
	return string(rs)
}

// List changes each text with Of. It drops the empty names and the repeats.
// The first copy of a repeat keeps its place.
func List(texts []string) []string {
	out := make([]string, 0, len(texts))
	for _, t := range texts {
		if name := Of(t); name != "" && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

// Key is the name without case and accents. The search of listTags compares
// it.
func Key(name string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	key, _, err := transform.String(t, strings.ToLower(name))
	if err != nil {
		return strings.ToLower(name)
	}
	return key
}

// ignorable has the property Default_Ignorable_Code_Point. Go has no table
// of it. Unicode derives it from these tables.
func ignorable(r rune) bool {
	switch {
	case unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector):
		return true
	case !unicode.Is(unicode.Cf, r):
		return false
	case unicode.In(r, unicode.White_Space, unicode.Prepended_Concatenation_Mark):
		return false
	case r >= 0xfff9 && r <= 0xfffb, r >= 0x13430 && r <= 0x1343f:
		return false
	}
	return true
}
