package web

import (
	"kredit/internal/buyers"
	"kredit/internal/orders"
	"sync"
)

// Services belong to one server, not one HTTP request or a process-global map.
// Lazy construction also supports the existing explicit-runtime test fixtures.
type domainServices struct {
	once   sync.Once
	orders orders.Service
	terms  buyers.TermsImportService
}

func (s *Server) initDomainServices() {
	s.domainServices.once.Do(func() {
		if s.domainServices.orders == nil {
			if s.runtime.Database != nil {
				s.domainServices.orders = orders.NewPostgresStore(s.runtime.Database.Raw(), s.runtime.Ledger)
			} else {
				s.domainServices.orders = orders.NewMemoryStore()
			}
		}
		if s.domainServices.terms == nil {
			if s.runtime.Database != nil {
				s.domainServices.terms = buyers.NewPostgresTermsImportStore(s.runtime.Database.Raw(), s.runtime.Ledger)
			} else {
				s.domainServices.terms = buyers.NewMemoryTermsImportStore(s.runtime.Ledger)
			}
		}
	})
}
