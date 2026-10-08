package common

import (
	"strings"
	"unicode/utf8"
)

// MaskUsername keeps enough shape for a public entry to remain recognizable
// without exposing the account identifier.
func MaskUsername(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return "***"
	}
	runes := []rune(username)
	switch utf8.RuneCountInString(username) {
	case 1:
		return "*"
	case 2:
		return string(runes[0]) + "*"
	default:
		maskedCount := len(runes) - 2
		if maskedCount > 3 {
			maskedCount = 3
		}
		return string(runes[0]) + strings.Repeat("*", maskedCount) + string(runes[len(runes)-1])
	}
}
