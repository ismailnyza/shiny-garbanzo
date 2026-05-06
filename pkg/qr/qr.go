package qr

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateToken creates a cryptographically random hex string for QR codes.
// n is the number of random bytes (default 32).
func GenerateToken(n ...int) (string, error) {
	size := 32
	if len(n) > 0 && n[0] > 0 {
		size = n[0]
	}

	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return hex.EncodeToString(b), nil
}
