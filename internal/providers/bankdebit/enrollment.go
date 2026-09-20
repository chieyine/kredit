// Package bankdebit holds the durable customer-authorization flow shared by
// native collectors that require bank details before returning consent steps.
package bankdebit

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/mandates"
	"kredit/internal/platformsettings"
)

type Details struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	Consent       bool   `json:"consent"`
}

func (d Details) Validate() error {
	email, e := mail.ParseAddress(d.Email)
	if e != nil || email.Address != d.Email || len(d.Name) < 2 || len(d.Name) > 100 || len(d.Address) < 5 || len(d.Address) > 100 || !regexp.MustCompile(`^\+?[0-9]{10,15}$`).MatchString(d.Phone) || !regexp.MustCompile(`^[0-9]{3,6}$`).MatchString(d.BankCode) || !regexp.MustCompile(`^[0-9]{10}$`).MatchString(d.AccountNumber) || !d.Consent {
		return errors.New("enter your name, email, phone, address and bank details, and accept the bank permission")
	}
	return nil
}

type ConsentTransfer struct {
	BankName      string    `json:"bank_name"`
	AccountName   string    `json:"account_name"`
	AccountNumber string    `json:"account_number"`
	AmountKobo    int64     `json:"amount_kobo"`
	ExpiresAt     time.Time `json:"expires_at"`
}
type Result struct {
	Reference        string           `json:"reference"`
	AuthorizationURL string           `json:"authorization_url,omitempty"`
	Transfer         *ConsentTransfer `json:"transfer,omitempty"`
}
type Enrollment struct {
	Version   int64                       `json:"version"`
	Provider  string                      `json:"provider"`
	Reference string                      `json:"reference"`
	State     string                      `json:"state"`
	Input     mandates.AuthorizationInput `json:"-"`
	Details   Details                     `json:"-"`
	Result    Result                      `json:"result"`
}
type Store struct {
	pool   *pgxpool.Pool
	crypto *platformsettings.Encryptor
}

func NewStore(pool *pgxpool.Pool, key string) *Store {
	return &Store{pool: pool, crypto: platformsettings.NewEncryptor(key)}
}
func (s *Store) tx(ctx context.Context, user string) (pgx.Tx, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true)`, user); e != nil {
		tx.Rollback(ctx)
		return nil, e
	}
	return tx, nil
}
func (s *Store) Create(ctx context.Context, provider string, in mandates.AuthorizationInput) (mandates.Mandate, error) {
	if !s.crypto.Ready() || in.Reference == "" || in.UserID == "" || in.AmountCeiling <= 0 {
		return mandates.Mandate{}, errors.New("encrypted durable bank authorization is unavailable")
	}
	tx, e := s.tx(ctx, in.UserID)
	if e != nil {
		return mandates.Mandate{}, e
	}
	defer tx.Rollback(ctx)
	b, _ := json.Marshal(in)
	_, e = tx.Exec(ctx, `INSERT INTO app.bank_debit_enrollments(provider,reference,user_id,input) VALUES($1,$2,$3::uuid,$4::jsonb)`, provider, in.Reference, in.UserID, b)
	if e != nil {
		return mandates.Mandate{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return mandates.Mandate{}, e
	}
	return mandates.Mandate{Provider: provider, ProviderID: in.Reference, Reference: in.Reference, Status: mandates.Pending, AmountCeiling: in.AmountCeiling, Variable: true, AuthorizationURL: "/workspace/purchases/bank-authorization/" + in.Reference}, nil
}
func (s *Store) Load(ctx context.Context, provider, ref string) (Enrollment, error) {
	var owner string
	if e := s.pool.QueryRow(ctx, `SELECT buyer_user_id FROM app.payment_mandate_by_provider($1,$2)`, provider, ref).Scan(&owner); e != nil {
		return Enrollment{}, errors.New("bank authorization not found")
	}
	tx, e := s.tx(ctx, owner)
	if e != nil {
		return Enrollment{}, e
	}
	defer tx.Rollback(ctx)
	v, e := s.read(ctx, tx, provider, ref, false)
	if e != nil {
		return v, e
	}
	return v, tx.Commit(ctx)
}
func (s *Store) read(ctx context.Context, tx pgx.Tx, provider, ref string, lock bool) (Enrollment, error) {
	v := Enrollment{Provider: provider, Reference: ref}
	var in, out []byte
	var cipher string
	query := `SELECT input,state,details_ciphertext,result,version FROM app.bank_debit_enrollments WHERE provider=$1 AND reference=$2`
	if lock {
		query += " FOR UPDATE"
	}
	if e := tx.QueryRow(ctx, query, provider, ref).Scan(&in, &v.State, &cipher, &out, &v.Version); e != nil {
		return v, e
	}
	if e := json.Unmarshal(in, &v.Input); e != nil {
		return v, e
	}
	if e := json.Unmarshal(out, &v.Result); e != nil {
		return v, e
	}
	if cipher != "" {
		plain, e := s.crypto.Decrypt("bank-debit:"+provider+":"+ref, cipher)
		if e != nil {
			return v, e
		}
		if e = json.Unmarshal([]byte(plain), &v.Details); e != nil {
			return v, e
		}
	}
	return v, nil
}

// Begin commits the uncertain outcome fence before the first external request.
// Repeated clicks, reconnects and restarts can never submit a second tokenization.
func (s *Store) Begin(ctx context.Context, provider, ref, user string, d Details) (Enrollment, error) {
	if e := d.Validate(); e != nil {
		return Enrollment{}, e
	}
	tx, e := s.tx(ctx, user)
	if e != nil {
		return Enrollment{}, e
	}
	defer tx.Rollback(ctx)
	v, e := s.read(ctx, tx, provider, ref, true)
	if e != nil {
		return v, e
	}
	if v.Input.UserID != user || v.State != "DRAFT" {
		return v, errors.New("this bank authorization has already started; check its status before trying again")
	}
	plain, _ := json.Marshal(d)
	cipher, e := s.crypto.Encrypt("bank-debit:"+provider+":"+ref, string(plain))
	if e != nil {
		return v, e
	}
	_, e = tx.Exec(ctx, `UPDATE app.bank_debit_enrollments SET state='STARTED',details_ciphertext=$3,version=version+1,updated_at=now() WHERE provider=$1 AND reference=$2`, provider, ref, cipher)
	if e != nil {
		return v, e
	}
	v.State = "STARTED"
	v.Version++
	v.Details = d
	return v, tx.Commit(ctx)
}
func (s *Store) Confirm(ctx context.Context, v Enrollment, result Result) error {
	if strings.TrimSpace(result.Reference) == "" {
		return errors.New("provider authorization reference missing")
	}
	tx, e := s.tx(ctx, v.Input.UserID)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	b, _ := json.Marshal(result)
	tag, e := tx.Exec(ctx, `UPDATE app.bank_debit_enrollments SET state='CONFIRMED',result=$3::jsonb,version=version+1,updated_at=now() WHERE provider=$1 AND reference=$2 AND state='STARTED' AND version=$4`, v.Provider, v.Reference, b, v.Version)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return errors.New("authorization state changed; reconciliation required")
	}
	return tx.Commit(ctx)
}

// CancelDraft only cancels a local session that has never contacted a provider.
func (s *Store) CancelDraft(ctx context.Context, v Enrollment) error {
	tx, e := s.tx(ctx, v.Input.UserID)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	tag, e := tx.Exec(ctx, `UPDATE app.bank_debit_enrollments SET state='CANCELLED',version=version+1,updated_at=now() WHERE provider=$1 AND reference=$2 AND state IN ('DRAFT','CANCELLED')`, v.Provider, v.Reference)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return errors.New("provider confirmation required before cancellation")
	}
	return tx.Commit(ctx)
}

type Enroller interface {
	mandates.Provider
	Enrollment(context.Context, string) (Enrollment, error)
	CompleteEnrollment(context.Context, string, string, Details) (Enrollment, error)
}

type RecoveryStore interface {
	Create(context.Context, string, mandates.AuthorizationInput) (mandates.Mandate, error)
	Load(context.Context, string, string) (Enrollment, error)
	Begin(context.Context, string, string, string, Details) (Enrollment, error)
	Confirm(context.Context, Enrollment, Result) error
	CancelDraft(context.Context, Enrollment) error
	Review(context.Context, string, string, string) (Enrollment, error)
	Resolve(context.Context, string, Enrollment, string, Result, string) error
}
