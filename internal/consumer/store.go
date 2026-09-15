package consumer

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"kredit/internal/access"
	"kredit/internal/auth"
	"kredit/internal/ledger"
	"kredit/internal/settlement"
	"regexp"
	"strings"
	"time"
)

type Store struct{ Pool *pgxpool.Pool }

func (s *Store) begin(ctx context.Context, actor, org string) (pgx.Tx, error) {
	if s == nil || s.Pool == nil {
		return nil, errors.New("Consumer purchases require the database.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `SELECT set_config('app.current_user_id',$1,true),set_config('app.current_organization_id',$2,true)`, actor, org)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

type Settings struct {
	Enabled  bool   `json:"enabled"`
	Evidence string `json:"review_evidence"`
}

// Settings is now an exceptional restriction control, not a retailer approval queue.
func (s *Store) Settings(ctx context.Context, actor, org string, in *Settings) (Settings, error) {
	tx, e := s.begin(ctx, actor, org)
	if e != nil {
		return Settings{}, e
	}
	defer tx.Rollback(ctx)
	if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); e != nil {
		return Settings{}, e
	}
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('consumer-eligibility:'||$1,0))`, org); e != nil {
		return Settings{}, e
	}
	if in != nil {
		if len(strings.TrimSpace(in.Evidence)) < 20 || len(in.Evidence) > 2000 {
			return Settings{}, errors.New("Record the reason for restricting this retailer or removing a restriction.")
		}
		_, e = tx.Exec(ctx, `INSERT INTO app.consumer_restrictions(organization_id,blocked,reason,updated_by) VALUES($1::uuid,$2,$3,$4::uuid) ON CONFLICT(organization_id) DO UPDATE SET blocked=EXCLUDED.blocked,reason=EXCLUDED.reason,updated_by=EXCLUDED.updated_by,updated_at=now()`, org, !in.Enabled, in.Evidence, actor)
		if e != nil {
			return Settings{}, e
		}
		_, e = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'consumer.restriction','organization',$2,'success','high',jsonb_build_object('blocked',$3::boolean,'reason',$4::text))`, actor, org, !in.Enabled, in.Evidence)
		if e != nil {
			return Settings{}, e
		}
	}
	out := Settings{Enabled: true}
	e = tx.QueryRow(ctx, `SELECT NOT blocked,reason FROM app.consumer_restrictions WHERE organization_id=$1::uuid`, org).Scan(&out.Enabled, &out.Evidence)
	if errors.Is(e, pgx.ErrNoRows) {
		e = nil
	}
	if e != nil {
		return out, e
	}
	return out, tx.Commit(ctx)
}

// ConnectBank reuses the exact provider-registered account. Knowing its last
// four digits alone never authorises a different receiving account.
func (s *Store) ConnectBank(ctx context.Context, actor, org, key, account string) error {
	if len(key) < 32 || !regexp.MustCompile(`^[0-9]{10}$`).MatchString(account) {
		return errors.New("Enter the ten-digit receiving account already registered for your business.")
	}
	tx, e := s.begin(ctx, actor, org)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('consumer-eligibility:'||$1,0))`, org); e != nil {
		return e
	}
	if e = sellerPermission(ctx, tx, actor, org, access.PermissionManageFinancial); e != nil {
		return e
	}
	rows, e := tx.Query(ctx, `SELECT r.id,r.provider,r.connection_identity,r.bank_code,p.settlement_bank_name,r.result->>'account_name' FROM app.settlement_registrations r JOIN app.supplier_onboarding_profiles p ON p.organization_id=r.organization_id AND p.settlement_provider=r.provider AND p.settlement_provider_reference=r.result->>'provider_reference' WHERE r.organization_id=$1::uuid AND r.state='REGISTERED' FOR SHARE OF r,p`, org)
	if e != nil {
		return e
	}
	var matched, bank, name string
	for rows.Next() {
		var id, provider, connection, code, b, n string
		if e = rows.Scan(&id, &provider, &connection, &code, &b, &n); e != nil {
			rows.Close()
			return e
		}
		if hmac.Equal([]byte(id), []byte(settlement.RegistrationID(key, org, provider, connection, code, account))) {
			matched, bank, name = id, b, n
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if matched == "" {
		return errors.New("That account does not match your registered business bank account. Check your settlement settings.")
	}
	_, e = tx.Exec(ctx, `INSERT INTO app.consumer_settings(organization_id,enabled,bank_name,account_name,account_number,review_evidence,updated_by,registration_id) VALUES($1::uuid,true,$2,$3,$4,'Automatically matched to the original provider bank registration',$5::uuid,$6) ON CONFLICT(organization_id) DO UPDATE SET enabled=true,bank_name=EXCLUDED.bank_name,account_name=EXCLUDED.account_name,account_number=EXCLUDED.account_number,registration_id=EXCLUDED.registration_id,review_evidence=EXCLUDED.review_evidence,updated_by=EXCLUDED.updated_by,updated_at=now()`, org, bank, name, account, actor, matched)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'consumer.bank.matched','organization',$2,'success','high',jsonb_build_object('registration',$3::text))`, actor, org, matched)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func sellerPermission(ctx context.Context, tx pgx.Tx, actor, org string, p access.Permission) error {
	var role access.Role
	e := tx.QueryRow(ctx, `SELECT m.role FROM app.memberships m JOIN app.users u ON u.id=m.user_id WHERE m.organization_id=$1::uuid AND m.user_id=$2::uuid AND m.status='active' AND u.status='active' FOR SHARE OF m,u`, org, actor).Scan(&role)
	if e == nil && access.Can(role, p) {
		return nil
	}
	return errors.New("Your current business role cannot perform this action.")
}
func (s *Store) Create(ctx context.Context, actor, org string, in Input) (Sale, error) {
	tx, e := s.begin(ctx, actor, org)
	if e != nil {
		return Sale{}, e
	}
	defer tx.Rollback(ctx)
	var ready bool
	if e = tx.QueryRow(ctx, `SELECT app.consumer_retailer_ready($1::uuid)`, org).Scan(&ready); e != nil || !ready {
		return Sale{}, errors.New("Finish business onboarding and connect your registered receiving account. Consumer sales activate automatically unless a restriction applies.")
	}
	if e = sellerPermission(ctx, tx, actor, org, access.PermissionCreateCredit); e != nil {
		return Sale{}, e
	}
	in.Target = auth.NormalizeIdentifier(in.Target)
	if in.TargetType == "email" {
		if !regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`).MatchString(in.Target) {
			return Sale{}, errors.New("Enter the customer's email.")
		}
	} else if in.TargetType == "phone" {
		if !regexp.MustCompile(`^\+234[789][01][0-9]{8}$`).MatchString(in.Target) {
			return Sale{}, errors.New("Use the customer's WhatsApp number in +234 format.")
		}
	} else {
		return Sale{}, errors.New("Choose email or WhatsApp.")
	}
	if len(in.Target) > 254 {
		return Sale{}, errors.New("Customer contact is too long.")
	}
	var enabled bool
	e = tx.QueryRow(ctx, `SELECT c.enabled,c.bank_name,c.account_name,c.account_number,o.legal_name,o.business_address FROM app.consumer_settings c JOIN app.organizations o ON o.id=c.organization_id JOIN app.supplier_onboarding_profiles p ON p.organization_id=o.id WHERE c.organization_id=$1::uuid AND p.readiness_state='pilot_ready' AND p.kyb_state='approved' AND (p.kyb_expires_at IS NULL OR p.kyb_expires_at>now()) AND p.settlement_state='verified' AND p.billing_state='configured' FOR SHARE OF c,p`, org).Scan(&enabled, &in.Terms.BankName, &in.Terms.AccountName, &in.Terms.AccountNumber, &in.Terms.SellerName, &in.Terms.SellerAddress)
	if e != nil || !enabled {
		return Sale{}, errors.New("Finish business onboarding and connect your registered receiving account.")
	}
	terms, hash, e := Prepare(in.Terms, time.Now())
	if e != nil {
		return Sale{}, e
	}
	raw, _ := json.Marshal(terms)
	var id string
	var created time.Time
	e = tx.QueryRow(ctx, `INSERT INTO app.consumer_sales(organization_id,created_by,target_type,target_value,terms,agreement_hash) VALUES($1::uuid,$2::uuid,$3,$4,$5::jsonb,$6) RETURNING id::text,created_at`, org, actor, in.TargetType, in.Target, raw, hash).Scan(&id, &created)
	if e != nil {
		return Sale{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Sale{}, e
	}
	v := Sale{ID: id, OrganizationID: org, TargetType: in.TargetType, Target: in.Target, Terms: terms, Hash: hash, State: "offered", Version: 1, CreatedAt: created, Events: []Event{}, Role: "seller"}
	v.Calculate()
	return v, nil
}

const saleSelect = `SELECT id::text,organization_id::text,COALESCE(buyer_user_id::text,''),target_type,target_value,terms,agreement_hash,state,customer_name,delivery_address,accepted_at,released_at,received_at,case_state,version,created_at FROM app.consumer_sales`

func scan(row pgx.Row) (Sale, error) {
	var s Sale
	var raw []byte
	e := row.Scan(&s.ID, &s.OrganizationID, &s.BuyerID, &s.TargetType, &s.Target, &raw, &s.Hash, &s.State, &s.CustomerName, &s.Address, &s.AcceptedAt, &s.ReleasedAt, &s.ReceivedAt, &s.CaseState, &s.Version, &s.CreatedAt)
	if e == nil {
		e = json.Unmarshal(raw, &s.Terms)
	}
	return s, e
}
func history(ctx context.Context, tx pgx.Tx, s *Sale) error {
	rows, e := tx.Query(ctx, `SELECT id::text,action,amount_kobo,reference,COALESCE(related_id::text,''),note,occurred_at,created_at FROM app.consumer_events WHERE sale_id=$1::uuid ORDER BY created_at,id`, s.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	s.Events = []Event{}
	for rows.Next() {
		var ev Event
		if e = rows.Scan(&ev.ID, &ev.Action, &ev.Amount, &ev.Reference, &ev.RelatedID, &ev.Note, &ev.At, &ev.RecordedAt); e != nil {
			return e
		}
		s.Events = append(s.Events, ev)
	}
	s.Calculate()
	return rows.Err()
}
func (s *Store) Get(ctx context.Context, actor, org, id string, admin bool) (Sale, error) {
	tx, e := s.begin(ctx, actor, org)
	if e != nil {
		return Sale{}, e
	}
	defer tx.Rollback(ctx)
	v, e := scan(tx.QueryRow(ctx, saleSelect+` WHERE id=$1::uuid`, id))
	if e != nil {
		return v, ErrUnavailable
	}
	if admin {
		if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); e != nil {
			return Sale{}, e
		}
		v.Role = "admin"
	} else if org != "" {
		if org != v.OrganizationID {
			return Sale{}, ErrUnavailable
		}
		if e = sellerPermission(ctx, tx, actor, org, access.PermissionReadFinancial); e != nil {
			return Sale{}, e
		}
		v.Role = "seller"
	} else {
		var match bool
		e = tx.QueryRow(ctx, `SELECT ($1::text=$2 OR ($1='' AND app.consumer_contact_matches($3,$4)))`, v.BuyerID, actor, v.TargetType, v.Target).Scan(&match)
		if e != nil || !match {
			return Sale{}, ErrUnavailable
		}
		v.Role = "buyer"
	}
	e = history(ctx, tx, &v)
	return v, e
}
func (s *Store) List(ctx context.Context, actor, org string, admin bool, before string) ([]Sale, error) {
	tx, e := s.begin(ctx, actor, org)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	query := saleSelect + ` WHERE (buyer_user_id=$1::uuid OR (buyer_user_id IS NULL AND app.consumer_contact_matches(target_type,target_value))) AND ($2='' OR id<NULLIF($2,'')::uuid) ORDER BY id DESC LIMIT 100`
	args := []any{actor, before}
	if admin {
		if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); e != nil {
			return nil, e
		}
		query = saleSelect + ` WHERE ($1='' OR id<NULLIF($1,'')::uuid) ORDER BY id DESC LIMIT 100`
		args = []any{before}
	} else if org != "" {
		if e = sellerPermission(ctx, tx, actor, org, access.PermissionReadFinancial); e != nil {
			return nil, e
		}
		query = saleSelect + ` WHERE organization_id=$1::uuid AND ($2='' OR id<NULLIF($2,'')::uuid) ORDER BY id DESC LIMIT 100`
		args = []any{org, before}
	}
	rows, e := tx.Query(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	items := []Sale{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for i := range items {
		if e = history(ctx, tx, &items[i]); e != nil {
			return nil, e
		}
	}
	return items, nil
}
func (s *Store) Act(ctx context.Context, actor, org, id string, admin bool, in Action) (Sale, error) {
	tx, e := s.begin(ctx, actor, org)
	if e != nil {
		return Sale{}, e
	}
	defer tx.Rollback(ctx)
	if admin {
		if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); e != nil {
			return Sale{}, e
		}
	}
	v, e := scan(tx.QueryRow(ctx, saleSelect+` WHERE id=$1::uuid FOR UPDATE`, id))
	if e != nil {
		return Sale{}, ErrUnavailable
	}
	role := "buyer"
	if admin {
		if e = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionPlatformOwner); e != nil {
			return Sale{}, e
		}
		role = "admin"
	} else if org != "" {
		if org != v.OrganizationID {
			return Sale{}, ErrUnavailable
		}
		permission := access.PermissionManageFinancial
		if in.Action == "release" {
			permission = access.PermissionReleaseGoods
		}
		if e = sellerPermission(ctx, tx, actor, org, permission); e != nil {
			return Sale{}, e
		}
		role = "seller"
	} else {
		var match bool
		e = tx.QueryRow(ctx, `SELECT ($1::text=$2 OR ($1='' AND app.consumer_contact_matches($3,$4)))`, v.BuyerID, actor, v.TargetType, v.Target).Scan(&match)
		if e != nil || !match {
			return Sale{}, ErrUnavailable
		}
	}
	if _, e = tx.Exec(ctx, `SELECT set_config('app.current_organization_id',$1,true)`, v.OrganizationID); e != nil {
		return Sale{}, e
	}
	if v.Version != in.Version {
		return Sale{}, errors.New("This purchase changed. Refresh it before continuing.")
	}
	if e = history(ctx, tx, &v); e != nil {
		return Sale{}, e
	}
	if in.Action == "accept" {
		var ready bool
		e = tx.QueryRow(ctx, `SELECT app.consumer_retailer_ready($1::uuid)`, v.OrganizationID).Scan(&ready)
		if e != nil || !ready {
			return Sale{}, errors.New("This retailer must finish its current verification before you accept. Existing payments and refunds remain available.")
		}
	}
	before := v
	ev, e := v.Apply(actor, role, in, time.Now().UTC())
	if e != nil {
		return Sale{}, e
	}
	var eventID string
	e = tx.QueryRow(ctx, `INSERT INTO app.consumer_events(sale_id,actor_id,action,amount_kobo,reference,related_id,note,occurred_at) VALUES($1::uuid,$2::uuid,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8) RETURNING id::text,occurred_at,created_at`, id, actor, ev.Action, ev.Amount, ev.Reference, ev.RelatedID, ev.Note, ev.At).Scan(&eventID, &ev.At, &ev.RecordedAt)
	if e != nil {
		return Sale{}, errors.New("This payment reference or decision is already recorded, or could not be saved. Refresh before retrying.")
	}
	if e = ledger.NewPostgresStore(s.Pool).PostConsumerEventTx(ctx, tx, eventID, ev.Action, ledger.Money(ev.Amount), before.ReleasedAt != nil && before.State != "cancelled", ledger.Money(before.Terms.Total-before.Reduction), ledger.Money(before.Paid-before.Refunded), ev.At); e != nil {
		return Sale{}, e
	}
	_, e = tx.Exec(ctx, `UPDATE app.consumer_sales SET buyer_user_id=NULLIF($2,'')::uuid,state=$3,customer_name=$4,delivery_address=$5,accepted_at=$6,released_at=$7,received_at=$8,case_state=$9,version=version+1,updated_at=now() WHERE id=$1::uuid`, id, v.BuyerID, v.State, v.CustomerName, v.Address, v.AcceptedAt, v.ReleasedAt, v.ReceivedAt, v.CaseState)
	if e != nil {
		return Sale{}, e
	}
	if e = notice(ctx, tx, v, eventID, in.Action); e != nil {
		return Sale{}, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO app.audit_events(actor_user_id,organization_id,action,resource_type,resource_id,outcome,severity,metadata) VALUES($1::uuid,$2::uuid,'consumer.action','consumer_sale',$3,'success','high',jsonb_build_object('action',$4::text,'event_id',$5::text))`, actor, v.OrganizationID, id, in.Action, eventID); e != nil {
		return Sale{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Sale{}, e
	}
	v.Events[len(v.Events)-1] = ev
	v.Events[len(v.Events)-1].ID = eventID
	v.Version++
	v.Role = role
	v.Calculate()
	return v, nil
}
