package guided

import "strings"

// NormalizeText is the duplicate key of suggested text: case-insensitive,
// whitespace collapsed.
func NormalizeText(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
