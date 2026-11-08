package publicapi

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
)

type fieldErrors []reviewsapi.ValidationError

func (e *fieldErrors) add(property, message string) {
	*e = append(*e, reviewsapi.ValidationError{Property: property, Message: message})
}

const asciiSpace = " \t\n\v\f\r"

var (
	recordIDPattern  = regexp.MustCompile(`^[a-z0-9]{15}$`)
	tagPattern       = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	timestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}Z$`)
)

// checkName checks the Name and Title schemas.
func checkName(errs *fieldErrors, property, v string) {
	if n := utf8.RuneCountInString(v); n < 1 || n > 255 {
		errs.add(property, "must have 1 to 255 characters")
	} else if strings.Trim(v, asciiSpace) == "" {
		errs.add(property, "must have a character that is not whitespace")
	}
}

func checkLongText(errs *fieldErrors, property, v string) {
	if utf8.RuneCountInString(v) > 10000 {
		errs.add(property, "must have at most 10000 characters")
	}
}

func checkTags(errs *fieldErrors, property string, tags []string) {
	if len(tags) > 20 {
		errs.add(property, "must have at most 20 items")
	}
	seen := make(map[string]bool, len(tags))
	for i, tag := range tags {
		item := fmt.Sprintf("%s[%d]", property, i)
		switch {
		case !validTag(tag):
			errs.add(item, "must have 1 to 63 lowercase letters, digits, and single hyphens between them")
		case seen[tag]:
			errs.add(item, "must be unique")
		}
		seen[tag] = true
	}
}

func validTag(tag string) bool {
	return len(tag) <= 63 && tagPattern.MatchString(tag)
}

func checkScore(errs *fieldErrors, property string, v, maxValue int) {
	if v < 1 || v > maxValue {
		errs.add(property, fmt.Sprintf("must be between 1 and %d", maxValue))
	}
}

func checkPassword(errs *fieldErrors, property, v string) {
	if n := utf8.RuneCountInString(v); n < 8 || n > 71 {
		errs.add(property, "must have 8 to 71 characters")
	} else if len(v) > 72 {
		errs.add(property, "must have at most 72 bytes in UTF-8")
	}
}

func checkEmail(errs *fieldErrors, property, v string) {
	if utf8.RuneCountInString(v) > 255 {
		errs.add(property, "must have at most 255 characters")
	}
}
