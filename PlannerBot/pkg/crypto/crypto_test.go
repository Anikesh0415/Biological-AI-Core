package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKeyConsistency(t *testing.T) {
	code := "PLAN-ABCD-1234"
	k1 := DeriveKey(code, nil)
	k2 := DeriveKey(code, nil)

	if len(k1) != KeyLen {
		t.Fatalf("expected key length %d, got %d", KeyLen, len(k1))
	}
	if !bytes.Equal(k1, k2) {
		t.Fatal("DeriveKey should be deterministic for identical inputs")
	}

	// Case-insensitivity check
	k3 := DeriveKey("plan-abcd-1234", nil)
	if !bytes.Equal(k1, k3) {
		t.Fatal("DeriveKey should be case-insensitive for user codes")
	}

	// Different codes should yield different keys
	k4 := DeriveKey("PLAN-DIFFERENT-CODE", nil)
	if bytes.Equal(k1, k4) {
		t.Fatal("different codes must yield different keys")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := DeriveKey("PLAN-SECRET-TEST", nil)
	message := []byte("Hello, this is a private sync reminder for my homework!")

	encrypted, err := Encrypt(message, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(encrypted, message) {
		t.Fatal("encrypted data should not equal plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, message) {
		t.Fatalf("expected %q, got %q", string(message), string(decrypted))
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := DeriveKey("PLAN-USER-ALICE", nil)
	key2 := DeriveKey("PLAN-USER-BOB", nil)

	message := []byte("Secret reminder data")
	encrypted, err := Encrypt(message, key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Bob attempting to decrypt Alice's data must fail
	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := DeriveKey("PLAN-TAMPER-TEST", nil)
	message := []byte("Tamper test payload")

	encrypted, err := Encrypt(message, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Tamper with one byte in the ciphertext
	encrypted[len(encrypted)-1] ^= 0xFF

	_, err = Decrypt(encrypted, key)
	if err == nil {
		t.Fatal("expected decryption to fail when ciphertext is tampered (AEAD authentication failed)")
	}
}

func TestGeneratePairingCode(t *testing.T) {
	code1 := GeneratePairingCode()
	code2 := GeneratePairingCode()

	if len(code1) < 10 {
		t.Fatalf("code too short: %s", code1)
	}
	if code1 == code2 {
		t.Fatalf("pairing codes should be random and distinct: %s vs %s", code1, code2)
	}
}

func TestHMACAuthentication(t *testing.T) {
	key := DeriveKey("PLAN-HMAC-KEY", nil)
	token := []byte("challenge-token-123456789")

	sig := HMACSign(token, key)

	if !HMACVerify(token, sig, key) {
		t.Fatal("HMAC verification failed for valid signature")
	}

	wrongKey := DeriveKey("PLAN-WRONG-KEY", nil)
	if HMACVerify(token, sig, wrongKey) {
		t.Fatal("HMAC verification should fail for wrong key")
	}

	tamperedToken := []byte("challenge-token-123456780")
	if HMACVerify(tamperedToken, sig, key) {
		t.Fatal("HMAC verification should fail for tampered token")
	}
}
