package platformsettings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
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

// Stored ciphertext carries the identity of the key that produced it, so a root
// key can be rotated without turning every stored secret into undecodable
// bytes. A value written before key identity existed has no prefix and is read
// with the current key, which is the only key it could have been written with.
const (
	keyedCiphertextPrefix = "k1."
	minimumRootKeyBytes   = 32
)

var errRootKeyMissing = errors.New("SETTINGS_ENCRYPTION_KEY must contain at least 32 bytes of independently generated key material")

type Encryptor struct {
	key   []byte
	keyID string
}

// NewEncryptor derives an AES-256-GCM key from the deployment root key. An
// absent or too-short key produces an encryptor that refuses every operation
// rather than one that silently uses a guessable default: a missing key must
// fail the secret operation, never quietly weaken it.
func NewEncryptor(key string) *Encryptor {
	if key == "" {
		key = os.Getenv("SETTINGS_ENCRYPTION_KEY")
	}
	if len(key) < minimumRootKeyBytes {
		return &Encryptor{}
	}
	material := sha256.Sum256([]byte("kredit/platform-settings/v1\x00" + key))
	identity := sha256.Sum256([]byte("kredit/platform-settings/key-id/v1\x00" + key))
	return &Encryptor{key: material[:], keyID: hex.EncodeToString(identity[:4])}
}

// Ready reports whether secret operations can be performed at all. Callers use
// it to explain a missing root key instead of failing at the first write.
func (e *Encryptor) Ready() bool { return e != nil && len(e.key) == 32 }

// KeyID identifies the root key in use without revealing it, so an operator can
// tell two environments apart and confirm a rotation took effect.
func (e *Encryptor) KeyID() string {
	if !e.Ready() {
		return ""
	}
	return e.keyID
}

func (e *Encryptor) gcm() (cipher.AEAD, error) {
	if !e.Ready() {
		return nil, errRootKeyMissing
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return aead, nil
}

// context binds a ciphertext to the setting it belongs to. Without it a
// ciphertext lifted from one settings row decrypts cleanly in another, so a
// mis-targeted write or a partial restore could swap one provider credential
// for a different one and nothing would notice.
func secretContext(settingKey string) []byte {
	return []byte("kredit/platform-settings/v1\x00" + settingKey)
}

// Encrypt seals a secret for one specific setting key.
func (e *Encryptor) Encrypt(settingKey, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if strings.TrimSpace(settingKey) == "" {
		return "", errors.New("a setting key is required to encrypt its secret")
	}
	aead, err := e.gcm()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("rand nonce: %w", err)
	}
	sealed := aead.Seal(nonce, nonce, []byte(plaintext), secretContext(settingKey))
	return keyedCiphertextPrefix + e.keyID + "." + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a secret for the setting key it was sealed against. A
// ciphertext written under a different root key is reported as such rather than
// as corruption, so a rotation is diagnosable.
func (e *Encryptor) Decrypt(settingKey, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !e.Ready() {
		return "", errRootKeyMissing
	}
	payload := ciphertext
	aad := secretContext(settingKey)
	if rest, ok := strings.CutPrefix(ciphertext, keyedCiphertextPrefix); ok {
		id, encoded, found := strings.Cut(rest, ".")
		if !found {
			return "", errors.New("malformed settings ciphertext")
		}
		if !hmac.Equal([]byte(id), []byte(e.keyID)) {
			return "", fmt.Errorf("settings secret was sealed with root key %s; the configured key is %s", id, e.keyID)
		}
		payload = encoded
	} else {
		// Written before key identity and context binding existed.
		aad = nil
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	aead, err := e.gcm()
	if err != nil {
		return "", err
	}
	if len(data) < aead.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, sealed := data[:aead.NonceSize()], data[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, sealed, aad)
	if err != nil {
		return "", fmt.Errorf("gcm decrypt: %w", err)
	}
	return string(plain), nil
}

// SecretFingerprint identifies a stored secret without being reversible. It is
// keyed with the root key, so the same credential in two environments produces
// two different fingerprints and a fingerprint cannot be attacked offline to
// recover a low-entropy secret.
func (e *Encryptor) SecretFingerprint(plaintext string) string {
	if plaintext == "" || !e.Ready() {
		return ""
	}
	mac := hmac.New(sha256.New, e.key)
	mac.Write([]byte("kredit/platform-settings/fingerprint/v1\x00" + plaintext))
	return hex.EncodeToString(mac.Sum(nil)[:8])
}

// MaskSecret describes a stored secret without disclosing any of it. Only the
// recognised vendor prefix survives, because a prefix identifies the kind of
// credential while the trailing characters would reduce the work needed to
// guess the rest.
func MaskSecret(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	for _, prefix := range []string{"sk_live_", "sk_test_", "live_sk_", "test_sk_", "pk_live_", "pk_test_"} {
		if strings.HasPrefix(trimmed, prefix) {
			return prefix + "••••••••"
		}
	}
	return "••••••••"
}
