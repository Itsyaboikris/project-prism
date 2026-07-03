package apikey

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const prefix = "prism_"

// Generate returns a new secure API key in the form prism_<32 random bytes, base64url>.
func Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
