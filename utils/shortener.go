package utils

import (
	crand "crypto/rand"
	"encoding/base64"
	"math/rand"
	"time"
)

const (
	// DefaultLength is the standard length for generated short codes.
	DefaultLength = 7
	// charset is the set of characters to use for generating short codes.
	// Using base64 URL safe characters.
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()),
)

// GenerateShortCode creates a random string of a specified length using the defined charset.
// This uses math/rand for speed but is less cryptographically secure. Fine for non-sensitive codes.
func GenerateShortCode(length int) string {
	if length <= 0 {
		length = DefaultLength
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateSecureShortCode creates a cryptographically secure random string.
// It's slightly slower but better if collision resistance or unpredictability
// is critical beyond simple uniqueness checks.
func GenerateSecureShortCode(length int) (string, error) {
	if length <= 0 {
		length = DefaultLength
	}
	// Calculate the number of random bytes needed. Each byte gives 8 bits of randomness.
	// Base64 encodes 6 bits per character. So, we need length * 6 bits.
	numBytes := (length*6 + 7) / 8 // +7 / 8 is equivalent to ceil(length*6 / 8)
	randomBytes := make([]byte, numBytes)
	_, err := crand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode random bytes to Base64 URL Safe string
	code := base64.URLEncoding.EncodeToString(randomBytes)

	// Trim padding and return the required length
	return code[:length], nil
}
