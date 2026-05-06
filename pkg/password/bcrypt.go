package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Hash generates a bcrypt hash of the password.
func Hash(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// Compare checks a plaintext password against a bcrypt hash.
func Compare(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
