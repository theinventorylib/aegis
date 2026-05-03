package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// EncryptionPrefix is prepended to ciphertext produced by SealWithDerivedKey
// so callers can distinguish encrypted values from legacy plaintext rows
// during incremental migrations and switch the format later without breaking
// existing data.
const EncryptionPrefix = "enc:v1:"

// SealWithKey encrypts plaintext using AES-256-GCM with the supplied 32-byte
// key (typically obtained via DeriveSecret with a purpose-specific label).
//
// The returned string has the form "enc:v1:" + base64(nonce|ciphertext|tag).
// An empty plaintext is returned as the empty string unchanged so callers
// can persist optional fields without forcing them through the codec.
func SealWithKey(key []byte, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if len(key) == 0 {
		return "", errors.New("crypto: empty encryption key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: nonce: %w", err)
	}
	ct := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return EncryptionPrefix + base64.RawStdEncoding.EncodeToString(out), nil
}

// OpenWithKey reverses SealWithKey using the supplied 32-byte key.
//
// Values that do not carry the EncryptionPrefix are treated as legacy
// plaintext and returned unchanged so callers can transparently read rows
// written before encryption was enabled. Re-writing such a row with
// SealWithKey will upgrade it.
func OpenWithKey(key []byte, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, EncryptionPrefix) {
		return value, nil
	}
	if len(key) == 0 {
		return "", errors.New("crypto: empty decryption key")
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, EncryptionPrefix))
	if err != nil {
		return "", fmt.Errorf("crypto: decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: gcm: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("crypto: ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: open: %w", err)
	}
	return string(pt), nil
}

// IsEncrypted reports whether value was produced by SealWithDerivedKey.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncryptionPrefix)
}
