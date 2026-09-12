package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/access"
)

type actorKey struct{}

func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// NativeLookup keeps interactive verification local and durable. Mono supplies
// identity evidence; it never supplies Kredit's representative authorisation.
type NativeLookup struct {
	pool                  *pgxpool.Pool
	name, endpoint, token string
	client                *http.Client
}

func NewNativeLookup(pool *pgxpool.Pool, name, endpoint, token string) (*NativeLookup, error) {
	u, err := url.Parse(endpoint)
	if pool == nil || name == "" || token == "" || err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("mono lookup requires a database, account name, HTTPS address and secret key")
	}
	return &NativeLookup{pool: pool, name: name, endpoint: strings.TrimRight(endpoint, "/"), token: token, client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (p *NativeLookup) Name() string { return p.name }
func (p *NativeLookup) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{PersonVerification: true, BusinessVerification: true, AuthorityVerification: true}
}
func (p *NativeLookup) begin(ctx context.Context) (pgx.Tx, error) {
	actor, _ := ctx.Value(actorKey{}).(string)
	if actor == "" {
		return nil, errors.New("authenticated verification actor is required")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, actor); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
func (p *NativeLookup) create(ctx context.Context, kind, subject, name, person, business string, requireReview bool) (VerificationSession, error) {
	tx, err := p.begin(ctx)
	if err != nil {
		return VerificationSession{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, _ := ctx.Value(actorKey{}).(string)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO app.native_identity_sessions(user_id,provider,subject_id,kind,full_name,person_id,business_id,requires_review) VALUES($1::uuid,$2,$3::uuid,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,$8) ON CONFLICT(provider,kind,subject_id) DO NOTHING RETURNING id::text`, actor, p.name, subject, kind, strings.TrimSpace(name), person, business, requireReview).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT id::text FROM app.native_identity_sessions WHERE provider=$1 AND kind=$2 AND subject_id=$3::uuid AND full_name=$4 AND person_id IS NOT DISTINCT FROM NULLIF($5,'')::uuid AND business_id IS NOT DISTINCT FROM NULLIF($6,'')::uuid AND requires_review=$7`, p.name, kind, subject, strings.TrimSpace(name), person, business, requireReview).Scan(&id)
	}
	if err != nil {
		return VerificationSession{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return VerificationSession{}, err
	}
	v, err := p.GetVerification(ctx, id)
	return VerificationSession{Provider: p.name, ProviderID: id, State: v.State, SafeResult: v.SafeResult, VerificationLevel: v.VerificationLevel, ExpiresAt: v.ExpiresAt}, err
}
func (p *NativeLookup) CreatePersonVerification(ctx context.Context, in PersonVerificationInput) (VerificationSession, error) {
	return p.create(ctx, "person", in.SubjectID, in.FullName, "", "", false)
}
func (p *NativeLookup) CreateBusinessVerification(ctx context.Context, in BusinessVerificationInput) (VerificationSession, error) {
	return p.create(ctx, "business", in.SubjectID, in.LegalName, "", "", in.RequireReview)
}
func (p *NativeLookup) CreateAuthorityVerification(ctx context.Context, in AuthorityVerificationInput) (VerificationSession, error) {
	return p.create(ctx, "authority", in.SubjectID, in.RoleTitle, in.PersonID, in.BusinessID, true)
}
func (p *NativeLookup) GetVerification(ctx context.Context, id string) (ProviderVerification, error) {
	tx, err := p.begin(ctx)
	if err != nil {
		return ProviderVerification{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	v := ProviderVerification{ProviderID: id}
	var expiry *time.Time
	err = tx.QueryRow(ctx, `SELECT subject_id::text,state,safe_result,expires_at FROM app.native_identity_sessions WHERE id=$1::uuid AND provider=$2`, id, p.name).Scan(&v.SubjectID, &v.State, &v.SafeResult, &expiry)
	if expiry != nil {
		v.ExpiresAt = *expiry
		if time.Now().After(*expiry) {
			v.State = "expired"
		}
	}
	if v.State == "verified" {
		v.VerificationLevel = 2
	}
	return v, err
}
func (p *NativeLookup) VerifyWebhook(context.Context, http.Header, []byte) (VerifiedIdentityEvent, error) {
	return VerifiedIdentityEvent{}, errors.New("mono lookup uses authenticated responses, not identity callbacks")
}
func (p *NativeLookup) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.endpoint+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("mono-sec-key", p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		return errors.New("verification response was not confirmed; refresh the saved request")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("verification provider returned status %d", res.StatusCode)
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(output); err != nil {
		return errors.New("verification response could not be read")
	}
	return nil
}

const NativeConsentVersion = "mono-lookup-2026-09-12"

type NativeCase struct {
	ID         string            `json:"id"`
	Provider   string            `json:"provider"`
	Kind       string            `json:"kind"`
	State      string            `json:"state"`
	Name       string            `json:"name"`
	Operation  string            `json:"operation"`
	Version    int64             `json:"version"`
	SafeResult map[string]string `json:"safe_result"`
	DocumentID string            `json:"document_id"`
}

func (p *NativeLookup) List(ctx context.Context, review bool) ([]NativeCase, error) {
	tx, err := p.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	actor, _ := ctx.Value(actorKey{}).(string)
	if review {
		if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionReviewCompliance); err != nil {
			return nil, err
		}
	}
	rows, err := tx.Query(ctx, `SELECT id::text,provider,kind,CASE WHEN state='verified' AND expires_at<=now() THEN 'expired' ELSE state END,full_name,operation,version,safe_result,COALESCE(document_id::text,'') FROM app.native_identity_sessions WHERE provider=$1 AND ($2 OR user_id=app.current_user_id()) ORDER BY (state NOT IN ('verified','failed') OR expires_at<=now()) DESC,created_at DESC,id DESC`, p.name, review)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []NativeCase{}
	for rows.Next() {
		var v NativeCase
		if err = rows.Scan(&v.ID, &v.Provider, &v.Kind, &v.State, &v.Name, &v.Operation, &v.Version, &v.SafeResult, &v.DocumentID); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (p *NativeLookup) EvidenceTarget(ctx context.Context, id string, write bool) (owner, document string, err error) {
	tx, err := p.begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = tx.QueryRow(ctx, `SELECT user_id::text,COALESCE(document_id::text,'') FROM app.native_identity_sessions WHERE id=$1::uuid AND provider=$2 AND (NOT $3 OR (user_id=app.current_user_id() AND state NOT IN ('verified','failed')))`, id, p.name, write).Scan(&owner, &document)
	return
}
func (p *NativeLookup) AttachEvidence(ctx context.Context, id, document string) error {
	tx, err := p.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE app.native_identity_sessions n SET document_id=$2::uuid,version=version+1,updated_at=now() WHERE n.id=$1::uuid AND n.provider=$3 AND n.user_id=app.current_user_id() AND n.state NOT IN ('verified','failed') AND EXISTS(SELECT 1 FROM app.documents d WHERE d.id=$2::uuid AND d.uploaded_by=n.user_id AND d.organization_id IS NULL AND d.purpose='identity_'||n.id::text AND d.upload_completed_at IS NOT NULL)`, id, document, p.name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("the document does not belong to this open verification")
	}
	return tx.Commit(ctx)
}
