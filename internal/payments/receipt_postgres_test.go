package payments

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublicReceiptWithRuntimeRole(t *testing.T) {
	f := newPaymentFixture(t, 10000)
	payment, _, err := f.store().RecordContext(f.ctx, RecordInput{ObligationID: f.obligationID, AmountKobo: 1500, SourceType: SourceCashRecorded, Currency: "NGN", RecordedBy: f.userID, IdempotencyKey: f.paymentKeyPrefix + "-receipt"})
	if err != nil {
		t.Fatal(err)
	}
	cfg := f.pool.Config().Copy()
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewPostgresStore(pool, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err = store.GetContext(ctx, payment.ID); err == nil {
		t.Fatal("ordinary payment lookup must still require tenant context")
	}
	receipt, err := store.PublicReceiptContext(ctx, payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Reference != payment.ID || receipt.AmountKobo != 1500 || receipt.State != StateRecognized {
		t.Fatalf("incorrect receipt: %+v", receipt)
	}
	body, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{f.userID, f.organizationID, "provider_reference", "buyer_user_id", "recorded_by"} {
		if strings.Contains(string(body), private) {
			t.Fatalf("receipt exposes private field %s", private)
		}
	}
	if _, err = f.store().ReverseContext(f.ctx, payment.ID, f.userID, "receipt correction"); err != nil {
		t.Fatal(err)
	}
	receipt, err = store.PublicReceiptContext(ctx, payment.ID)
	if err != nil || receipt.State != StateReversed {
		t.Fatalf("receipt must reflect reversal: %+v, %v", receipt, err)
	}
}
