package credit

import (
	"errors"
)

func (s *Store) GetForSupplier(requestID, orgID string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.requests[requestID]
	if r == nil || r.SupplierOrganizationID != orgID {
		return View{}, ErrRequestNotFound
	}
	view := s.viewLocked(r)
	if view.Mandate != nil {
		view.Mandate.AuthorizationURL = ""
	}
	return view, nil
}
func (s *Store) GetForBuyer(requestID, buyerUserID string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.requests[requestID]
	if r == nil || !s.canPurchaseLocked(r, buyerUserID, "read") || r.State == Draft {
		return View{}, ErrRequestNotFound
	}
	return s.buyerViewLocked(r, buyerUserID), nil
}
func (s *Store) GetPublic(requestID string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.requests[requestID]
	if r == nil {
		return View{}, ErrRequestNotFound
	}
	return s.viewLocked(r), nil
}
func (s *Store) GetByObligationForBuyer(obligationID, buyerUserID string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.requests {
		if r.ObligationID == obligationID && s.canPurchaseLocked(r, buyerUserID, "read") {
			return s.buyerViewLocked(r, buyerUserID), nil
		}
	}
	return View{}, errors.New("obligation not found")
}
func (s *Store) ListForSupplier(orgID string) []View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []View{}
	for _, r := range s.requests {
		if r.SupplierOrganizationID == orgID {
			view := s.viewLocked(r)
			if view.Mandate != nil {
				view.Mandate.AuthorizationURL = ""
			}
			out = append(out, view)
		}
	}
	return out
}

// ListForBuyer returns the buyer's complete credit-request history. Callers
// should apply presentation rules (for example, only exposing a buyer's own
// relationship) at the API boundary.
func (s *Store) ListForBuyer(buyerUserID string) []View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []View{}
	for _, r := range s.requests {
		if s.canPurchaseLocked(r, buyerUserID, "read") && r.State != Draft {
			out = append(out, s.buyerViewLocked(r, buyerUserID))
		}
	}
	return out
}

func (s *Store) viewLocked(r *CreditRequest) View {
	v := View{Request: cloneRequest(*r), Receipts: []ReceiptConfirmation{}}
	if a := s.agreements[r.AgreementVersionID]; a != nil {
		v.Agreement = cloneAgreement(*a)
	}
	if a := s.acceptances[r.AcceptanceID]; a != nil {
		cp := *a
		v.Acceptance = &cp
	}
	if m := s.mandateMap[r.MandateID]; m != nil {
		cp := *m
		v.Mandate = &cp
	}
	if rel := s.releases[r.ReleaseID]; rel != nil {
		cp := *rel
		v.Release = &cp
	}
	for _, receipt := range s.receipts[r.ID] {
		v.Receipts = append(v.Receipts, *receipt)
	}
	if o := s.obligations[r.ObligationID]; o != nil {
		cp := *o
		v.Obligation = &cp
	}
	return v
}
