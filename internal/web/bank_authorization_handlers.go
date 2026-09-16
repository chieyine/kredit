package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"kredit/internal/collections"
	"kredit/internal/db"
	"kredit/internal/jobs"
	"kredit/internal/mandates"
	"kredit/internal/providers/bankdebit"
	"kredit/internal/settlement"
)

func (s *Server) buyerBankEnrollment(w http.ResponseWriter, r *http.Request) {
	_, user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if r.Method == "POST" && !s.requireCSRF(w, r) {
		return
	}
	current, _, found, err := s.findBuyerMandate(r.Context(), user.ID, r.PathValue("reference"))
	if financialReadError(w, err) {
		return
	}
	if !found {
		writeProblem(w, 404, "authorization_not_found", "This bank permission was not found.")
		return
	}
	client := s.runtime.NativeBankAccounts[current.Provider]
	if client == nil {
		writeProblem(w, 503, "provider_unavailable", "The original bank provider is not connected.")
		return
	}
	var enrollment bankdebit.Enrollment
	if r.Method == "POST" {
		if !s.config.RealCollections || current.Provider != s.config.CollectionProvider {
			writeProblem(w, 409, "authorizations_paused", "New permissions are paused on this provider. You can still check or cancel an existing permission.")
			return
		}
		if current.Status == mandates.Cancelled || current.Status == mandates.Expired {
			writeProblem(w, 409, "permission_closed", "Start fresh bank permission from your sale.")
			return
		}
		var details bankdebit.Details
		if decodeJSON(w, r, &details) != nil {
			writeProblem(w, 400, "invalid_request", "Check your bank details.")
			return
		}
		if err = details.Validate(); err != nil {
			writeProblem(w, 400, "invalid_request", err.Error())
			return
		}
		enrollment, err = client.CompleteEnrollment(r.Context(), current.ProviderID, user.ID, details)
	} else if reader, supported := client.(interface {
		Instructions(context.Context, string) (bankdebit.Enrollment, error)
	}); supported {
		enrollment, err = reader.Instructions(r.Context(), current.ProviderID)
	} else {
		enrollment, err = client.Enrollment(r.Context(), current.ProviderID)
	}
	if err != nil {
		writeProblem(w, 409, "authorization_unconfirmed", "We could not confirm the bank response. Refresh this page before doing anything else; do not start another authorization.")
		return
	}
	if enrollment.Input.UserID != user.ID {
		writeProblem(w, 404, "authorization_not_found", "This bank permission was not found.")
		return
	}
	var banks []settlement.Bank
	if enrollment.State == "DRAFT" {
		banks, err = client.Banks(r.Context())
		if err != nil {
			writeProblem(w, 503, "banks_unavailable", "The bank list is unavailable. Please try again shortly.")
			return
		}
	}
	writeJSON(w, 200, map[string]any{"enrollment": enrollment, "mandate": current, "banks": banks})
}
func (s *Server) nativeBankWebhook(w http.ResponseWriter, r *http.Request) {
	client := s.runtime.NativeBankAccounts[r.PathValue("account")]
	if client == nil || s.runtime.WebhookJobs == nil {
		writeProblem(w, 503, "provider_unavailable", "This collection account is not configured.")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if err != nil {
		writeProblem(w, 400, "invalid_event", "Could not read event.")
		return
	}
	notice, err := client.ParseNotice(r.Header, raw)
	if err != nil {
		writeProblem(w, 401, "invalid_event", "Invalid provider event.")
		return
	}
	if notice.EventID != "" {
		payload, _ := json.Marshal(notice)
		if err = s.runtime.WebhookJobs.EnqueueProviderWebhook(r.Context(), jobs.ProviderWebhookArgs{Provider: client.Name(), EventID: notice.EventID, EventType: notice.Type, Payload: payload, SignatureValid: true}); err != nil {
			writeProblem(w, 503, "event_unavailable", "Could not save event; retry delivery.")
			return
		}
	}
	writeJSON(w, 200, map[string]bool{"received": true})
}
func (r *Runtime) handleNativeBankNotice(ctx context.Context, args jobs.ProviderWebhookArgs) error {
	client := r.NativeBankAccounts[args.Provider]
	if client == nil || r.Database == nil || !args.SignatureValid {
		return errors.New("unverified native bank event")
	}
	var notice bankdebit.Notice
	if json.Unmarshal(args.Payload, &notice) != nil {
		return errors.New("invalid native bank notice")
	}
	if notice.PaymentReference != "" {
		var id, org string
		if err := r.Database.Raw().QueryRow(ctx, `SELECT attempt_id::text,organization_id::text FROM app.collection_attempt_identity_by_external($1)`, notice.PaymentReference).Scan(&id, &org); err != nil {
			return errors.New("payment attempt not yet available")
		}
		ctx = db.WithTenantContext(ctx, "", org)
		reader, ok := r.Collections.(interface {
			GetAttemptContext(context.Context, string) (collections.Attempt, bool)
		})
		if !ok {
			return errors.New("durable collection reader unavailable")
		}
		attempt, ok := reader.GetAttemptContext(ctx, id)
		if !ok || attempt.Provider != args.Provider {
			return errors.New("payment belongs to another provider account")
		}
		_, err := r.Collections.Reconcile(ctx, id)
		return err
	}
	if notice.MandateReference != "" {
		ref, err := client.LocalReference(ctx, notice.MandateReference)
		if err != nil {
			return err
		}
		m, err := r.Mandates.GetMandate(mandates.WithProvider(ctx, args.Provider), ref)
		if err != nil {
			return err
		}
		return r.applyVerifiedMandate(ctx, m, notice.EventID)
	}
	return errors.New("provider event has no reconciliation reference")
}
