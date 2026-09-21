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
	"kredit/internal/providers/paystack"
)

func paystackEmailLookup(database *db.Pool) paystack.EmailLookup {
	return func(ctx context.Context, user string) (string, error) {
		tx, err := database.Raw().Begin(ctx)
		if err != nil {
			return "", err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, user); err != nil {
			return "", err
		}
		var email string
		err = tx.QueryRow(ctx, `SELECT COALESCE(normalized_email,'') FROM app.users WHERE id=$1::uuid AND status='active'`, user).Scan(&email)
		if err != nil {
			return "", err
		}
		return email, tx.Commit(ctx)
	}
}
func (s *Server) paystackWebhook(w http.ResponseWriter, r *http.Request) {
	client := s.runtime.PaystackAccounts[r.PathValue("account")]
	if client == nil || s.runtime.WebhookJobs == nil {
		writeProblem(w, 503, "provider_unavailable", "This collection account is not configured.")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if err != nil {
		writeProblem(w, 400, "invalid_webhook", "Could not read event.")
		return
	}
	notice, err := client.ParseWebhook(r.Header.Get("x-paystack-signature"), raw)
	if err != nil {
		writeProblem(w, 401, "webhook_rejected", "Invalid provider event.")
		return
	}
	if notice.EventID != "" {
		payload, _ := json.Marshal(notice)
		if err = s.runtime.WebhookJobs.EnqueueProviderWebhook(r.Context(), jobs.ProviderWebhookArgs{Provider: client.Name(), EventID: notice.EventID, EventType: notice.Type, Payload: payload, SignatureValid: true}); err != nil {
			writeProblem(w, 503, "webhook_unavailable", "Could not save event; retry delivery.")
			return
		}
	}
	writeJSON(w, 200, map[string]bool{"received": true})
}
func (r *Runtime) handlePaystackNotice(ctx context.Context, args jobs.ProviderWebhookArgs) error {
	if r.Database == nil || !args.SignatureValid {
		return errors.New("unverified provider notice")
	}
	var notice paystack.Notice
	if json.Unmarshal(args.Payload, &notice) != nil || notice.Type != "charge.success" || notice.Reference == "" {
		return errors.New("invalid payment notice")
	}
	var id, org string
	if err := r.Database.Raw().QueryRow(ctx, `SELECT attempt_id::text,organization_id::text FROM app.collection_attempt_identity_by_external($1)`, notice.Reference).Scan(&id, &org); err != nil {
		return errors.New("collection attempt not yet available")
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
		return errors.New("payment belongs to another collection account")
	}
	_, err := r.Collections.Reconcile(ctx, id)
	return err
}
