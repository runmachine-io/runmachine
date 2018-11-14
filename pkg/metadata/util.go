package metadata

import "strings"

// normalizeUuid simple lowecases and removes all non-alphanumeric characters
// from the supplied string
func normalizeUuid(subject string) string {
	return strings.ToLower(strings.Replace(subject, "-", "", -1))
}
