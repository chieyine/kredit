package identity

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"kredit/internal/access"
)

type NativeAction struct {
	Action          string `json:"action"`
	Value           string `json:"value"`
	ConsentVersion  string `json:"consent_version"`
	ExpectedVersion int64  `json:"expected_version"`
	Evidence        string `json:"evidence"`
}

// Act fences each external operation before sending. It never retries a saved
// unknown response automatically, and never persists a phone number, NIN or OTP.
func (p *NativeLookup) Act(ctx context.Context, id string, in NativeAction) error {
	if in.Action == "approve" || in.Action == "reject" || in.Action == "retry" {
		return p.review(ctx, id, in)
	}
	tx, err := p.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var kind, state, name, operation, reference string
	var version int64
	var attempts int
	var requiresReview bool
	var expiry *time.Time
	err = tx.QueryRow(ctx, `SELECT kind,state,full_name,operation,COALESCE(phone_reference,''),version,attempts,phone_expires_at,requires_review FROM app.native_identity_sessions WHERE id=$1::uuid AND provider=$2 AND user_id=app.current_user_id() FOR UPDATE`, id, p.name).Scan(&kind, &state, &name, &operation, &reference, &version, &attempts, &expiry, &requiresReview)
	if err != nil {
		return err
	}
	if version != in.ExpectedVersion {
		return errors.New("verification changed; refresh before continuing")
	}
	if in.Action == "renew" {
		tag, err := tx.Exec(ctx, `UPDATE app.native_identity_sessions SET state='pending',safe_result='{}',phone_reference=NULL,phone_expires_at=NULL,document_id=NULL,operation='',attempts=0,reviewed_by=NULL,review_evidence=NULL,expires_at=NULL,version=version+1,updated_at=now() WHERE id=$1::uuid AND state='verified' AND expires_at<=now()`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("only an expired successful check can be renewed")
		}
		return tx.Commit(ctx)
	}
	if state == "verified" || state == "failed" {
		return errors.New("this verification already has a final decision")
	}
	// An expired OTP can be replaced after its five-minute validity. No monetary
	// operation uses this rule. Unknown business results need evidence review.
	if operation != "" && (kind != "person" || expiry == nil || !time.Now().After(*expiry)) {
		return errors.New("the saved request needs confirmation before another request can be sent")
	}
	value := strings.TrimSpace(in.Value)
	switch in.Action {
	case "phone_start":
		if kind != "person" || in.ConsentVersion != NativeConsentVersion {
			return errors.New("accept the current identity lookup notice")
		}
		if !regexp.MustCompile(`^(0[789][01][0-9]{8}|\+?234[789][01][0-9]{8})$`).MatchString(value) {
			return errors.New("enter a valid Nigerian phone number linked to your NIN")
		}
		if expiry != nil && time.Now().Before(*expiry) {
			return errors.New("use the code already sent or wait for it to expire")
		}
	case "phone_verify":
		if kind != "person" || reference == "" || expiry == nil || !time.Now().Before(*expiry) || attempts >= 5 || !regexp.MustCompile(`^[0-9]{4,8}$`).MatchString(value) {
			return errors.New("enter a current code; request a new code after expiry")
		}
	case "business_lookup":
		if kind != "business" || in.ConsentVersion != NativeConsentVersion || !regexp.MustCompile(`^[A-Za-z0-9 /-]{2,40}$`).MatchString(value) {
			return errors.New("accept the lookup notice and enter the CAC registration number")
		}
	default:
		return errors.New("choose a supported verification action")
	}
	_, err = tx.Exec(ctx, `UPDATE app.native_identity_sessions SET operation=$2,state='in_progress',version=version+1,updated_at=now(),consent_version=CASE WHEN $2='phone_verify' THEN consent_version ELSE $3 END,consent_at=CASE WHEN $2='phone_verify' THEN consent_at ELSE now() END,phone_expires_at=CASE WHEN $2='phone_start' THEN now()+interval '5 minutes' ELSE phone_expires_at END,attempts=CASE WHEN $2='phone_start' THEN 0 WHEN $2='phone_verify' THEN attempts+1 ELSE attempts END WHERE id=$1::uuid`, id, in.Action, in.ConsentVersion)
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	result := map[string]string{}
	nextState := "pending"
	nextRef := reference
	nextExpiry := expiry
	switch in.Action {
	case "phone_start":
		var out struct {
			Status string `json:"status"`
			Data   struct {
				Reference string `json:"reference"`
				Expires   int    `json:"expires_in_seconds"`
			} `json:"data"`
		}
		err = p.request(ctx, http.MethodPost, "/v3/lookup/phone/initiate", map[string]string{"phone_number": value}, &out)
		if err == nil && (out.Status != "successful" || out.Data.Reference == "" || out.Data.Expires <= 0 || out.Data.Expires > 300) {
			err = errors.New("phone lookup did not confirm a valid code request")
		}
		if err == nil {
			nextRef = out.Data.Reference
			t := time.Now().UTC().Add(time.Duration(out.Data.Expires) * time.Second)
			nextExpiry = &t
		}
	case "phone_verify":
		var out struct {
			Status string `json:"status"`
			Data   struct {
				NIN    string `json:"nin"`
				First  string `json:"first_name"`
				Last   string `json:"last_name"`
				Middle string `json:"middle_name"`
			} `json:"data"`
		}
		err = p.request(ctx, http.MethodPost, "/v3/lookup/phone/verify", map[string]string{"reference": reference, "otp": value}, &out)
		if err == nil && (out.Status != "successful" || !regexp.MustCompile(`^[0-9]{11}$`).MatchString(out.Data.NIN)) {
			err = errors.New("the provider did not confirm identity; wait for the code to expire before starting again")
		}
		if err == nil {
			verifiedName := strings.Join(strings.Fields(out.Data.First+" "+out.Data.Middle+" "+out.Data.Last), " ")
			result = SafeVerificationResult(map[string]string{"verified_name": verifiedName, "nin_status": "verified"})
			nextState = "review"
			if normalName(name) == normalName(verifiedName) {
				nextState = "verified"
			}
		}
	case "business_lookup":
		var out struct {
			Status string `json:"status"`
			Data   []struct {
				RC       string `json:"rc_number"`
				Name     string `json:"approved_name"`
				Active   bool   `json:"active"`
				Approved bool   `json:"registration_approved"`
			} `json:"data"`
		}
		err = p.request(ctx, http.MethodGet, "/v3/lookup/cac?exact=true&search="+url.QueryEscape(value), nil, &out)
		if err == nil && out.Status != "successful" {
			err = errors.New("the provider did not confirm the business lookup")
		}
		if err == nil {
			nextState = "review"
			matches := 0
			for _, b := range out.Data {
				if normalRegistration(b.RC) == normalRegistration(value) && b.Active && b.Approved {
					matches++
					result = SafeVerificationResult(map[string]string{"verified_name": b.Name, "cac_status": "verified"})
					if normalName(name) == normalName(b.Name) && !requiresReview {
						nextState = "verified"
					}
				}
			}
			if matches != 1 {
				result = map[string]string{"cac_status": "review"}
				nextState = "review"
			}
		}
	}
	if err != nil {
		return err
	}
	tx, err = p.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var validUntil *time.Time
	if nextState == "verified" {
		t := time.Now().UTC().AddDate(1, 0, 0)
		validUntil = &t
	}
	tag, err := tx.Exec(ctx, `UPDATE app.native_identity_sessions SET state=$2,safe_result=$3,phone_reference=NULLIF($4,''),phone_expires_at=$5,operation='',expires_at=$6,version=version+1,updated_at=now() WHERE id=$1::uuid AND version=$7 AND operation=$8`, id, nextState, result, nextRef, nextExpiry, validUntil, version+1, in.Action)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("verification changed while checking; refresh its saved status")
	}
	return tx.Commit(ctx)
}
func normalName(s string) string { return strings.ToUpper(strings.Join(strings.Fields(s), " ")) }
func normalRegistration(s string) string {
	s = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(s), " ", ""), "-", ""))
	for _, prefix := range []string{"RC", "BN", "IT"} {
		s = strings.TrimPrefix(s, prefix)
	}
	return s
}
func (p *NativeLookup) review(ctx context.Context, id string, in NativeAction) error {
	if len(strings.TrimSpace(in.Evidence)) < 20 || len(in.Evidence) > 2000 {
		return errors.New("record the authority document or reviewed identity evidence, including its reference")
	}
	tx, err := p.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, _ := ctx.Value(actorKey{}).(string)
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionReviewCompliance); err != nil {
		return err
	}
	var kind, state, operation, owner string
	var version int64
	var safe map[string]string
	var person, business *string
	err = tx.QueryRow(ctx, `SELECT kind,state,operation,version,safe_result,person_id::text,business_id::text,user_id::text FROM app.native_identity_sessions WHERE id=$1::uuid AND provider=$2 FOR UPDATE`, id, p.name).Scan(&kind, &state, &operation, &version, &safe, &person, &business, &owner)
	if err != nil {
		return err
	}
	if in.Action == "retry" && version == in.ExpectedVersion && state == "failed" {
		_, err = tx.Exec(ctx, `UPDATE app.native_identity_sessions SET state='pending',safe_result='{}',document_id=NULL,phone_reference=NULL,phone_expires_at=NULL,operation='',attempts=0,consent_version=NULL,consent_at=NULL,expires_at=NULL,reviewed_by=$2::uuid,review_evidence=$3,version=version+1,updated_at=now() WHERE id=$1::uuid`, id, actor, strings.TrimSpace(in.Evidence))
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if in.Action == "retry" && version == in.ExpectedVersion && state != "verified" && state != "failed" {
		_, err = tx.Exec(ctx, `UPDATE app.native_identity_sessions SET operation='',state='review',reviewed_by=$2::uuid,review_evidence=$3,version=version+1,updated_at=now() WHERE id=$1::uuid`, id, actor, strings.TrimSpace(in.Evidence))
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if version != in.ExpectedVersion || state == "verified" || state == "failed" || operation != "" {
		return errors.New("refresh the request; only a completed check awaiting review can be decided")
	}
	if in.Action == "approve" {
		if kind == "person" && safe["nin_status"] != "verified" {
			return errors.New("successful OTP identity evidence is required")
		}
		if kind == "business" && safe["cac_status"] != "verified" {
			return errors.New("confirmed active CAC registration evidence is required")
		}
		if kind == "authority" {
			if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, owner); err != nil {
				return err
			}
			var clean bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.native_identity_sessions n JOIN app.documents d ON d.id=n.document_id WHERE n.id=$1::uuid AND d.purpose='identity_'||n.id::text AND d.uploaded_by=n.user_id AND d.scan_state='CLEAN' AND d.upload_completed_at IS NOT NULL)`, id).Scan(&clean); err != nil {
				return err
			}
			if !clean {
				return errors.New("a scanned authority document must be attached before approval")
			}
			var current bool
			if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, owner); err != nil {
				return err
			}
			err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT state='verified' AND verification_level>=2 AND expires_at>now() FROM app.verification_cases WHERE subject_type='person' AND subject_id=$1::uuid ORDER BY started_at DESC,id DESC LIMIT 1),false) AND COALESCE((SELECT state='verified' AND verification_level>=2 AND expires_at>now() FROM app.verification_cases WHERE subject_type='business' AND subject_id=$2::uuid ORDER BY started_at DESC,id DESC LIMIT 1),false)`, person, business).Scan(&current)
			if _, resetErr := tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); resetErr != nil {
				return resetErr
			}
			if err != nil {
				return err
			}
			if !current {
				return errors.New("refresh successful person and business checks before reviewing authority")
			}
		}
	}
	decision := "verified"
	if in.Action == "reject" {
		decision = "failed"
	}
	_, err = tx.Exec(ctx, `UPDATE app.native_identity_sessions SET state=$2,reviewed_by=$3::uuid,review_evidence=$4,expires_at=now()+interval '1 year',version=version+1,updated_at=now() WHERE id=$1::uuid`, id, decision, actor, strings.TrimSpace(in.Evidence))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
