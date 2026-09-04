package collections

import (
	"context"
	"kredit/internal/ledger"
	"testing"
	"time"
)

func TestCollectionUsesAcceptedFeeTermsRatherThanCurrentDefaults(t *testing.T) {
	f := financialFixture(t, &ledger.FeeTerms{PolicyRevision: 10, BaseBPS: 25, CollectionBPS: 75})
	ctx := context.Background()
	p := &observedProvider{MockProvider: NewMockProvider("secret")}
	e := f.engine(p)
	a, err := e.Start(ctx, f.id, "recorded-fee:"+f.id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	event := Webhook{EventID: "recorded-fee-success:" + f.id, ExternalReference: a.ExternalReference, ProviderCollectionID: a.ProviderCollectionID, State: ProviderSucceeded, SucceededAmountKobo: 50000000}
	event.Signature = p.Sign(event)
	if _, err = e.ProcessWebhook(ctx, event); err != nil {
		t.Fatal(err)
	}
	var fee, rate int64
	if err = f.pool.QueryRow(ctx, `SELECT amount_kobo,rate_basis_points FROM app.fees WHERE obligation_id=$1::uuid`, f.id).Scan(&fee, &rate); err != nil {
		t.Fatal(err)
	}
	if fee != 375000 || rate != 75 {
		t.Fatalf("recorded fee lost: fee=%d rate=%d", fee, rate)
	}
	if _, err = f.pool.Exec(ctx, `UPDATE app.credit_requests SET fee_terms='{"policy_revision":11,"base_bps":50,"collection_bps":50}' WHERE id=$1::uuid`, f.request); err == nil {
		t.Fatal("existing fee terms were rewritten")
	}
}
