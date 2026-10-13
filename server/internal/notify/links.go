package notify

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// linkCodeAlphabet is the full alphanumeric range. Codes are tapped, never
// typed, so there is no reason to avoid look-alike characters, and the wider
// range keeps codes short.
const linkCodeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// LinkCodeLength is the length of a tracked link code. 62^8 codes make one
// guessed by enumeration vanishingly unlikely, and a guess only reveals a
// resort's public mountain report page anyway.
const LinkCodeLength = 8

// NewLinkCode returns a random code identifying one alert's tracked link.
func NewLinkCode() (string, error) {
	code := make([]byte, LinkCodeLength)
	limit := big.NewInt(int64(len(linkCodeAlphabet)))

	for i := range code {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", fmt.Errorf("generating link code: %w", err)
		}

		code[i] = linkCodeAlphabet[n.Int64()]
	}

	return string(code), nil
}

// IsLinkCode reports whether s has the shape of a code from NewLinkCode, so
// junk can be turned away before it reaches the database.
func IsLinkCode(s string) bool {
	if len(s) != LinkCodeLength {
		return false
	}

	for i := range len(s) {
		if !strings.ContainsRune(linkCodeAlphabet, rune(s[i])) {
			return false
		}
	}

	return true
}
