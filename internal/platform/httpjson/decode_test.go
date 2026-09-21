package httpjson

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"strings"
	"testing"
)

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestDecodeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		limit      int64
		ok         bool
	}{
		{"one object", `{"v":9007199254740993}`, 100, true},
		{"exact limit", `{"v":1}`, 7, true},
		{"whitespace", "{\"v\":1} \n", 100, true},
		{"second JSON", `{"v":1}{}`, 100, false},
		{"garbage", `{"v":1}garbage`, 100, false},
		{"oversized", `{"v":1} `, 7, false},
		{"invalid JSON", `{"v":`, 100, false},
		{"empty", ``, 100, false},
		{"zero limit", `{}`, 0, false},
		{"negative limit", `{}`, -1, false},
		{"overflowing limit", `{}`, math.MaxInt64, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out json.RawMessage
			err := Decode(strings.NewReader(tc.body), tc.limit, &out)
			if (err == nil) != tc.ok {
				t.Fatalf("err=%v want success=%v", err, tc.ok)
			}
			if tc.ok && !bytes.Equal(out, bytes.TrimSpace([]byte(tc.body))) {
				t.Fatal("valid JSON bytes changed")
			}
		})
	}
	var out any
	if Decode(nil, 100, &out) == nil {
		t.Fatal("nil reader accepted")
	}
	if Decode(io.MultiReader(strings.NewReader(`{}`), failedReader{}), 100, &out) == nil {
		t.Fatal("incomplete response accepted")
	}
}
