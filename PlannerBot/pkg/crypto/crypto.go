package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// DefaultSalt is the standard domain salt for pairing key derivation.
	DefaultSalt = "planner_bot_p2p_salt_v1"
	// KeyLen is the length of an AES-256 key in bytes.
	KeyLen = 32
	// NonceLen is the standard GCM nonce length in bytes.
	NonceLen = 12
)

var (
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
	ErrDecryptionFailed   = errors.New("crypto: decryption or authentication failed (invalid key or tampered data)")
	ErrInvalidKeyLength   = errors.New("crypto: key must be 32 bytes for AES-256")
)

// DeriveKey derives a cryptographically strong 32-byte AES-256 key from a user pairing code.
// It implements PBKDF2-HMAC-SHA256 with 100,000 iterations in pure standard library Go.
func DeriveKey(userCode string, salt []byte) []byte {
	if len(salt) == 0 {
		salt = []byte(DefaultSalt)
	}

	cleanCode := strings.TrimSpace(strings.ToUpper(userCode))
	return pbkdf2SHA256([]byte(cleanCode), salt, 100000, KeyLen)
}

// pbkdf2SHA256 performs standard PBKDF2 key derivation using HMAC-SHA256.
func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var result []byte
	buf := make([]byte, 4)

	for block := 1; block <= numBlocks; block++ {
		// U_1 = PRF(password, salt || INT_32_BE(block))
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf)
		u := prf.Sum(nil)

		// T_block = U_1
		t := make([]byte, hashLen)
		copy(t, u)

		// U_2 ... U_iter
		for i := 2; i <= iter; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for j := 0; j < hashLen; j++ {
				t[j] ^= u[j]
			}
		}

		result = append(result, t...)
	}

	return result[:keyLen]
}

// Encrypt encrypts and authenticates plaintext using AES-256-GCM.
// The output format is: [12-byte Nonce] + [Ciphertext + 16-byte GCM Tag].
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != KeyLen {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}

	// Seal appends the ciphertext and tag to nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt authenticates and decrypts AES-256-GCM ciphertext.
// Expects input format: [12-byte Nonce] + [Ciphertext + 16-byte GCM Tag].
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != KeyLen {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// GeneratePairingCode generates a user-friendly pairing code like "PLAN-4829-9182".
func GeneratePairingCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	part1 := hex.EncodeToString(b[:2])
	part2 := hex.EncodeToString(b[2:])
	return fmt.Sprintf("PLAN-%s-%s", strings.ToUpper(part1), strings.ToUpper(part2))
}

// GenerateNonceToken generates a random hex string for challenge-response handshakes.
func GenerateNonceToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HMACSign signs data with a shared secret key using HMAC-SHA256.
func HMACSign(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// HMACVerify checks if the signature matches using constant-time comparison.
func HMACVerify(data, signature, key []byte) bool {
	expected := HMACSign(data, key)
	return hmac.Equal(signature, expected)
}
