package buyers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func invitationFingerprint(input CreateInvitationInput) []byte {
	encoded, _ := json.Marshal(input)
	digest := sha256.Sum256(encoded)
	return digest[:]
}

// The caller holds the source-reference transaction lock. Tokens are only
// returned inside the already-authorized supplier invitation boundary.
func (s *PostgresStore) replayImport(ctx context.Context, tx pgx.Tx, organizationID string, input CreateInvitationInput, acceptedReplay bool) (CreateInvitationResult, bool, error) {
	var result CreateInvitationResult
	var fingerprint, encrypted []byte
	i := &result.Invitation
	err := tx.QueryRow(ctx, `SELECT id::text,organization_id::text,target_type,proposed_legal_name,COALESCE(proposed_trading_name,''),proposed_business_type,proposed_address,proposed_industry,status,expires_at,created_at,source_fingerprint,token_ciphertext FROM app.buyer_invitations WHERE organization_id=$1::uuid AND source_reference=$2`, organizationID, input.SourceReference).Scan(&i.ID, &i.OrganizationID, &i.TargetType, &i.ProposedLegalName, &i.ProposedTradingName, &i.ProposedBusinessType, &i.ProposedAddress, &i.ProposedIndustry, &i.Status, &i.ExpiresAt, &i.CreatedAt, &fingerprint, &encrypted)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, false, nil
	}
	if err != nil {
		return result, false, err
	}
	if !bytes.Equal(fingerprint, invitationFingerprint(input)) {
		return result, true, errors.New("this import reference already belongs to different customer details")
	}
	if acceptedReplay && i.Status == "accepted" {
		result.Replayed = true
		return result, true, nil
	}
	if i.Status != "pending" || !i.ExpiresAt.After(time.Now()) {
		return result, true, errors.New("the imported invitation is already accepted, revoked or expired; review its existing record")
	}
	token, err := s.decrypt(encrypted)
	if err != nil {
		return result, true, err
	}
	result.RawToken = string(token)
	result.Replayed = true
	return result, true, nil
}
