package util

import "strings"

// NormalizeUuid simple lowecases and removes all non-alphanumeric characters
// from the supplied string
func NormalizeUuid(subject string) string {
	return strings.ToLower(strings.Replace(subject, "-", "", -1))
}
