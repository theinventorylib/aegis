package core

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	// Default Argon2id parameters (recommended by OWASP)
	defaultTime      = 1
	defaultMemory    = 64 * 1024 // 64 MB
	defaultThreads   = 4
	defaultKeyLength = 32
	saltLength       = 16
)

// HashPassword hashes a password using Argon2id
func HashPassword(password string, time, memory uint32, threads uint8, keyLen uint32) (string, error) {
	if time == 0 {
		time = defaultTime
	}
	if memory == 0 {
		memory = defaultMemory
	}
	if threads == 0 {
		threads = defaultThreads
	}
	if keyLen == 0 {
		keyLen = defaultKeyLength
	}

	// Generate a random salt
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate the hash
	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	// Encode the hash and salt
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		time,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// VerifyPassword verifies a password against an Argon2id hash
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash
	var version int
	var memory, time uint32
	var threads uint8
	var salt, hash []byte

	_, err := fmt.Sscanf(encodedHash, "$argon2id$v=%d$m=%d,t=%d,p=%d$", &version, &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("invalid hash format: %w", err)
	}

	// Extract salt and hash
	parts := make([]string, 6)
	n := 0
	for i, char := range encodedHash {
		if char == '$' {
			n++
			if n >= 4 {
				parts[4] = encodedHash[i+1:]
				break
			}
		}
	}

	// Parse salt and hash from the last two parts
	saltHash := make([]string, 0, 2)
	lastPart := parts[4]
	for i := len(lastPart) - 1; i >= 0; i-- {
		if lastPart[i] == '$' {
			saltHash = append(saltHash, lastPart[i+1:])
			saltHash = append(saltHash, lastPart[:i])
			break
		}
	}

	if len(saltHash) != 2 {
		return false, fmt.Errorf("invalid hash format: missing salt or hash")
	}

	salt, err = base64.RawStdEncoding.DecodeString(saltHash[1])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(saltHash[0])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Generate hash from the provided password
	keyLen := uint32(len(hash))
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	// Compare hashes
	if len(computedHash) != len(hash) {
		return false, nil
	}

	for i := range hash {
		if hash[i] != computedHash[i] {
			return false, nil
		}
	}

	return true, nil
}
