package core

import (
	"strings"
	"testing"
)

func testCryptoKey() []byte {
	// 32-byte key for AES-256.
	key, err := randomBytes(32)
	if err != nil {
		panic(err)
	}
	return key
}

func TestSealOpenRoundTrip(t *testing.T) {
	key := testCryptoKey()
	pt := "super-secret-refresh-token"
	sealed, err := SealWithKey(key, pt)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if !IsEncrypted(sealed) {
		t.Fatal("sealed value missing EncryptionPrefix")
	}
	if strings.Contains(sealed, pt) {
		t.Fatal("plaintext leaked into sealed value")
	}
	opened, err := OpenWithKey(key, sealed)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened != pt {
		t.Fatalf("round trip: got %q want %q", opened, pt)
	}
}

func TestOpenWrongKeyFails(t *testing.T) {
	sealed, err := SealWithKey(testCryptoKey(), "data")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := OpenWithKey(testCryptoKey(), sealed); err == nil {
		t.Fatal("expected failure opening with a different key")
	}
}

func TestTamperedCiphertextFails(t *testing.T) {
	key := testCryptoKey()
	sealed, err := SealWithKey(key, "data")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	// Flip a byte inside the base64 payload.
	tampered := sealed[:len(sealed)-4] + "AAAA"
	if _, err := OpenWithKey(key, tampered); err == nil {
		t.Fatal("expected failure on tampered ciphertext")
	}
}

func TestSealEmptyPlaintextUnchanged(t *testing.T) {
	sealed, err := SealWithKey(testCryptoKey(), "")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed != "" {
		t.Fatalf("empty plaintext should stay empty, got %q", sealed)
	}
}

func TestOpenLegacyPlaintextUnchanged(t *testing.T) {
	value := "legacy-plaintext-token"
	opened, err := OpenWithKey(testCryptoKey(), value)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened != value {
		t.Fatalf("legacy value should pass through unchanged, got %q", opened)
	}
}

func TestSealWithEmptyKeyFails(t *testing.T) {
	if _, err := SealWithKey(nil, "data"); err == nil {
		t.Fatal("expected failure with empty key")
	}
}

func TestIsEncrypted(t *testing.T) {
	if IsEncrypted("plain") {
		t.Fatal("plain value reported encrypted")
	}
	sealed, _ := SealWithKey(testCryptoKey(), "x")
	if !IsEncrypted(sealed) {
		t.Fatal("sealed value not reported encrypted")
	}
}
