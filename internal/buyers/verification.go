package buyers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"kredit/internal/identity"
)

// RefreshVerification resumes saved provider references. STARTED without a
// reference is deliberately held for reconciliation: creating another session
// after an ambiguous response could duplicate identity processing.
func (s *PostgresStore) RefreshVerification(ctx context.Context, userID string) (Portal, error) {
	return s.RefreshBusinessVerification(ctx, userID, "")
}
func (s *PostgresStore) RefreshBusinessVerification(ctx context.Context, userID, businessID string) (Portal, error) {
	ctx = identity.WithActor(ctx, userID)
	portal, err := s.ReadBusinessPortal(ctx, userID, businessID)
	if err != nil {
		return Portal{}, err
	}
	if s.identity == nil {
		return portal, errors.New("identity verification is not configured")
	}
	subjects := []struct{ kind, id string }{{"person", portal.Person.ID}, {"business", portal.Business.ID}, {"authority", portal.Representative.ID}}
	for _, subject := range subjects {
		tx, err := s.beginTxContext(ctx, userID, "")
		if err != nil {
			return portal, err
		}
		var intentID, state, reference, providerName string
		var version int64
		err = tx.QueryRow(ctx, `SELECT id::text,state,COALESCE(provider_reference,''),version,provider FROM app.buyer_verification_intents WHERE user_id=$1::uuid AND subject_type=$2 AND subject_id=$3::uuid ORDER BY created_at DESC,id DESC LIMIT 1 FOR UPDATE`, userID, subject.kind, subject.id).Scan(&intentID, &state, &reference, &version, &providerName)
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			continue
		} // historical, already verified accounts
		if err != nil {
			_ = tx.Rollback(ctx)
			return portal, err
		}
		if providerName == "unavailable-identity" && state == "NEW" && reference == "" {
			if s.identity.Name() == "unavailable-identity" {
				_ = tx.Rollback(ctx)
				return portal, errors.New("identity verification is not yet configured")
			}
			providerName = s.identity.Name()
			err = tx.QueryRow(ctx, `INSERT INTO app.buyer_verification_intents(user_id,subject_type,subject_id,provider) VALUES($1::uuid,$2,$3::uuid,$4) ON CONFLICT(subject_type,subject_id,provider) DO UPDATE SET version=app.buyer_verification_intents.version RETURNING id::text,state,COALESCE(provider_reference,''),version`, userID, subject.kind, subject.id, providerName).Scan(&intentID, &state, &reference, &version)
			if err != nil {
				_ = tx.Rollback(ctx)
				return portal, err
			}
		}
		selected, selectErr := identity.Resolve(s.identity, providerName)
		if selectErr != nil {
			_ = tx.Rollback(ctx)
			return portal, selectErr
		}
		if state == "STARTED" {
			_ = tx.Rollback(ctx)
			return portal, errors.New("a verification request needs provider confirmation; contact support before starting another")
		}
		if state == "NEW" {
			err = tx.QueryRow(ctx, `UPDATE app.buyer_verification_intents SET state='STARTED',started_at=now(),version=version+1 WHERE id=$1::uuid RETURNING version`, intentID).Scan(&version)
			if err != nil {
				_ = tx.Rollback(ctx)
				return portal, err
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return portal, err
		}
		var session identity.VerificationSession
		if state == "NEW" {
			switch subject.kind {
			case "person":
				session, err = selected.CreatePersonVerification(ctx, identity.PersonVerificationInput{SubjectID: subject.id, FullName: portal.Person.FullName})
			case "business":
				session, err = selected.CreateBusinessVerification(ctx, identity.BusinessVerificationInput{SubjectID: subject.id, LegalName: portal.Business.LegalName, BusinessType: portal.Business.BusinessType, Address: portal.Business.BusinessAddress})
			case "authority":
				session, err = selected.CreateAuthorityVerification(ctx, identity.AuthorityVerificationInput{SubjectID: subject.id, PersonID: portal.Person.ID, BusinessID: portal.Business.ID, RoleTitle: portal.Representative.RoleTitle})
			}
			if err != nil {
				return portal, err
			}
		} else {
			var result identity.ProviderVerification
			result, err = selected.GetVerification(ctx, reference)
			if err != nil {
				return portal, err
			}
			if result.ProviderID != reference || result.SubjectID != subject.id {
				return portal, errors.New("verification response does not match this account")
			}
			session = identity.VerificationSession{Provider: selected.Name(), ProviderID: result.ProviderID, State: result.State, VerificationLevel: result.VerificationLevel, SafeResult: result.SafeResult, ExpiresAt: result.ExpiresAt}
		}
		if session.ProviderID == "" || session.Provider != selected.Name() {
			return portal, errors.New("verification response is incomplete")
		}
		session.State = strings.ToLower(session.State)
		tx, err = s.beginTxContext(ctx, userID, "")
		if err != nil {
			return portal, err
		}
		var currentVersion int64
		if err = tx.QueryRow(ctx, `SELECT version FROM app.buyer_verification_intents WHERE id=$1::uuid FOR UPDATE`, intentID).Scan(&currentVersion); err != nil {
			_ = tx.Rollback(ctx)
			return portal, err
		}
		if currentVersion != version {
			_ = tx.Rollback(ctx)
			return portal, errors.New("verification changed while checking; refresh its current status")
		}
		err = insertVerification(ctx, tx, subject.id, subject.kind, session, time.Now().UTC())
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE app.buyer_verification_intents SET state='CREATED',provider_reference=$2,version=version+1 WHERE id=$1::uuid AND (provider_reference IS NULL OR provider_reference=$2)`, intentID, session.ProviderID)
		}
		// All three verified, unexpired cases are required. Pending and failed cases
		// remain visible and cannot authorise accepting a sale.
		verified := verificationComplete(session) && (session.ExpiresAt.IsZero() || session.ExpiresAt.After(time.Now()))
		status := "pending_verification"
		if verified {
			status = "verified"
		}
		if err == nil {
			switch subject.kind {
			case "person":
				_, err = tx.Exec(ctx, `UPDATE app.persons SET status=$2 WHERE id=$1::uuid AND status IN ('invited','pending_verification','verified')`, subject.id, status)
			case "business":
				_, err = tx.Exec(ctx, `UPDATE app.businesses SET status=$2 WHERE id=$1::uuid AND status IN ('pending_verification','verified')`, subject.id, status)
			case "authority":
				status = "pending"
				if verified {
					status = "verified"
				}
				_, err = tx.Exec(ctx, `UPDATE app.business_representatives SET authority_verification_status=$2 WHERE id=$1::uuid AND authority_verification_status IN ('pending','verified')`, subject.id, status)
			}
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return portal, err
		}
		if err = tx.Commit(ctx); err != nil {
			return portal, err
		}
	}
	return s.ReadBusinessPortal(ctx, userID, businessID)
}

func (s *Store) RefreshVerification(ctx context.Context, userID string) (Portal, error) {
	return s.RefreshBusinessVerification(ctx, userID, "")
}
func (s *Store) RefreshBusinessVerification(ctx context.Context, userID, businessID string) (Portal, error) {
	s.acceptMu.Lock()
	defer s.acceptMu.Unlock()
	ctx = identity.WithActor(ctx, userID)
	portal, err := s.ReadBusinessPortal(ctx, userID, businessID)
	if err != nil {
		return Portal{}, err
	}
	if s.identity == nil {
		return portal, errors.New("identity verification is not configured")
	}
	caps := s.identity.Capabilities()
	if !caps.PersonVerification || !caps.BusinessVerification || !caps.AuthorityVerification {
		return portal, errors.New("identity verification is not configured")
	}
	for _, subject := range []struct{ kind, id string }{{"person", portal.Person.ID}, {"business", portal.Business.ID}, {"authority", portal.Representative.ID}} {
		s.mu.Lock()
		state, exists := s.verificationIntents[subject.id]
		if !exists {
			s.mu.Unlock()
			continue
		}
		if state == "STARTED" {
			s.mu.Unlock()
			return portal, errors.New("a verification request needs provider confirmation; contact support")
		}
		if state == "NEW" {
			s.verificationIntents[subject.id] = "STARTED"
		}
		s.mu.Unlock()
		var session identity.VerificationSession
		if state == "NEW" {
			switch subject.kind {
			case "person":
				session, err = s.identity.CreatePersonVerification(ctx, identity.PersonVerificationInput{SubjectID: subject.id, FullName: portal.Person.FullName})
			case "business":
				session, err = s.identity.CreateBusinessVerification(ctx, identity.BusinessVerificationInput{SubjectID: subject.id, LegalName: portal.Business.LegalName, BusinessType: portal.Business.BusinessType, Address: portal.Business.BusinessAddress})
			case "authority":
				session, err = s.identity.CreateAuthorityVerification(ctx, identity.AuthorityVerificationInput{SubjectID: subject.id, PersonID: portal.Person.ID, BusinessID: portal.Business.ID, RoleTitle: portal.Representative.RoleTitle})
			}
		} else {
			var result identity.ProviderVerification
			result, err = s.identity.GetVerification(ctx, state)
			if err == nil && (result.ProviderID != state || result.SubjectID != subject.id) {
				err = errors.New("verification response does not match this account")
			}
			session = identity.VerificationSession{Provider: s.identity.Name(), ProviderID: result.ProviderID, State: result.State, VerificationLevel: result.VerificationLevel, ExpiresAt: result.ExpiresAt, SafeResult: result.SafeResult}
		}
		if err != nil {
			return portal, err
		}
		if session.ProviderID == "" || session.Provider != s.identity.Name() {
			return portal, errors.New("verification response is incomplete")
		}
		s.mu.Lock()
		s.verificationIntents[subject.id] = session.ProviderID
		s.addVerificationLocked(subject.id, subject.kind, session)
		status := "pending_verification"
		verified := verificationComplete(session) && (session.ExpiresAt.IsZero() || session.ExpiresAt.After(s.now()))
		if verified {
			status = "verified"
		}
		switch subject.kind {
		case "person":
			s.persons[subject.id].Status = status
		case "business":
			s.businesses[subject.id].Status = status
		case "authority":
			if !verified {
				status = "pending"
			}
			s.representatives[subject.id].AuthorityStatus = status
		}
		s.mu.Unlock()
	}
	return s.ReadBusinessPortal(ctx, userID, businessID)
}

// VerificationCurrent checks the actual case evidence, including expiry. A
// cached profile status alone cannot keep an expired verification valid.
func VerificationCurrent(portal Portal, now time.Time) bool {
	if portal.Person.Status != "verified" || portal.Business.Status != "verified" || portal.Representative.AuthorityStatus != "verified" {
		return false
	}
	for kind, subject := range map[string]string{"person": portal.Person.ID, "business": portal.Business.ID, "authority": portal.Representative.ID} {
		var latest *VerificationCase
		for i := range portal.VerificationCases {
			item := &portal.VerificationCases[i]
			if item.SubjectID == subject && item.SubjectType == kind && (latest == nil || item.StartedAt.After(latest.StartedAt)) {
				latest = item
			}
		}
		if latest == nil || latest.State != "verified" || latest.VerificationLevel < 2 || (!latest.ExpiresAt.IsZero() && !now.Before(latest.ExpiresAt)) {
			return false
		}
	}
	return true
}
