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

// Password hashing uses Argon2id, a memory-hard key derivation function resistant
// to both GPU and side-channel attacks. It's the recommended algorithm from OWASP
// for password storage as of 2024.
//
// Hashes are stored in PHC (Password Hashing Competition) string format:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<base64-salt>$<base64-hash>
//
// This format is portable across languages and includes all parameters needed
// for verification.

const (
	// Default Argon2id parameters following OWASP recommendations.
	// These provide strong security while remaining performant on modern hardware.
	// Values can be tuned higher for increased security at the cost of performance.

	// defaultTime is the number of iterations (time cost)
	defaultTime = DefaultArgon2Time

	// defaultMemory is the amount of memory in KiB (memory cost)
	defaultMemory = DefaultArgon2Memory

	// defaultThreads is the degree of parallelism
	defaultThreads = DefaultArgon2Threads

	// defaultKeyLength is the length of the derived key in bytes
	defaultKeyLength = DefaultArgon2KeyLength

	// saltLength is the size of the random salt in bytes (16 bytes = 128 bits)
	saltLength = SaltLength
)

// HashPassword creates a secure password hash using the Argon2id algorithm.
//
// The function generates a cryptographically random salt and derives a hash from
// the password using the specified parameters. The result is encoded in PHC
// string format, which includes all parameters needed for later verification.
//
// Parameters:
//   - password: The plaintext password to hash
//   - time: Number of iterations (higher = slower but more secure). Use 0 for defaults.
//   - memory: Memory usage in KiB (higher = more RAM needed). Use 0 for defaults.
//   - threads: Degree of parallelism (typically CPU core count). Use 0 for defaults.
//   - keyLen: Length of the derived key in bytes. Use 0 for defaults.
//
// Returns a PHC-formatted string like:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
//
// This format is portable and includes the version, parameters, salt, and hash,
// allowing verification even if default parameters change in the future.
//
// Example:
//
//	hash, err := HashPassword("my-secret-password", 0, 0, 0, 0)
//	// hash = "$argon2id$v=19$m=65536,t=3,p=4$..."
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
		return "", NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to generate salt", err)
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

// VerifyPassword verifies a plaintext password against an Argon2id hash.
//
// The function parses the PHC-formatted hash string to extract the algorithm
// parameters, salt, and expected hash. It then re-hashes the provided password
// with the same parameters and compares the results using constant-time comparison
// to prevent timing attacks.
//
// Parameters:
//   - password: The plaintext password to verify
//   - encodedHash: The PHC-formatted hash string from HashPassword or database
//
// Returns:
//   - bool: true if the password matches, false otherwise
//   - error: validation error if the hash format is invalid or corrupted
//
// Expected PHC format:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<base64-salt>$<base64-hash>
//
// The function parses:
//  1. Algorithm identifier (must be "argon2id")
//  2. Version (e.g., v=19)
//  3. Parameters: m=memory, t=time, p=parallelism
//  4. Base64-encoded salt
//  5. Base64-encoded hash to verify against
//
// Example:
//
//	ok, err := VerifyPassword("my-secret-password", storedHash)
//	if err != nil {
//		return err // Hash is malformed
//	}
//	if !ok {
//		return errors.New("invalid password")
//	}
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse PHC-style format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ValidationError{Field: "hash", Message: "invalid hash format"}
	}

	// Parse version (e.g., "v=19")
	if !strings.HasPrefix(parts[2], "v=") {
		return false, ValidationError{Field: "hash", Message: "invalid version in hash"}
	}

	// Parse parameters: "m=65536,t=3,p=4"
	params := parts[3]
	var memory uint32
	var t uint32
	var threads uint8

	// Extract m (memory), t (time), and p (parallelism) from comma-separated pairs
	for _, kv := range strings.Split(params, ",") {
		kvParts := strings.SplitN(kv, "=", 2)
		if len(kvParts) != 2 {
			return false, ValidationError{Field: "hash", Message: "invalid params in hash"}
		}
		k := kvParts[0]
		v := kvParts[1]
		switch k {
		case "m":
			mi, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return false, ValidationError{Field: "hash", Message: fmt.Sprintf("invalid memory value: %v", err)}
			}
			memory = uint32(mi)
		case "t":
			ti, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return false, ValidationError{Field: "hash", Message: fmt.Sprintf("invalid time value: %v", err)}
			}
			t = uint32(ti)
		case "p":
			pi, err := strconv.ParseUint(v, 10, 8)
			if err != nil {
				return false, ValidationError{Field: "hash", Message: fmt.Sprintf("invalid threads value: %v", err)}
			}
			threads = uint8(pi)
		default:
			// ignore unknown params
		}
	}

	// salt and hash are base64 encoded
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ValidationError{Field: "hash", Message: fmt.Sprintf("failed to decode salt: %v", err)}
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ValidationError{Field: "hash", Message: fmt.Sprintf("failed to decode hash: %v", err)}
	}

	// Compute key using the same parameters
	if len(hash) == 0 {
		return false, ValidationError{Field: "hash", Message: "invalid hash length"}
	}
	// Safely convert length to uint32, guarding against overflow
	// We check that len(hash) fits in uint32 before conversion
	if len(hash) > int(^uint32(0)) {
		return false, ValidationError{Field: "hash", Message: "hash length too large"}
	}
	keyLen := uint32(len(hash)) // #nosec G115 -- overflow checked above
	computed := argon2.IDKey([]byte(password), salt, t, memory, threads, keyLen)

	// Use constant-time comparison to mitigate timing attacks
	if subtle.ConstantTimeCompare(computed, hash) == 1 {
		return true, nil
	}
	return false, nil
}
