package emailotp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// OTPStorageMethod defines how OTPs are stored in the database.
//
// This interface allows different storage strategies:
//   - PlainOTPStorage: Store OTPs in plain text (not recommended for production)
//   - HashedOTPStorage: Store bcrypt hashes (recommended, irreversible)
//   - EncryptedOTPStorage: Store AES-256-GCM encrypted OTPs (reversible)
//
// Security Considerations:
//   - Plain text: Fast but insecure, suitable only for development
//   - Hashed: Secure but slower, best for production
//   - Encrypted: Reversible but requires secure key management
type OTPStorageMethod interface {
	// Store encodes an OTP for database storage.
	//
	// Parameters:
	//   - otp: Plain text OTP code
	//
	// Returns:
	//   - string: Encoded OTP for storage
	//   - error: If encoding fails
	Store(otp string) (string, error)

	// Compare verifies an input OTP against the stored value.
	//
	// Parameters:
	//   - stored: Encoded OTP from database
	//   - input: Plain text OTP from user
	//
	// Returns:
	//   - bool: true if OTPs match
	//   - error: If comparison fails
	Compare(stored, input string) (bool, error)
}

// ========== Plain Text Storage (default, not recommended for production) ==========

// PlainOTPStorage stores OTPs in plain text.
//
// Security Warning:
// This is NOT recommended for production. OTPs are stored without encryption,
// making them vulnerable to database breaches.
//
// Use Cases:
//   - Development and testing
//   - Low-security environments
//
// Example:
//
//	storage := emailotp.NewPlainOTPStorage()
//	stored, _ := storage.Store("123456")  // Returns "123456"
type PlainOTPStorage struct{}

// NewPlainOTPStorage creates a new plain text OTP storage
func NewPlainOTPStorage() *PlainOTPStorage {
	return &PlainOTPStorage{}
}

// Store returns the OTP as-is
func (p *PlainOTPStorage) Store(otp string) (string, error) {
	return otp, nil
}

// Compare performs direct string comparison
func (p *PlainOTPStorage) Compare(stored, input string) (bool, error) {
	return stored == input, nil
}

// ========== Hashed Storage (recommended) ==========

// HashedOTPStorage stores OTPs using bcrypt hashing.
//
// Security Benefits:
//   - OTPs cannot be reversed from database
//   - Resistant to rainbow table attacks
//   - Configurable work factor (cost)
//
// Performance:
//   - Slower than plain text due to bcrypt cost
//   - Default cost: 10 (recommended balance)
//
// Use Cases:
//   - Production environments
//   - High-security applications
//   - Compliance requirements (GDPR, PCI-DSS)
//
// Example:
//
//	storage := emailotp.NewHashedOTPStorage(10)  // Cost factor 10
//	stored, _ := storage.Store("123456")        // Returns bcrypt hash
//	valid, _ := storage.Compare(stored, "123456") // true
type HashedOTPStorage struct {
	cost int // bcrypt cost factor (default: 10, range: 4-31)
}

// NewHashedOTPStorage creates a new hashed OTP storage
func NewHashedOTPStorage(cost int) *HashedOTPStorage {
	if cost == 0 {
		cost = bcrypt.DefaultCost // Usually 10
	}
	return &HashedOTPStorage{cost: cost}
}

// Store hashes the OTP using bcrypt
func (h *HashedOTPStorage) Store(otp string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(otp), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash OTP: %w", err)
	}
	return string(hashed), nil
}

// Compare verifies the OTP against the bcrypt hash
func (h *HashedOTPStorage) Compare(stored, input string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(input))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to compare OTP: %w", err)
	}
	return true, nil
}

// ========== Encrypted Storage (for reversible encryption) ==========

// EncryptedOTPStorage stores OTPs using AES-256-GCM encryption.
//
// Security Characteristics:
//   - OTPs are encrypted with AES-256 (industry standard)
//   - GCM mode provides authenticated encryption
//   - Requires secure 32-byte encryption key
//
// Use Cases:
//   - When OTPs need to be retrieved in plain text
//   - Audit logging requirements
//   - Multi-system verification
//
// Security Warning:
// Key management is critical. Store encryption keys securely
// (environment variables, secrets management systems).
//
// Example:
//
//	// From 32-byte key
//	key := []byte("12345678901234567890123456789012")  // Must be exactly 32 bytes
//	storage, _ := emailotp.NewEncryptedOTPStorage(key)
//
//	// From string (hashed to 32 bytes)
//	storage := emailotp.NewEncryptedOTPStorageFromString("my-secret-key")
type EncryptedOTPStorage struct {
	key []byte // 32-byte encryption key for AES-256
}

// NewEncryptedOTPStorage creates a new encrypted OTP storage
// key must be exactly 32 bytes for AES-256
func NewEncryptedOTPStorage(key []byte) (*EncryptedOTPStorage, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes for AES-256")
	}
	return &EncryptedOTPStorage{key: key}, nil
}

// NewEncryptedOTPStorageFromString creates encrypted storage from a string key (hashed to 32 bytes)
func NewEncryptedOTPStorageFromString(keyString string) *EncryptedOTPStorage {
	hash := sha256.Sum256([]byte(keyString))
	return &EncryptedOTPStorage{key: hash[:]}
}

// Store encrypts the OTP using AES-256-GCM
func (e *EncryptedOTPStorage) Store(otp string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(otp), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Compare decrypts the stored OTP and compares it with the input
func (e *EncryptedOTPStorage) Compare(stored, input string) (bool, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return false, fmt.Errorf("failed to decode encrypted OTP: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return false, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return false, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return false, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt OTP: %w", err)
	}

	return string(plaintext) == input, nil
}

// ========== Factory Function ==========

// NewOTPStorage creates the appropriate OTP storage method based on config
func NewOTPStorage(method string, encryptionKey string) (OTPStorageMethod, error) {
	switch method {
	case "plain":
		return NewPlainOTPStorage(), nil
	case "hashed":
		return NewHashedOTPStorage(bcrypt.DefaultCost), nil
	case "encrypted":
		if encryptionKey == "" {
			return nil, fmt.Errorf("encryption key required for encrypted OTP storage")
		}
		return NewEncryptedOTPStorageFromString(encryptionKey), nil
	default:
		return nil, fmt.Errorf("unsupported OTP storage method: %s (use 'plain', 'hashed', or 'encrypted')", method)
	}
}
