package service

import (
	"unicode"
)

// IsStrongPassword adalah fungsi murni untuk memeriksa kekuatan password.
// Aturan: Minimal 8 karakter, mengandung huruf besar, huruf kecil, dan angka.
func IsStrongPassword(p string) bool {
	if len(p) < 8 {
		return false
	}

	var hasUpper, hasLower, hasDigit bool

	for _, char := range p {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	return hasUpper && hasLower && hasDigit
}
