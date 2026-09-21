package referrals

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
)

func TestAuditReferralPagePreservesExactNumbers(t *testing.T) {
	for _, amount := range []string{"9007199254740993", "9223372036854775807", "-9223372036854775808"} {
		raw := []byte(`[{"id":"record","amount_kobo":` + amount + `,"nested":{"version":9007199254740993}}]`)
		rows, cursor, err := decodeReadPage(raw)
		if err != nil || cursor != "" {
			t.Fatalf("page: %v %s", err, cursor)
		}
		encoded, err := json.Marshal(rows)
		if err != nil || !bytes.Equal(encoded, raw) {
			t.Fatalf("number changed: %s error=%v", encoded, err)
		}
	}
}
func TestAuditReferralPageCursorAndLimits(t *testing.T) {
	rows := make([]map[string]any, 101)
	for i := range rows {
		rows[i] = map[string]any{"id": strconv.Itoa(i), "amount_kobo": i}
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	page, next, err := decodeReadPage(raw)
	if err != nil || len(page) != 100 || next != "99" {
		t.Fatalf("page length=%d cursor=%q error=%v", len(page), next, err)
	}
	for _, raw := range [][]byte{nil, []byte(`null`), []byte(`{}`), []byte(`[] {}`), bytes.ReplaceAll(raw, []byte(`"id":"99"`), []byte(`"id":null`))} {
		if _, _, err := decodeReadPage(raw); err == nil {
			t.Fatal("accepted malformed page or cursor")
		}
	}
	rows = append(rows, rows[0])
	raw, err = json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := decodeReadPage(raw); err == nil {
		t.Fatal("accepted an oversized database page")
	}
	if page, next, err := decodeReadPage([]byte(`[]`)); err != nil || len(page) != 0 || next != "" {
		t.Fatal("empty page failed")
	}
}
