package orders

import (
	"context"
	"errors"
	"testing"
)

func TestAuditOrderMemoryRejectsMixedOrderAndPartialDispatch(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	for _, order := range []string{"order-a", "order-b"} {
		if err := s.CreateLineItems(ctx, order, []LineItem{{Description: "cartons", UnitPriceKobo: 100, Quantity: 10}}); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := s.ListLineItems(ctx, "order-a")
	b, _ := s.ListLineItems(ctx, "order-b")
	for _, items := range [][]ShipmentItem{
		{{LineItemID: b[0].ID, Quantity: 1}},
		{{LineItemID: a[0].ID, Quantity: 1}, {LineItemID: b[0].ID, Quantity: 1}},
		{{LineItemID: a[0].ID, Quantity: 1}, {LineItemID: a[0].ID, Quantity: 1}},
		{{LineItemID: a[0].ID, Quantity: 11}},
		{{LineItemID: a[0].ID, Quantity: -1}},
		{},
	} {
		if _, err := s.DispatchShipment(ctx, DispatchInput{OrderID: "order-a", SupplierOrganizationID: "supplier", DispatchedBy: "maker", Items: items}); err == nil {
			t.Fatal("invalid shipment accepted")
		}
		lines, _ := s.ListLineItems(ctx, "order-a")
		ships, _ := s.ListShipments(ctx, "order-a")
		if lines[0].FulfilledQuantity != 0 || len(ships) != 0 {
			t.Fatalf("partial dispatch committed: %+v %+v", lines, ships)
		}
	}
}

func TestAuditOrderMemoryReceiptReplayAndCopies(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	if err := s.CreateLineItems(ctx, "a", []LineItem{{Description: "cartons", Quantity: 10, UnitPriceKobo: 100}}); err != nil {
		t.Fatal(err)
	}
	lines, _ := s.ListLineItems(ctx, "a")
	ship, err := s.DispatchShipment(ctx, DispatchInput{OrderID: "a", SupplierOrganizationID: "supplier", DispatchedBy: "maker", Items: []ShipmentItem{{LineItemID: lines[0].ID, Quantity: 6}}})
	if err != nil {
		t.Fatal(err)
	}
	ship.Items[0].Quantity = 100
	ships, _ := s.ListShipments(ctx, "a")
	if ships[0].Items[0].Quantity != 6 {
		t.Fatal("returned shipment aliases store")
	}
	in := DeliveryInput{ShipmentID: ship.ID, OrderID: "b", ReceivedBy: "buyer", SignedProof: "confirmed cartons"}
	if _, err = s.RecordDeliveryReceipt(ctx, in); err == nil {
		t.Fatal("different order receipt accepted")
	}
	in.OrderID = "a"
	rec, err := s.RecordDeliveryReceipt(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.RecordDeliveryReceipt(ctx, in)
	if err != nil || again.ID != rec.ID {
		t.Fatalf("exact receipt retry: %+v %v", again, err)
	}
	in.SignedProof = "different evidence"
	if _, err = s.RecordDeliveryReceipt(ctx, in); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting receipt: %v", err)
	}
}

func TestAuditOrderInputsRejectOverflowAndMissingEvidence(t *testing.T) {
	for _, input := range []LineItem{
		{Description: "carton", UnitPriceKobo: 9007199254740991, Quantity: 2},
		{Description: "carton", UnitPriceKobo: -1, Quantity: 1},
		{Description: "carton", UnitPriceKobo: 1, Quantity: 0},
		{Description: " ", UnitPriceKobo: 1, Quantity: 1},
		{Description: "carton", UnitPriceKobo: 1, Quantity: 1, FulfilledQuantity: 1},
	} {
		if validateLineItems("a", []LineItem{input}) == nil {
			t.Fatalf("invalid money/quantity accepted: %+v", input)
		}
	}
	if validateDelivery(DeliveryInput{ShipmentID: "s", OrderID: "o", ReceivedBy: "b", SignedProof: " "}) == nil {
		t.Fatal("missing receipt evidence accepted")
	}
}
