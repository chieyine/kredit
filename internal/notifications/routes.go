package notifications

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"kredit/internal/platformsettings"
)

// Routing is committed before contacting a provider. Changing the active
// connector affects new events, never an existing send or its receipt lookup.
func (p *ConfiguredProvider) messageRoute(ctx context.Context, event string, candidate *platformsettings.NotificationConnector) (*platformsettings.NotificationConnector, error) {
	if p.pool == nil || len(p.submissionKey) < 32 || p.encryptor == nil || !p.encryptor.Ready() || event == "" {
		return nil, errors.New("persistent provider routing is unavailable")
	}
	mac := hmac.New(sha256.New, p.submissionKey)
	mac.Write([]byte("message-route:" + p.channel + ":" + event))
	key := hex.EncodeToString(mac.Sum(nil))
	var ciphertext string
	err := p.pool.QueryRow(ctx, `SELECT config_ciphertext FROM app.message_routes WHERE event_key=$1 AND channel=$2`, key, p.channel).Scan(&ciphertext)
	if errors.Is(err, pgx.ErrNoRows) && candidate != nil {
		encoded, e := json.Marshal(candidate)
		if e != nil {
			return nil, e
		}
		sealed, e := p.encryptor.Encrypt("message-route:"+key, string(encoded))
		if e != nil {
			return nil, e
		}
		adapter := candidate.Adapter
		if adapter == "" {
			adapter = DefaultAdapter(p.channel)
		}
		_, e = p.pool.Exec(ctx, `INSERT INTO app.message_routes(event_key,channel,adapter,config_ciphertext) VALUES($1,$2,$3,$4) ON CONFLICT(event_key) DO NOTHING`, key, p.channel, adapter, sealed)
		if e != nil {
			return nil, e
		}
		err = p.pool.QueryRow(ctx, `SELECT config_ciphertext FROM app.message_routes WHERE event_key=$1 AND channel=$2`, key, p.channel).Scan(&ciphertext)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("original provider route is unavailable; review the message before retrying")
	}
	if err != nil {
		return nil, err
	}
	plain, err := p.encryptor.Decrypt("message-route:"+key, ciphertext)
	if err != nil {
		return nil, errors.New("original provider connection cannot be decrypted")
	}
	var config platformsettings.NotificationConnector
	if json.Unmarshal([]byte(plain), &config) != nil {
		return nil, errors.New("original provider connection is invalid")
	}
	return &config, nil
}
