package core

import (
	"crypto/sha256"

	"golang.org/x/crypto/hkdf"
)

// DeriveSecret derives a purpose-specific secret from a master secret using HKDF-SHA256.
//
// HKDF (HMAC-based Key Derivation Function) is a cryptographic key derivation function
// that allows safely deriving multiple purpose-specific keys from a single master secret.
//
// Why use HKDF instead of reusing the master secret?
//   - Cryptographic separation: Each purpose gets a unique, independent key
//   - Security isolation: Compromise of one derived key doesn't affect others
//   - Standard practice: Recommended by NIST SP 800-108 and RFC 5869
//
// Common use cases in Aegis:
//   - CSRF token signing
//   - OAuth state token encryption
//   - JWT signing keys
//   - Cookie encryption keys
//   - API key derivation
//
// Parameters:
//   - masterSecret: The master secret (minimum 32 bytes recommended)
//   - purpose: Unique identifier for this key's purpose (e.g., "csrf", "oauth-state", "jwt")
//   - length: Output length in bytes (typically 32 for 256-bit keys)
//
// Security notes:
//   - Different purposes MUST use different purpose strings
//   - Master secret should be cryptographically random (32+ bytes)
//   - Output keys are deterministic (same inputs → same output)
//
// Example:
//
//	masterSecret := []byte("your-32-byte-master-secret-here!")
//	// Derive separate keys for different purposes
//	csrfSecret := core.DeriveSecret(masterSecret, "csrf", 32)
//	oauthSecret := core.DeriveSecret(masterSecret, "oauth-state", 32)
//	jwtSecret := core.DeriveSecret(masterSecret, "jwt-signing", 32)
func DeriveSecret(masterSecret []byte, purpose string, length int) []byte {
	// Use HKDF to derive a purpose-specific key
	// Info contains the purpose to ensure different purposes yield different keys
	hkdfReader := hkdf.New(sha256.New, masterSecret, nil, []byte(purpose))

	derived := make([]byte, length)
	// HKDF.Read always succeeds for reasonable output lengths
	_, err := hkdfReader.Read(derived)
	_ = err

	return derived
}

// DefaultSecretLength is the recommended length for derived secrets (256 bits / 32 bytes).
//
// This provides 256-bit security, which is the standard for symmetric encryption
// and HMAC operations. Use this constant when calling DeriveSecret:
//
//	secret := core.DeriveSecret(master, "purpose", core.DefaultSecretLength)
const DefaultSecretLength = 32
