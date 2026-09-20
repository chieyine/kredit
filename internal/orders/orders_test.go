package orders

import (
	"context"
	"testing"
)

func TestOrderLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	orderID := "order-test-1"
	orgID := "org-supplier-1"
	seller := "user-seller-1"
	buyer := "user-buyer-1"
	reviewer := "user-finance-1"

	// 1. Line items
	items := []LineItem{
		{SKU: "CEMENT-50KG", Description: "Dangote 3X 50kg cement", UnitPriceKobo: 750000, Quantity: 100},
		{SKU: "REBAR-12MM", Description: "12mm High tensile rebar 12m", UnitPriceKobo: 1200000, Quantity: 20},
	}
	if err := store.CreateLineItems(ctx, orderID, items); err != nil {
		t.Fatalf("create line items: %v", err)
	}

	loaded, err := store.ListLineItems(ctx, orderID)
	if err != nil || len(loaded) != 2 {
		t.Fatalf("expected 2 line items, got %d (err: %v)", len(loaded), err)
	}
	if loaded[0].TotalKobo != 75000000 {
		t.Fatalf("expected 75000000 kobo, got %d", loaded[0].TotalKobo)
	}

	// 2. Partial shipment dispatch
	shipment, err := store.DispatchShipment(ctx, DispatchInput{
		OrderID:                orderID,
		SupplierOrganizationID: orgID,
		TrackingReference:      "TRK-9901",
		Carrier:                "Kredit Logistics Fleet",
		DispatchedBy:           seller,
		Items: []ShipmentItem{
			{LineItemID: loaded[0].ID, Quantity: 50},
		},
	})
	if err != nil {
		t.Fatalf("dispatch shipment: %v", err)
	}
	if shipment.Status != "in_transit" {
		t.Fatalf("expected in_transit status, got %s", shipment.Status)
	}

	// Verify line item fulfilled quantity was updated
	loadedAfter, _ := store.ListLineItems(ctx, orderID)
	if loadedAfter[0].FulfilledQuantity != 50 {
		t.Fatalf("expected fulfilled quantity 50, got %d", loadedAfter[0].FulfilledQuantity)
	}

	// 3. Delivery receipt
	receipt, err := store.RecordDeliveryReceipt(ctx, DeliveryInput{
		ShipmentID:     shipment.ID,
		OrderID:        orderID,
		ReceivedBy:     buyer,
		ConditionNotes: "50 bags received in good condition at site warehouse",
		SignedProof:    "SIG-BUYER-PROOF-KEY-9912",
	})
	if err != nil {
		t.Fatalf("record receipt: %v", err)
	}
	if receipt.SignedProofHash == "" {
		t.Fatal("expected signed proof hash")
	}

	// 4. Approved credit notes (dual control)
	cn, err := store.CreateCreditNote(ctx, CreditNoteInput{
		OrderID:                orderID,
		SupplierOrganizationID: orgID,
		AmountKobo:             1500000,
		Reason:                 "Partial price rebate for volume promotion",
		IssuedBy:               seller,
	})
	if err != nil {
		t.Fatalf("create credit note: %v", err)
	}
	if cn.Status != "draft" {
		t.Fatalf("expected draft status, got %s", cn.Status)
	}

	// Issuer cannot self-approve
	if err := store.ApproveCreditNote(ctx, cn.ID, seller); err != ErrDualControl {
		t.Fatalf("expected ErrDualControl for self-approval, got %v", err)
	}

	// Independent reviewer approves
	if err := store.ApproveCreditNote(ctx, cn.ID, reviewer); err != nil {
		t.Fatalf("approve credit note: %v", err)
	}

	notes, err := store.ListCreditNotes(ctx, orderID)
	if err != nil || len(notes) != 1 {
		t.Fatalf("expected 1 credit note, got %d", len(notes))
	}
	if notes[0].Status != "approved" || notes[0].ApprovedBy != reviewer {
		t.Fatalf("expected approved status by reviewer, got %+v", notes[0])
	}
}
