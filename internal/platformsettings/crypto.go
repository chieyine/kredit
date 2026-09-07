package platformsettings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Encryptor struct {
	key []byte
}

func NewEncryptor(key string) *Encryptor {
	if key == "" {
		key = os.Getenv("SETTINGS_ENCRYPTION_KEY")
	}
	if key == "" {
		key = os.Getenv("TOKEN_SECRET")
	}
	if key == "" {
		key = "kredit-default-dev-settings-encryption-key-32b"
	}
	h := sha256.Sum256([]byte(key))
	return &Encryptor{key: h[:]}
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("rand nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, cipherTextBytes := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherTextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("gcm decrypt: %w", err)
	}
	return string(plain), nil
}

func SecretFingerprint(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:8]) // first 16 hex chars (64-bit fingerprint)
}

func MaskSecret(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "sk_live_") {
		if len(trimmed) > 12 {
			return "sk_live_•••••••" + trimmed[len(trimmed)-4:]
		}
		return "sk_live_•••••••"
	}
	if strings.HasPrefix(trimmed, "sk_test_") {
		if len(trimmed) > 12 {
			return "sk_test_•••••••" + trimmed[len(trimmed)-4:]
		}
		return "sk_test_•••••••"
	}
	if len(trimmed) > 8 {
		return trimmed[:2] + "••••••••" + trimmed[len(trimmed)-4:]
	}
	return "••••••••"
}
