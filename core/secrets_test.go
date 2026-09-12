package core

import (
	"bytes"
	"testing"
)

func TestDeriveSecretDeterministic(t *testing.T) {
	master := []byte("0123456789abcdef0123456789abcdef")
	a := DeriveSecret(master, "csrf", DefaultSecretLength)
	b := DeriveSecret(master, "csrf", DefaultSecretLength)
	if !bytes.Equal(a, b) {
		t.Fatal("same inputs must produce the same key")
	}
	if len(a) != DefaultSecretLength {
		t.Fatalf("length: got %d want %d", len(a), DefaultSecretLength)
	}
}

func TestDeriveSecretSeparatesPurposes(t *testing.T) {
	master := []byte("0123456789abcdef0123456789abcdef")
	csrf := DeriveSecret(master, "csrf", DefaultSecretLength)
	oauth := DeriveSecret(master, "oauth-state", DefaultSecretLength)
	if bytes.Equal(csrf, oauth) {
		t.Fatal("different purposes must yield different keys")
	}
}

func TestDeriveSecretSeparatesMasters(t *testing.T) {
	a := DeriveSecret([]byte("master-one-master-one-master-one!"), "csrf", DefaultSecretLength)
	b := DeriveSecret([]byte("master-two-master-two-master-two!"), "csrf", DefaultSecretLength)
	if bytes.Equal(a, b) {
		t.Fatal("different masters must yield different keys")
	}
}

func TestDeriveSecretEmptyInputsYieldNil(t *testing.T) {
	if got := DeriveSecret([]byte("0123456789abcdef0123456789abcdef"), "", 32); got != nil {
		t.Fatal("empty purpose must yield nil")
	}
	if got := DeriveSecret(nil, "csrf", 32); got != nil {
		t.Fatal("empty master secret must yield nil")
	}
}

func TestHashTokenHexIsSha256Hex(t *testing.T) {
	h := hashTokenHex("token")
	if len(h) != 64 {
		t.Fatalf("sha256 hex length: %d", len(h))
	}
	if !isHashedToken(h) {
		t.Fatal("hash should be recognized by isHashedToken")
	}
	if isHashedToken("token") {
		t.Fatal("plaintext token must not be misidentified as hashed")
	}
}
