package referrals

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Keep database JSON intact: decoding monetary values through interface{}
// would silently convert bigint amounts and versions to float64.
func decodeReadPage(raw []byte) ([]json.RawMessage, string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return nil, "", errors.New("invalid referral page")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, "", err
	}
	if len(rows) > 101 {
		return nil, "", errors.New("referral page exceeds its limit")
	}
	var cursor string
	if len(rows) == 101 {
		var last struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rows[99], &last); err != nil || last.ID == "" {
			return nil, "", errors.New("invalid referral page cursor")
		}
		cursor = last.ID
		rows = rows[:100]
	}
	return rows, cursor, nil
}
