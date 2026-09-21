// Package httpjson decodes bounded, complete JSON responses from external services.
package httpjson

import (
	"encoding/json"
	"errors"
	"io"
	"math"
)

// Decode reads at most limit+1 bytes and accepts exactly one complete JSON value.
// A decoder's first successful Decode, or an EOF manufactured by LimitReader,
// must not make a truncated provider result authoritative. The caller owns body
// closure and the HTTP request timeout. Vendor-added JSON fields remain allowed.
func Decode(body io.Reader, limit int64, output any) error {
	if body == nil || limit < 1 || limit == math.MaxInt64 {
		return errors.New("a response reader and bounded positive size limit are required")
	}
	raw, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return errors.New("provider response is incomplete")
	}
	if int64(len(raw)) > limit {
		return errors.New("provider response exceeds the size limit")
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return errors.New("provider response is not a complete JSON value")
	}
	return nil
}
