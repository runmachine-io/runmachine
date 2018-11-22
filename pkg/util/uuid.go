package util

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	// Note that we lowercase and remove hyphens before attempting to match
	// this regex, which is why this is nice and simple
	regexUuid = regexp.MustCompile("^[0-9a-f]{32}$")
)

// NewNormalizedUuid returns a newly-generated, stripped to 32 hex digits UUID4
func NewNormalizedUuid() string {
	return NormalizeUuid(uuid.New().String())
}

// NormalizeUuid simple lowecases and removes all non-alphanumeric characters
// from the supplied string
func NormalizeUuid(subject string) string {
	return strings.ToLower(strings.Replace(subject, "-", "", -1))
}

// IsUuidLike return true if the supplied string looks to be a UUID, false
// otherwise
func IsUuidLike(subject string) bool {
	switch len(subject) {
	case 32:
		return regexUuid.MatchString(strings.ToLower(subject))
	case 36:
		return regexUuid.MatchString(strings.Replace(strings.ToLower(subject), "-", "", -1))
	default:
		return false
	}
}
