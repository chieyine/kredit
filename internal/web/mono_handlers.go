package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"kredit/internal/audit"
	"kredit/internal/collections"
	"kredit/internal/db"
	"kredit/internal/jobs"
	"kredit/internal/mandates"
	"kredit/internal/providers/mono"
)

func (s *Server) monoWebhook(w http.ResponseWriter, r *http.Request) {
	if s.runtime.Mono == nil || s.runtime.WebhookJobs == nil {
		writeProblem(w, 503, "provider_unavailable", "The bank collection connection is not configured.")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if err != nil {
		writeProblem(w, 400, "invalid_webhook", "webhook body could not be read")
		return
	}
	notice, err := s.runtime.Mono.ParseWebhook(r.Header.Get("mono-webhook-secret"), raw)
	if err != nil {
		writeProblem(w, 401, "webhook_rejected", "webhook authentication or payload is invalid")
		return
	}
	safe, _ := json.Marshal(notice)
	err = s.runtime.WebhookJobs.EnqueueProviderWebhook(r.Context(), jobs.ProviderWebhookArgs{Provider: "mono-sweep", EventID: notice.EventID, EventType: notice.Type, Payload: safe, SignatureValid: true})
	if err != nil {
		writeProblem(w, 503, "webhook_unavailable", "webhook could not be saved; retry delivery")
		return
	}
	writeJSON(w, 200, map[string]bool{"received": true})
}

// HandleProviderNotice treats callbacks as a reconciliation signal. It never
// posts the webhook's amount directly; the server-to-server lookup is authoritative.
func (r *Runtime) HandleProviderNotice(ctx context.Context, args jobs.ProviderWebhookArgs) error {
	if args.Provider != "mono-sweep" || r.Mono == nil || r.Database == nil || !args.SignatureValid {
		return errors.New("unsupported provider notice")
	}
	var notice mono.Notice
	if json.Unmarshal(args.Payload, &notice) != nil {
		return errors.New("invalid provider notice")
	}
	if strings.Contains(notice.Type, "debit_attempt") {
		return nil
	} // Aggregate final event/periodic lookup owns financial effect.
	if strings.Contains(notice.Type, ".debit.") {
		var id, organizationID string
		if err := r.Database.Raw().QueryRow(ctx, `SELECT attempt_id::text,organization_id::text FROM app.collection_attempt_identity_by_external($1)`, notice.Reference).Scan(&id, &organizationID); err != nil {
			return errors.New("debit attempt is not yet available")
		}
		ctx = db.WithTenantContext(ctx, "", organizationID)
		var attempt collections.Attempt
		var ok bool
		if scoped, supported := r.Collections.(interface {
			GetAttemptContext(context.Context, string) (collections.Attempt, bool)
		}); supported {
			attempt, ok = scoped.GetAttemptContext(ctx, id)
		} else {
			attempt, ok = r.Collections.GetAttempt(id)
		}
		if !ok || attempt.MandateReference != notice.MandateID {
			return errors.New("debit mandate does not match the reserved attempt")
		}
		_, err := r.Collections.Reconcile(ctx, id)
		return err
	}
	var mandate mandates.Mandate
	var err error
	if notice.BlockStatus != "" {
		blocker, ok := r.Mandates.(mandates.BlockingProvider)
		if !ok {
			return errors.New("mandate blocking is unavailable")
		}
		mandate, err = blocker.BlockMandate(ctx, notice.MandateID, notice.BlockStatus, notice.EventID)
	} else {
		mandate, err = r.Mandates.GetMandate(ctx, notice.MandateID)
	}
	if err != nil {
		return err
	}
	views, err := r.readCreditForBuyer(ctx, mandate.UserID)
	if err != nil {
		return err
	}
	for _, view := range views {
		if view.Mandate != nil && view.Mandate.ProviderID == notice.MandateID {
			if _, err = r.Credit.SetMandate(view.Request.ID, mandate.UserID, mandate); err != nil {
				return err
			}
		}
	}
	if r.TradeLines != nil {
		lines, err := r.readTradeLinesForBuyer(mandate.UserID)
		if err != nil {
			return err
		}
		active := mandate.Status == mandates.Active
		for _, line := range lines {
			if line.MandateID != mandate.ID || line.MandateActive == active {
				continue
			}
			if _, err := r.TradeLines.SetMandateState(line.ID, mandate.ID, active); err != nil {
				return err
			}
		}
	}
	r.Audit.Append(audit.Event{ActorUserID: "provider", Action: "mandate.provider_verified", ResourceType: "payment_mandate", ResourceID: mandate.ID, Outcome: string(mandate.Status)})
	return nil
}

func (s *Server) createRepaymentCustomer(w http.ResponseWriter, r *http.Request) {
	session, user, ok := s.requireAuth(w, r)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	if s.runtime.Mono == nil || s.runtime.Database == nil {
		writeProblem(w, 503, "provider_unavailable", "The bank collection connection is not configured.")
		return
	}
	businessID, _ := pathID(r, "businessID")
	tx, err := s.runtime.Database.Raw().Begin(r.Context())
	if err != nil {
		writeProblem(w, 503, "database_unavailable", "database is unavailable")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err = tx.Exec(r.Context(), `SELECT set_config('app.current_user_id',$1,true)`, user.ID); err != nil {
		writeProblem(w, 503, "database_unavailable", "database is unavailable")
		return
	}
	var owned bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app.businesses WHERE id=$1::uuid AND owner_user_id=$2::uuid)`, businessID, user.ID).Scan(&owned); err != nil || !owned {
		writeProblem(w, 404, "business_not_found", "business was not found")
		return
	}
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,82420))`, businessID); err != nil {
		writeProblem(w, 503, "database_unavailable", "database is unavailable")
		return
	}
	var exists bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app.provider_customer_bindings WHERE provider='mono-sweep' AND buyer_business_id=$1::uuid)`, businessID).Scan(&exists); err != nil {
		writeProblem(w, 503, "database_unavailable", "database is unavailable")
		return
	}
	if exists {
		writeJSON(w, 200, map[string]bool{"registered": true})
		return
	}
	var in mono.CustomerInput
	if err = decodeJSON(w, r, &in); err != nil {
		writeProblem(w, 400, "invalid_customer", "customer details could not be read")
		return
	}
	in, err = mono.ValidateCustomerInput(in)
	if err != nil {
		writeProblem(w, 400, "invalid_customer", "Complete customer details and consent are required.")
		return
	}
	var pending bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app.customer_registration_attempts WHERE business_id=$1 AND state='PENDING')`, businessID).Scan(&pending); err != nil {
		writeProblem(w, 503, "registration_unavailable", "Registration could not be checked.")
		return
	}
	if pending {
		writeProblem(w, 503, "registration_unconfirmed", "An earlier registration needs review in the admin bank connection panel before another attempt.")
		return
	}
	var attemptID string
	if err = tx.QueryRow(r.Context(), `INSERT INTO app.customer_registration_attempts(business_id,user_id,identity_fingerprint,consent_version) VALUES($1,$2,$3,$4) RETURNING id::text`, businessID, user.ID, s.registrationFingerprint(in.BVN), in.ConsentVersion).Scan(&attemptID); err != nil {
		writeProblem(w, 503, "registration_unavailable", "Registration intent could not be saved.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeProblem(w, 503, "registration_unconfirmed", "Registration intent needs review before retrying.")
		return
	}
	providerCtx, providerCancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer providerCancel()
	reference, err := s.runtime.Mono.CreateCustomer(providerCtx, in)
	if err != nil {
		writeProblem(w, 503, "registration_unconfirmed", "The provider result is unconfirmed. Review this attempt in the admin bank connection panel before retrying.")
		return
	}
	completionCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 15*time.Second)
	defer cancel()
	tx, err = s.runtime.Database.Raw().Begin(completionCtx)
	if err != nil {
		writeProblem(w, 503, "registration_unconfirmed", "Registration needs reconciliation.")
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(completionCtx, `SELECT set_config('app.current_user_id',$1,true)`, user.ID); err != nil {
		writeProblem(w, 503, "registration_unconfirmed", "Registration needs reconciliation.")
		return
	}
	var owner string
	if err = tx.QueryRow(completionCtx, `SELECT owner_user_id::text FROM app.businesses WHERE id=$1 FOR SHARE`, businessID).Scan(&owner); err != nil || owner != user.ID {
		writeProblem(w, 503, "registration_unconfirmed", "Business ownership must be reconciled before registration can be attached.")
		return
	}
	if _, err = tx.Exec(completionCtx, `INSERT INTO app.provider_customer_bindings(provider,buyer_user_id,buyer_business_id,provider_customer_reference,consent_version) VALUES('mono-sweep',$1::uuid,$2::uuid,$3,$4)`, user.ID, businessID, reference, in.ConsentVersion); err != nil {
		writeProblem(w, 503, "registration_unconfirmed", "customer registration needs reconciliation")
		return
	}
	var completedID string
	if err = tx.QueryRow(completionCtx, `UPDATE app.customer_registration_attempts SET state='CONFIRMED',provider_reference=$2,resolved_at=now() WHERE id=$1 AND state='PENDING' RETURNING id::text`, attemptID, reference).Scan(&completedID); err != nil {
		writeProblem(w, 503, "registration_unconfirmed", "Registration needs reconciliation.")
		return
	}
	if err = tx.Commit(completionCtx); err != nil {
		writeProblem(w, 503, "registration_unconfirmed", "customer registration needs reconciliation")
		return
	}
	s.runtime.Audit.Append(audit.Event{ActorUserID: user.ID, Action: "repayment.customer_registered", ResourceType: "business", ResourceID: businessID, Outcome: "success"})
	writeJSON(w, 201, map[string]bool{"registered": true})
}
