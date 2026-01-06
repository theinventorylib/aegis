package core

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// This file exports private functions for testing purposes.
// These exports are only available to the core package tests.

// ulidMutex protects concurrent ULID generation for testing
var ulidMutex sync.Mutex

// GenerateULID generates a ULID for testing with proper synchronization
func GenerateULID() string {
	ulidMutex.Lock()
	defer ulidMutex.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), defaultIDConfig.entropy).String()
}

// GenerateUUID generates a UUID for testing
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateSecureToken generates a secure token for testing
func GenerateSecureToken() string {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// GenerateRandomSuffix generates a random suffix for testing
func GenerateRandomSuffix(length int) string {
	if length <= 0 {
		return ""
	}
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		// Use crypto/rand for proper randomness
		b := make([]byte, 1)
		_, _ = rand.Read(b)
		result[i] = charset[int(b[0])%len(charset)]
	}
	return string(result)
}

// ConstantTimeCompare is exported for testing
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
