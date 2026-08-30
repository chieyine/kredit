package credit

import (
	"testing"
	"time"

	"kredit/internal/ledger"
)

func TestPostgresCreditStoreImplementsServiceAndFailsClosedWithoutDatabase(t *testing.T) {
	store := NewPostgresStore(nil, NewStore(nil, ledger.NewStore()))
	var _ Service = store
	if _, err := store.Create(CreateInput{SupplierOrganizationID: "org-1", SupplierLegalName: "Supplier", BuyerUserID: "buyer-1", BuyerBusinessID: "business-1", BuyerLegalName: "Buyer", PrincipalKobo: 100, Currency: "NGN", GoodsDescription: "goods", DueDate: "2026-09-01", CollectionAt: nowForTest()}); err == nil {
		t.Fatal("credit writes must fail closed without a database")
	}
}

func nowForTest() (value time.Time) { return time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC) }
