package web

import (
	"kredit/internal/orders"
	"sync"
)

// Services belong to one server, not one HTTP request or a process-global map.
// Lazy construction also supports the existing explicit-runtime test fixtures.
type domainServices struct {
	once   sync.Once
	orders orders.Service
}

func (s *Server) initDomainServices() {
	s.once.Do(func() {
		if s.orders == nil {
			if s.runtime.Database != nil {
				s.orders = orders.NewPostgresStore(s.runtime.Database.Raw(), s.runtime.Ledger)
			} else {
				s.orders = orders.NewMemoryStore()
			}
		}
	})
}
