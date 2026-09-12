package mandates

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"kredit/internal/access"
)

// Router creates only on the active account. Every saved mandate is resolved
// locally to one original account before any provider request is made.
type Router struct {
	Provider
	pool     *pgxpool.Pool
	accounts map[string]*PostgresProvider
}

func NewRouter(pool *pgxpool.Pool, active Provider, retained ...*PostgresProvider) (*Router, error) {
	if pool == nil || active == nil {
		return nil, errors.New("mandate routing requires durable storage and an active provider")
	}
	r := &Router{Provider: active, pool: pool, accounts: map[string]*PostgresProvider{}}
	if p, ok := active.(*PostgresProvider); ok {
		r.accounts[p.Name()] = p
	}
	for _, p := range retained {
		if p == nil {
			return nil, errors.New("invalid saved mandate provider")
		}
		if _, exists := r.accounts[p.Name()]; exists {
			return nil, errors.New("duplicate mandate account name")
		}
		r.accounts[p.Name()] = p
	}
	return r, nil
}

type providerKey struct{}

func WithProvider(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, providerKey{}, name)
}
func (r *Router) selectAccount(ctx context.Context, id string) (*PostgresProvider, error) {
	wanted, _ := ctx.Value(providerKey{}).(string)
	var found *PostgresProvider
	for name, p := range r.accounts {
		if wanted != "" && name != wanted {
			continue
		}
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app.payment_mandate_by_provider($1,$2))`, name, id).Scan(&exists); err != nil {
			return nil, err
		}
		if exists {
			if found != nil {
				return nil, errors.New("mandate reference is ambiguous; select its original provider account")
			}
			found = p
		}
	}
	if found == nil {
		return nil, errors.New("the original mandate account is unavailable; restore its saved connection")
	}
	return found, nil
}
func (r *Router) GetMandate(ctx context.Context, id string) (Mandate, error) {
	p, e := r.selectAccount(ctx, id)
	if e != nil {
		return Mandate{}, e
	}
	return p.GetMandate(ctx, id)
}
func (r *Router) CancelMandate(ctx context.Context, id, reason string) (Mandate, error) {
	p, e := r.selectAccount(ctx, id)
	if e != nil {
		return Mandate{}, e
	}
	return p.CancelMandate(ctx, id, reason)
}
func (r *Router) RestoreAuthorization(ctx context.Context, id string) (Mandate, error) {
	p, e := r.selectAccount(ctx, id)
	if e != nil {
		return Mandate{}, e
	}
	return p.RestoreAuthorization(ctx, id)
}
func (r *Router) BlockMandate(ctx context.Context, id string, status Status, event string) (Mandate, error) {
	p, e := r.selectAccount(ctx, id)
	if e != nil {
		return Mandate{}, e
	}
	return p.BlockMandate(ctx, id, status, event)
}
func (r *Router) ReadForBuyer(ctx context.Context, user string) ([]Mandate, error) {
	return NewPostgresProvider(r.pool, r.Name()).ReadForBuyer(ctx, user)
}
func (r *Router) ResolveTradeLineMandate(ctx context.Context, id, user, business, org string) (Mandate, error) {
	return NewPostgresProvider(r.pool, r.Name()).ResolveTradeLineMandate(ctx, id, user, business, org)
}
func (r *Router) ListAuthorizationAttempts(ctx context.Context, actor string) ([]AuthorizationAttempt, error) {
	return NewPostgresProvider(r.pool, r.Name()).ListAuthorizationAttempts(ctx, actor)
}
func (r *Router) ResolveAuthorizationAttempt(ctx context.Context, actor, id, action, reference, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = access.LockPlatformAuthority(ctx, tx, actor, access.PermissionProviderOperations); err != nil {
		return err
	}
	var name string
	if err = tx.QueryRow(ctx, `SELECT provider FROM app.mandate_authorization_intents WHERE id=$1::uuid`, id).Scan(&name); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	p := r.accounts[name]
	if p == nil {
		return errors.New("restore the original saved mandate account before recovery")
	}
	return p.ResolveAuthorizationAttempt(ctx, actor, id, action, reference, reason)
}
