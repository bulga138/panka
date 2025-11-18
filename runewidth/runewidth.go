package runewidth

import (
	"unicode"
	"unicode/utf8"
)

func RuneWidth(r rune) int {
	// Invalid rune
	if !utf8.ValidRune(r) {
		return 0
	}

	// Explicitly zero-width characters
	if isExplicitZeroWidth(r) {
		return 0
	}

	// Unicode categories that are typically zero-width
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}

	// Wide characters (simplified CJK detection)
	if isWideCharacter(r) {
		return 2
	}

	// Default to narrow
	return 1
}

func StringWidth(s string) int {
	width := 0
	for _, r := range s {
		width += RuneWidth(r)
	}
	return width
}

func isExplicitZeroWidth(r rune) bool {
	switch r {
	case '\u202F', '\u200B', '\u200C', '\u200D', '\uFEFF',
		'\u2060', '\u200E', '\u200F', '\u2028', '\u2029':
		return true
	}
	return false
}

func isWideCharacter(r rune) bool {
	// Basic CJK ranges - extend as needed
	return (r >= 0x1100 && r <= 0x115F) || // Hangul Jamo
		(r >= 0x2329 && r <= 0x232A) || // Angle brackets
		(r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) ||
		(r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility
		(r >= 0xFE10 && r <= 0xFE19) || // Vertical forms
		(r >= 0xFE30 && r <= 0xFE6F) || // CJK Compatibility Forms
		(r >= 0xFF00 && r <= 0xFFEF) // Fullwidth forms
}
