package bankdebit

import (
	"encoding/json"
	"testing"
)

func TestExactNairaKobo(t *testing.T) {
	for _, kobo := range []int64{1, 99, 100, 12345, 9007199254740991} {
		n := Naira(kobo)
		got, e := Kobo(n)
		if e != nil || got != kobo {
			t.Fatalf("%d became %s: %d %v", kobo, n, got, e)
		}
	}
	for _, n := range []json.Number{"0.001", "-1", "92233720368547759", "bad"} {
		if _, e := Kobo(n); e == nil {
			t.Fatalf("accepted %s", n)
		}
	}
}
