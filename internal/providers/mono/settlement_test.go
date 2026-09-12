package mono

import (
	"encoding/json"
	"errors"
	"kredit/internal/collections"
	"kredit/internal/settlement"
	"net/http"
	"testing"
)

func TestBankRegistrationUsesProviderAccountEvidence(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(map[bool]string{false: "matched", true: "mismatched"}[mismatch], func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("mono-sec-key") != "test_sk_fixture" {
					t.Error("missing provider authentication")
				}
				switch r.URL.Path {
				case "/v3/banks/list":
					if r.Method != "GET" {
						t.Error("wrong bank lookup method")
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"status": "successful", "data": map[string]any{"banks": []map[string]string{{"name": "Synthetic Bank", "nip_code": "000013", "bank_code": "050"}}}})
				case "/v2/payments/payout/sub-account":
					if r.Method != "POST" {
						t.Error("wrong registration method")
					}
					var body map[string]string
					if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 2 || body["nip_code"] != "000013" || body["account_number"] != "1234567890" {
						t.Error("registration does not match bank contract")
					}
					account := "1234567890"
					if mismatch {
						account = "0987654321"
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"status": "successful", "data": map[string]string{"id": "provider-sub-account", "name": "Synthetic Seller", "account_number": account, "nip_code": "000013"}})
				default:
					t.Error("unexpected provider path")
					http.NotFound(w, r)
				}
			})
			banks, err := c.Banks(t.Context())
			if err != nil || len(banks) != 1 || banks[0].Code != "000013" {
				t.Fatalf("NIP bank routing lost: %+v %v", banks, err)
			}
			destination, err := c.CreateDestination(t.Context(), settlement.Input{Reference: "request", OrganizationID: "seller", BankCode: banks[0].Code, AccountNumber: "1234567890"})
			if mismatch {
				if !errors.Is(err, settlement.ErrUnknown) {
					t.Fatal("different account accepted", err)
				}
			} else if err != nil || destination.ProviderReference != "provider-sub-account" || destination.AccountLast4 != "7890" || destination.AccountName != "Synthetic Seller" {
				t.Fatalf("destination evidence: %+v %v", destination, err)
			}
		})
	}
}

func TestDebitUsesFrozenExactSplit(t *testing.T) {
	called := false
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		var body map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("bad body")
		}
		var split struct {
			Type         string `json:"type"`
			FeeBearer    string `json:"fee_bearer"`
			Distribution []struct {
				Account string `json:"account"`
				Value   int64  `json:"value"`
			} `json:"distribution"`
		}
		if json.Unmarshal(body["split"], &split) != nil || split.Type != "fixed" || split.FeeBearer != "business" || len(split.Distribution) != 1 || split.Distribution[0].Account != "original-account" || split.Distribution[0].Value != 20000 {
			t.Error("incorrect frozen split")
		}
		w.WriteHeader(503)
	})
	c.partial = false
	route := &collections.SettlementRoute{Provider: c.Name(), Connection: c.ConnectionIdentity(), Destination: "original-account", Method: "provider_split", NetAmountKobo: 20000}
	in := collections.Request{ExternalReference: "synthetic-split", MandateReference: "original-mandate", AmountKobo: 20000, Currency: "NGN", SettlementRoute: route}
	_, _ = c.Submit(t.Context(), in)
	if !called {
		t.Fatal("split was not submitted")
	}
	called = false
	route.Connection = "other-account"
	if _, err := c.Submit(t.Context(), in); err == nil || called {
		t.Fatal("split used a different provider account")
	}
}
