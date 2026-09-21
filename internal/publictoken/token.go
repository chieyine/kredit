package publictoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const maxTokenBytes = 8192

var tokenEncoding = base64.RawURLEncoding.Strict()

type payload struct {
	Purpose string `json:"p"`
	ID      string `json:"i"`
	Expires int64  `json:"e"`
}

func Issue(key, purpose, id string, expires time.Time) (string, error) {
	if key == "" || purpose == "" || id == "" || expires.IsZero() {
		return "", errors.New("token key, purpose, id, and expiry are required")
	}
	// Reject lossy JSON string conversion and bound allocation before encoding.
	if !utf8.ValidString(purpose) || !utf8.ValidString(id) || len(purpose) > maxTokenBytes || len(id) > maxTokenBytes {
		return "", errors.New("invalid token reference")
	}
	body, err := json.Marshal(payload{Purpose: purpose, ID: id, Expires: expires.Unix()})
	if err != nil {
		return "", err
	}
	if tokenEncoding.EncodedLen(len(body))+1+tokenEncoding.EncodedLen(sha256.Size) > maxTokenBytes {
		return "", errors.New("token reference exceeds the size limit")
	}
	encoded := tokenEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + tokenEncoding.EncodeToString(mac.Sum(nil)), nil
}

func Parse(key, token, purpose string, now time.Time) (string, error) {
	if key == "" || purpose == "" || len(token) > maxTokenBytes {
		return "", errors.New("invalid token configuration")
	}
	parts := strings.Split(token, ".")
	// Strict base64 still ignores CR/LF, so reject those explicitly. A public
	// reference must have one textual representation, including its signature.
	if len(parts) != 2 || strings.ContainsAny(token, "\r\n") || len(parts[1]) != tokenEncoding.EncodedLen(sha256.Size) {
		return "", errors.New("invalid token")
	}
	signature, err := tokenEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("invalid token")
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", errors.New("invalid token")
	}
	body, err := tokenEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid token")
	}
	var value payload
	if json.Unmarshal(body, &value) != nil || value.Purpose != purpose || value.ID == "" {
		return "", errors.New("invalid token")
	}
	if now.Unix() >= value.Expires {
		return "", errors.New("expired token")
	}
	return value.ID, nil
}
