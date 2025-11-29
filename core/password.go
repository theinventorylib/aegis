package core

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

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
	// Expected PHC-style format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid hash format")
	}

	// version
	if !strings.HasPrefix(parts[2], "v=") {
		return false, fmt.Errorf("invalid version in hash")
	}
	// params: m=...,t=...,p=...
	params := parts[3]
	var memory uint32
	var t uint32
	var threads uint8

	for _, kv := range strings.Split(params, ",") {
		kvParts := strings.SplitN(kv, "=", 2)
		if len(kvParts) != 2 {
			return false, fmt.Errorf("invalid params in hash")
		}
		k := kvParts[0]
		v := kvParts[1]
		switch k {
		case "m":
			mi, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return false, fmt.Errorf("invalid memory value: %w", err)
			}
			memory = uint32(mi)
		case "t":
			ti, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return false, fmt.Errorf("invalid time value: %w", err)
			}
			t = uint32(ti)
		case "p":
			pi, err := strconv.ParseUint(v, 10, 8)
			if err != nil {
				return false, fmt.Errorf("invalid threads value: %w", err)
			}
			threads = uint8(pi)
		default:
			// ignore unknown params
		}
	}

	// salt and hash are base64 encoded
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Compute key using the same parameters
	if len(hash) == 0 {
		return false, fmt.Errorf("invalid hash length")
	}
	// Safely convert length to uint32, guarding against overflow
	// We check that len(hash) fits in uint32 before conversion
	if len(hash) > int(^uint32(0)) {
		return false, fmt.Errorf("hash length too large")
	}
	keyLen := uint32(len(hash)) // #nosec G115 -- overflow checked above
	computed := argon2.IDKey([]byte(password), salt, t, memory, threads, keyLen)

	// Use constant-time comparison to mitigate timing attacks
	if subtle.ConstantTimeCompare(computed, hash) == 1 {
		return true, nil
	}
	return false, nil
}
