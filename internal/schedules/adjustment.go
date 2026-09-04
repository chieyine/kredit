package schedules

import (
	"errors"
	"kredit/internal/ledger"
)

// ReducePrincipal removes forgiven principal from the latest unpaid items.
// apply must use the caller's financial lock and must not reenter this store.
// Items are committed only after the authoritative balance update succeeds.
func (s *Store) ReducePrincipal(obligationID string, outstanding, amount ledger.Money, resolvingDispute bool, apply func() error) error {
	if s.pool != nil {
		return errors.New("durable schedule adjustments require a transaction")
	}
	if amount <= 0 || amount > outstanding || apply == nil {
		return errors.New("invalid principal adjustment")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.byObligation[obligationID]
	if id == "" {
		return errors.New("schedule not found")
	}
	items := cloneItems(s.items[id])
	var total ledger.Money
	for _, item := range items {
		if item.State == ItemCancelled {
			continue
		}
		if item.DisputedKobo > 0 && !resolvingDispute {
			return errors.New("resolve disputed instalments before adjusting principal")
		}
		var err error
		total, err = ledger.CheckedAdd(total, item.PrincipalDueKobo-item.AllocatedKobo)
		if err != nil {
			return err
		}
	}
	if total != outstanding {
		return errors.New("schedule and outstanding balance disagree; reconcile before adjusting")
	}
	remaining := amount
	for i := len(items) - 1; i >= 0 && remaining > 0; i-- {
		item := &items[i]
		if item.State == ItemCancelled {
			continue
		}
		take := min(remaining, item.PrincipalDueKobo-item.AllocatedKobo)
		if take == 0 {
			continue
		}
		next := item.PrincipalDueKobo - take
		if next == 0 {
			item.State = ItemCancelled
			item.DisputedKobo = 0
		} else {
			item.PrincipalDueKobo = next
			item.DisputedKobo = min(item.DisputedKobo, next-item.AllocatedKobo)
			if next == item.AllocatedKobo {
				item.State = ItemPaid
			}
		}
		remaining -= take
	}
	if remaining != 0 {
		return errors.New("adjustment exceeds remaining schedule")
	}
	if err := apply(); err != nil {
		return err
	}
	for i := range items {
		*s.items[id][i] = items[i]
	}
	return nil
}
