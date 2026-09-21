package notifications

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// The only caller transcribes voice notes, whose existing parser limit is 4 MiB.
// Metadata and download reads both have a one-byte overflow sentinel.
const maxMetaVoiceBytes = 4 << 20
const maxMetaMetadataBytes = 64 << 10
const metaMediaOrigin = "https://lookaside.fbsbx.com"

var metaMediaEndpoint = regexp.MustCompile(`^https://graph\.facebook\.com/(v[0-9]+\.[0-9]+)/([0-9]+)/messages$`)
var metaMediaID = regexp.MustCompile(`^[0-9]{1,64}$`)

func downloadMetaVoice(ctx context.Context, source *http.Client, endpoint, phoneID, token, mediaID string) ([]byte, string, error) {
	match := metaMediaEndpoint.FindStringSubmatch(endpoint)
	if source == nil || len(match) != 3 || match[2] != phoneID || !metaMediaID.MatchString(mediaID) || strings.TrimSpace(token) == "" || strings.ContainsAny(token, "\r\n") {
		return nil, "", errors.New("invalid media request configuration")
	}
	// A copied client prevents any caller from accidentally enabling redirects
	// that would send an authorization token outside the validated origin.
	client := *source
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	metadataURL := "https://graph.facebook.com/" + match[1] + "/" + mediaID + "?phone_number_id=" + url.QueryEscape(phoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, "", errors.New("invalid media metadata request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := client.Do(req)
	if err != nil {
		return nil, "", errors.New("media metadata request failed")
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(response.Body)
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("media metadata status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxMetaMetadataBytes+1))
	if err != nil || len(body) > maxMetaMetadataBytes {
		return nil, "", errors.New("invalid or oversized media metadata")
	}
	var metadata struct {
		ID       string      `json:"id"`
		Product  string      `json:"messaging_product"`
		URL      string      `json:"url"`
		MIME     string      `json:"mime_type"`
		FileSize json.Number `json:"file_size"`
		SHA256   string      `json:"sha256"`
	}
	if json.Unmarshal(body, &metadata) != nil || metadata.ID != mediaID || metadata.Product != "whatsapp" {
		return nil, "", errors.New("media metadata identity does not match")
	}
	size, err := metadata.FileSize.Int64()
	if err != nil || size <= 0 || size > maxMetaVoiceBytes {
		return nil, "", errors.New("voice note is empty or too large")
	}
	mediaType, _, err := mime.ParseMediaType(metadata.MIME)
	if err != nil || !strings.HasPrefix(mediaType, "audio/") {
		return nil, "", errors.New("media is not an audio voice note")
	}
	digest, err := hex.DecodeString(metadata.SHA256)
	if err != nil || len(digest) != sha256.Size {
		digest, err = base64.StdEncoding.DecodeString(metadata.SHA256)
	}
	if err != nil || len(digest) != sha256.Size {
		return nil, "", errors.New("media checksum is missing or invalid")
	}
	if !allowedMetaMediaURL(metadata.URL) {
		return nil, "", errors.New("media download origin is not allowed")
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, metadata.URL, nil)
	if err != nil {
		return nil, "", errors.New("invalid media download request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	response, err = client.Do(req)
	if err != nil {
		// Do not expose short-lived signed media URLs through error logging.
		return nil, "", errors.New("media download failed")
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(response.Body)
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("media download status %d", response.StatusCode)
	}
	if response.ContentLength > maxMetaVoiceBytes {
		return nil, "", errors.New("voice note is too large")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxMetaVoiceBytes+1))
	if err != nil || len(data) > maxMetaVoiceBytes || int64(len(data)) != size {
		return nil, "", errors.New("media download is incomplete or oversized")
	}
	actual := sha256.Sum256(data)
	if !bytes.Equal(actual[:], digest) {
		return nil, "", errors.New("media content does not match its checksum")
	}
	return data, mediaType, nil
}

func allowedMetaMediaURL(raw string) bool {
	if len(raw) > 8192 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.Fragment != "" || strings.ToLower(u.Hostname()) != strings.TrimPrefix(metaMediaOrigin, "https://") || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	// No arbitrary crawler paths or credential-bearing URLs are accepted.
	return u.RawPath == "" && (strings.HasPrefix(u.Path, "/whatsapp_business/attachments/") || strings.HasPrefix(u.Path, "/whatsapp_v2/attachments/")) && !strings.Contains(u.Path, "/../") && !strings.Contains(u.Path, "\\")
}
