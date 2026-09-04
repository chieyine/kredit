package publictoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type payload struct {
	Purpose string `json:"p"`
	ID      string `json:"i"`
	Expires int64  `json:"e"`
}

func Issue(key, purpose, id string, expires time.Time) (string, error) {
	if key == "" || purpose == "" || id == "" || expires.IsZero() {
		return "", errors.New("token key, purpose, id, and expiry are required")
	}
	body, err := json.Marshal(payload{Purpose: purpose, ID: id, Expires: expires.Unix()})
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func Parse(key, token, purpose string, now time.Time) (string, error) {
	if key == "" || purpose == "" || len(token) > 8192 {
		return "", errors.New("invalid token configuration")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("invalid token")
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", errors.New("invalid token")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
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
