package buyers

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EnrollPurchasingStaff records the staff member's own notices and establishes
// pending verification subjects. A delegation is never treated as KYC evidence.
func (s *PostgresStore) EnrollPurchasingStaff(ctx context.Context, user, profile string, in AcceptInput) (Portal, error) {
	if !in.ConsentsAccepted || strings.TrimSpace(in.FullName) == "" || len(in.FullName) > 200 || in.TermsVersion == "" || in.PrivacyVersion == "" || in.IdentityNoticeVersion != IdentityNoticeVersion {
		return Portal{}, errors.New("current notices and full name required")
	}
	tx, err := s.beginTxContext(ctx, user, "")
	if err != nil {
		return Portal{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT app.lock_purchase_permission($1::uuid,'read',0)`, profile); err != nil {
		return Portal{}, err
	}
	var person string
	if _, err = tx.Exec(ctx, `INSERT INTO app.persons(id,user_id,full_name,status) VALUES($1::uuid,$2::uuid,$3,'pending_verification') ON CONFLICT(user_id) DO NOTHING`, uuid.NewString(), user, strings.TrimSpace(in.FullName)); err != nil {
		return Portal{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT id::text FROM app.persons WHERE user_id=$1::uuid FOR UPDATE`, user).Scan(&person); err != nil {
		return Portal{}, err
	}
	var representative string
	err = tx.QueryRow(ctx, `SELECT id::text FROM app.business_representatives WHERE business_id=$1::uuid AND person_id=$2::uuid AND authority_verification_status IN ('pending','verified') ORDER BY created_at DESC LIMIT 1`, profile, person).Scan(&representative)
	if errors.Is(err, pgx.ErrNoRows) {
		representative = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO app.business_representatives(id,business_id,person_id,role_title,authority_type,authority_verification_status) VALUES($1::uuid,$2::uuid,$3::uuid,'Delegated purchaser','business_staff','pending')`, representative, profile, person)
	}
	if err != nil {
		return Portal{}, err
	}
	provider := "unavailable-identity"
	if s.identity != nil {
		provider = s.identity.Name()
	}
	for kind, subject := range map[string]string{"person": person, "authority": representative} {
		if _, err = tx.Exec(ctx, `INSERT INTO app.buyer_verification_intents(user_id,subject_type,subject_id,provider) VALUES($1::uuid,$2,$3::uuid,$4) ON CONFLICT(subject_type,subject_id,provider) DO NOTHING`, user, kind, subject, provider); err != nil {
			return Portal{}, err
		}
	}
	for kind, version := range map[string]string{"buyer_portal": in.TermsVersion, "privacy_notice": in.PrivacyVersion, "identity_verification": in.IdentityNoticeVersion} {
		if _, err = tx.Exec(ctx, `INSERT INTO app.identity_consents(user_id,consent_type,version) SELECT $1::uuid,$2,$3 WHERE NOT EXISTS(SELECT 1 FROM app.identity_consents WHERE user_id=$1::uuid AND consent_type=$2 AND version=$3)`, user, kind, version); err != nil {
			return Portal{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Portal{}, err
	}
	return s.ReadBusinessPortal(ctx, user, profile)
}
