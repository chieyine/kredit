package mono

import (
	"context"
	"encoding/json"
	"fmt"
	"kredit/internal/billing"
	"net/http"
	"testing"
	"time"
)

func TestSeparateFeeMandateAndBusinessCustomer(t *testing.T) {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	end := start.AddDate(1, 0, 0)
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Method == "POST" {
			if json.NewDecoder(r.Body).Decode(&body) != nil {
				t.Fatal("invalid body")
			}
		}
		switch r.URL.Path {
		case "/v2/customers":
			if body["first_name"] != "Example" || body["last_name"] != "Trading Limited" {
				t.Error("business name was replaced with shareholder name")
			}
			_, _ = fmt.Fprint(w, `{"status":"successful","data":{"id":"fee-customer"}}`)
		case "/v2/payments/initiate":
			if body["mandate_type"] != "emandate" || body["debit_type"] != "variable" || body["amount"] != float64(100000) {
				t.Error("wrong fee authorization")
			}
			if _, ok := body["allow_partial_sweep"]; ok {
				t.Error("fee mandate enabled Sweep")
			}
			_, _ = fmt.Fprint(w, `{"status":"successful","data":{"mandate_id":"fee-mandate","mono_url":"https://authorise.mono.co/fee","reference":"fee-reference"}}`)
		case "/v3/payments/mandates/fee-mandate":
			_, _ = fmt.Fprintf(w, `{"status":"successful","data":{"id":"fee-mandate","customer":"fee-customer","reference":"fee-reference","amount":100000,"mandate_type":"emandate","debit_type":"variable","status":"approved","approved":true,"ready_to_debit":true,"start_date":%q,"end_date":%q}}`, start.Format(time.RFC3339), end.Add(22*time.Hour+59*time.Minute).Format(time.RFC3339))
		case "/v3/payments/mandates/fee-mandate/debit":
			if body["fee_bearer"] != "business" || body["reference"] != "fee-debit-reference" || body["narration"] != "Kredit seller fees Ref:fee-debit-reference" {
				t.Error("fee debit mixed with buyer repayment or charged provider costs to seller")
			}
			_, _ = fmt.Fprint(w, `{"status":"successful","data":{"status":"processing","reference":"fee-debit-reference","mandate":"fee-mandate","live_mode":false}}`)
		default:
			t.Error("unexpected fee endpoint", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	if err := c.ValidateFeeCustomer(billing.FeeCustomer{BusinessName: "Example Trading Limited", BVN: "bad"}); err == nil {
		t.Fatal("invalid customer accepted before the submission fence")
	}
	ctx := context.Background()
	customer, err := c.CreateFeeCustomer(ctx, billing.FeeCustomer{BusinessName: "Example Trading Limited", Email: "owner@example.test", Phone: "08012345678", Address: "Synthetic address", BVN: "12345678901"})
	if err != nil || customer != "fee-customer" {
		t.Fatal(err)
	}
	a, err := c.CreateFeeAuthorization(ctx, billing.FeeAuthorization{Customer: customer, Reference: "fee-reference", Ceiling: 100000, StartsAt: start, EndsAt: end})
	if err != nil || a.ID != "fee-mandate" {
		t.Fatal(err)
	}
	checked, err := c.ReadFeeAuthorization(ctx, a.ID)
	if err != nil || !checked.Ready || checked.Customer != customer {
		t.Fatal("fee identity/readiness", checked, err)
	}
	if _, err = c.SubmitFeeDebit(ctx, billing.FeeDebitRequest{Mandate: a.ID, Reference: "fee-debit-reference", Amount: 30000}); err != nil {
		t.Fatal(err)
	}
}
