package web

import (
	"kredit/internal/access"
	"kredit/internal/consumer"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s *Server) consumerSales(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if s.runtime.Database == nil {
		writeProblem(w, 503, "consumer_unavailable", "Consumer purchases are unavailable.")
		return
	}
	store := &consumer.Store{Pool: s.runtime.Database.Raw()}
	org := r.PathValue("organizationID")
	id := r.PathValue("saleID")
	admin := strings.HasPrefix(r.URL.Path, "/api/v1/ops/")
	if org != "" {
		if _, e := pathID(r, "organizationID"); e != nil {
			writeProblem(w, 400, "invalid_business", "Invalid business.")
			return
		}
	}
	if id != "" {
		if _, e := pathID(r, "saleID"); e != nil {
			writeProblem(w, 400, "invalid_purchase", "Invalid purchase.")
			return
		}
	}
	if admin {
		if _, _, _, ok = s.requirePlatformAccess(w, r, access.PermissionPlatformOwner); !ok {
			return
		}
	}
	if r.Method == "POST" {
		if !s.requireCSRF(w, r) {
			return
		}
		if (org != "" || admin) && !s.requireFreshMFA(w, session) {
			return
		}
	}
	if strings.HasSuffix(r.URL.Path, "/consumer-bank") {
		var in struct {
			AccountNumber string `json:"account_number"`
		}
		if !decodeJSONRequest(w, r, &in) {
			return
		}
		if e := store.ConnectBank(r.Context(), user.ID, org, s.config.SettingsEncryptionKey, in.AccountNumber); e != nil {
			writeProblem(w, 409, "consumer_bank_unconfirmed", e.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"saved": true})
		return
	}
	if strings.HasSuffix(r.URL.Path, "/consumer-settings") {
		if !admin {
			writeProblem(w, 403, "owner_required", "Super-admin authority is required.")
			return
		}
		var in *consumer.Settings
		if r.Method == "POST" {
			in = &consumer.Settings{}
			if !decodeJSONRequest(w, r, in) {
				return
			}
		}
		result, e := store.Settings(r.Context(), user.ID, org, in)
		if e != nil {
			writeProblem(w, 409, "consumer_settings_unavailable", e.Error())
			return
		}
		writeJSON(w, 200, result)
		return
	}
	if r.Method == "GET" {
		if id != "" {
			result, e := store.Get(r.Context(), user.ID, org, id, admin)
			if e != nil {
				writeProblem(w, 404, "purchase_unavailable", "This purchase is unavailable for your account.")
				return
			}
			writeJSON(w, 200, result)
		} else {
			before := r.URL.Query().Get("before")
			if before != "" {
				if _, err := uuid.Parse(before); err != nil {
					writeProblem(w, 400, "invalid_cursor", "Invalid page reference.")
					return
				}
			}
			result, e := store.List(r.Context(), user.ID, org, admin, before)
			if e != nil {
				writeProblem(w, 503, "purchases_unavailable", "Purchases could not be loaded.")
				return
			}
			next := ""
			if len(result) == 100 {
				next = result[len(result)-1].ID
			}
			writeJSON(w, 200, map[string]any{"items": result, "next_cursor": next})
		}
		return
	}
	if id == "" {
		if org == "" || admin {
			writeProblem(w, 403, "retailer_required", "Create purchases from your retailer account.")
			return
		}
		var in consumer.Input
		if !decodeJSONRequest(w, r, &in) {
			return
		}
		result, e := store.Create(r.Context(), user.ID, org, in)
		if e != nil {
			writeProblem(w, 422, "purchase_invalid", e.Error())
			return
		}
		writeJSON(w, 201, result)
		return
	}
	var in consumer.Action
	if !decodeJSONRequest(w, r, &in) {
		return
	}
	result, e := store.Act(r.Context(), user.ID, org, id, admin, in)
	if e != nil {
		writeProblem(w, 409, "purchase_action_unconfirmed", e.Error())
		return
	}
	writeJSON(w, 200, result)
}
