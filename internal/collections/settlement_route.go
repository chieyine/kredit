package collections

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"kredit/internal/billing"
	"kredit/internal/ledger"
)

// SettlementRoute is a snapshot, not a claim that money reached the seller.
// It survives later bank changes and appears in the reconciliation worklist.
type SettlementRoute struct {
	BaseFees         []billing.SplitFee `json:"base_fees,omitempty"`
	Method           string             `json:"method"`
	NetAmountKobo    ledger.Money       `json:"net_amount_kobo"`
	FeeAmountKobo    ledger.Money       `json:"fee_amount_kobo"`
	Provider         string             `json:"provider"`
	Connection       string             `json:"connection"`
	Destination      string             `json:"destination"`
	AccountName      string             `json:"account_name"`
	AccountLast4     string             `json:"account_last4"`
	BankName         string             `json:"bank_name"`
	ProfileVersion   int                `json:"profile_version"`
	BillingMethod    string             `json:"billing_method"`
	BillingReference string             `json:"billing_reference"`
}

func (e *PostgresEngine) RequireSettlementRoute() { e.requireSettlementRoute = true }
func freezeSettlementRoute(ctx context.Context, tx pgx.Tx, obligation string) (*SettlementRoute, error) {
	route := &SettlementRoute{}
	err := tx.QueryRow(ctx, `SELECT p.settlement_provider,r.connection_identity,p.settlement_provider_reference,p.settlement_account_name,p.settlement_account_last4,p.settlement_bank_name,p.version,p.billing_method,p.billing_provider_reference FROM app.supplier_onboarding_profiles p JOIN app.obligations o ON o.supplier_organization_id=p.organization_id JOIN app.settlement_registrations r ON r.organization_id=p.organization_id AND r.provider=p.settlement_provider AND r.result->>'provider_reference'=p.settlement_provider_reference AND r.state='REGISTERED' WHERE o.id=$1::uuid AND p.settlement_state='verified' AND p.billing_state='configured' ORDER BY r.created_at DESC LIMIT 1 FOR SHARE OF p`, obligation).Scan(&route.Provider, &route.Connection, &route.Destination, &route.AccountName, &route.AccountLast4, &route.BankName, &route.ProfileVersion, &route.BillingMethod, &route.BillingReference)
	if err != nil {
		return nil, errors.New("a verified seller bank and billing arrangement are required before collection")
	}
	return route, nil
}

type settlementRouteLoader func(context.Context, string, ledger.Money) (*SettlementRoute, error)
