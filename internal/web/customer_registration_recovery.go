package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"kredit/internal/access"
)

func (s *Server) registrationFingerprint(identity string) string {
	mac := hmac.New(sha256.New, []byte(s.config.TokenHashKey))
	_, _ = mac.Write([]byte("customer-registration-identity:" + identity))
	return hex.EncodeToString(mac.Sum(nil))
}
func (s *Server) listCustomerRegistrations(w http.ResponseWriter, r *http.Request) {
	_, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	tx, err := s.runtime.Database.Raw().Begin(ctx)
	if err != nil {
		writeProblem(w, 503, "registrations_unavailable", "Registrations could not be loaded.")
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, user.ID); err != nil {
		writeProblem(w, 503, "registrations_unavailable", "Registrations could not be loaded.")
		return
	}
	rows, err := tx.Query(ctx, `SELECT id::text,business_id::text,created_at FROM app.customer_registration_attempts WHERE state='PENDING' ORDER BY created_at,id LIMIT 100`)
	if err != nil {
		writeProblem(w, 503, "registrations_unavailable", "Registrations could not be loaded.")
		return
	}
	type item struct {
		ID         string    `json:"id"`
		BusinessID string    `json:"business_id"`
		CreatedAt  time.Time `json:"created_at"`
	}
	items := []item{}
	for rows.Next() {
		var value item
		if err = rows.Scan(&value.ID, &value.BusinessID, &value.CreatedAt); err != nil {
			rows.Close()
			writeProblem(w, 503, "registrations_unavailable", "Registrations could not be loaded.")
			return
		}
		items = append(items, value)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeProblem(w, 503, "registrations_unavailable", "Registrations could not be loaded.")
		return
	}
	writeJSON(w, 200, map[string]any{"registrations": items})
}
func (s *Server) resolveCustomerRegistration(w http.ResponseWriter, r *http.Request) {
	session, user, _, ok := s.requirePlatformAccess(w, r, access.PermissionProviderOperations)
	if !ok || !s.requireFreshMFA(w, session) || !s.requireCSRF(w, r) {
		return
	}
	var in struct {
		Action    string `json:"action"`
		Reference string `json:"provider_reference"`
		Reason    string `json:"reason"`
	}
	if decodeJSON(w, r, &in) != nil || len(strings.TrimSpace(in.Reason)) < 20 || len(in.Reason) > 2000 || (in.Action != "link" && in.Action != "not_created") {
		writeProblem(w, 400, "invalid_resolution", "Choose an action and record at least 20 characters explaining the provider evidence checked.")
		return
	}
	in.Reference = strings.TrimSpace(in.Reference)
	if len(in.Reference) > 128 {
		writeProblem(w, 400, "invalid_reference", "Provider reference is too long.")
		return
	}
	if in.Action == "not_created" {
		in.Reference = ""
	}
	if s.runtime.Mono == nil && in.Action == "link" {
		writeProblem(w, 503, "provider_unavailable", "Connect Mono before verifying a customer reference.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	tx, err := s.runtime.Database.Raw().Begin(ctx)
	if err != nil {
		writeProblem(w, 503, "resolution_unavailable", "The result could not be saved.")
		return
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = access.LockPlatformAuthority(ctx, tx, user.ID, access.PermissionProviderOperations); err != nil {
		writeProblem(w, 403, "resolution_forbidden", "Your authority could not be verified.")
		return
	}
	id, _ := pathID(r, "attemptID")
	var business, buyer, fingerprint, consent, state, existingRef string
	if err = tx.QueryRow(ctx, `SELECT business_id::text,user_id::text,identity_fingerprint,consent_version,state,COALESCE(provider_reference,'') FROM app.customer_registration_attempts WHERE id=$1 FOR UPDATE`, id).Scan(&business, &buyer, &fingerprint, &consent, &state, &existingRef); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeProblem(w, 404, "registration_not_found", "Registration was not found.")
		} else {
			writeProblem(w, 503, "resolution_unavailable", "Registration could not be loaded. Try again.")
		}
		return
	}
	if state != "PENDING" {
		if (state == "CONFIRMED" && in.Action == "link" && existingRef == in.Reference) || (state == "NOT_CREATED" && in.Action == "not_created") {
			writeJSON(w, 200, map[string]bool{"resolved": true})
			return
		}
		writeProblem(w, 409, "registration_changed", "This registration has already been resolved differently.")
		return
	}
	var mature bool
	if err = tx.QueryRow(ctx, `SELECT created_at<now()-interval '2 minutes' FROM app.customer_registration_attempts WHERE id=$1`, id).Scan(&mature); err != nil {
		writeProblem(w, 503, "resolution_unavailable", "Registration could not be loaded. Try again.")
		return
	}
	if !mature {
		writeProblem(w, 409, "registration_in_progress", "Wait two minutes for the original request to finish before resolving it.")
		return
	}
	if in.Action == "link" {
		identity, lookupErr := s.runtime.Mono.CustomerIdentity(ctx, strings.TrimSpace(in.Reference))
		if lookupErr != nil {
			writeProblem(w, 409, "identity_unconfirmed", "The provider reference could not be verified against the original identity.")
			return
		}
		actual := s.registrationFingerprint(identity)
		if !hmac.Equal([]byte(actual), []byte(fingerprint)) {
			writeProblem(w, 409, "identity_mismatch", "The customer reference belongs to a different identity.")
			return
		}
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, buyer); err != nil {
			writeProblem(w, 503, "resolution_unavailable", "The result could not be saved.")
			return
		}
		var owner string
		if err = tx.QueryRow(ctx, `SELECT owner_user_id::text FROM app.businesses WHERE id=$1 FOR SHARE`, business).Scan(&owner); err != nil || owner != buyer {
			writeProblem(w, 409, "ownership_changed", "Business ownership has changed and needs separate review.")
			return
		}
		if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, buyer); err != nil {
			writeProblem(w, 503, "resolution_unavailable", "The result could not be saved.")
			return
		}
		if _, err = tx.Exec(ctx, `INSERT INTO app.provider_customer_bindings(provider,buyer_user_id,buyer_business_id,provider_customer_reference,consent_version) VALUES('mono-sweep',$1,$2,$3,$4)`, buyer, business, in.Reference, consent); err != nil {
			writeProblem(w, 409, "binding_conflict", "An existing registration must be reviewed before attaching this reference.")
			return
		}
		state = "CONFIRMED"
	} else {
		state = "NOT_CREATED"
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, user.ID); err != nil {
		writeProblem(w, 503, "resolution_unavailable", "The result could not be saved.")
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE app.customer_registration_attempts SET state=$2,provider_reference=NULLIF($3,''),resolved_by=$4,resolution_note=$5,resolved_at=now() WHERE id=$1`, id, state, in.Reference, user.ID, in.Reason); err != nil {
		writeProblem(w, 503, "resolution_unavailable", "The result could not be saved.")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeProblem(w, 503, "resolution_unconfirmed", "Check the registration before retrying this resolution.")
		return
	}
	writeJSON(w, 200, map[string]bool{"resolved": true})
}
