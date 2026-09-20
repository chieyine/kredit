package web

import (
	"context"
	"kredit/internal/collections"
	"kredit/internal/corrections"
	"kredit/internal/credit"
	"kredit/internal/disputes"
	"kredit/internal/organizations"
	"kredit/internal/paymentclaims"
	"kredit/internal/payments"
	"kredit/internal/tradelines"
	"net/http"
	"time"
)

// readMembership distinguishes "not a member" from "could not check". The
// authorization gate needs that difference: a failed lookup must not be
// reported to a business owner as a refusal, and must not be written to the
// audit trail as a denial nobody decided.
func (r *Runtime) readMembership(ctx context.Context, organizationID, userID string) (organizations.Membership, error) {
	if source, ok := r.Organizations.(interface {
		ReadMembership(context.Context, string, string) (organizations.Membership, error)
	}); ok {
		return source.ReadMembership(ctx, organizationID, userID)
	}
	membership, found := r.Organizations.Membership(organizationID, userID)
	if !found {
		return organizations.Membership{}, organizations.ErrMembershipNotFound
	}
	return membership, nil
}

func (r *Runtime) getCreditForSupplier(ctx context.Context, requestID, organizationID string) (credit.View, error) {
	if source, ok := r.Credit.(interface {
		GetForSupplierContext(context.Context, string, string) (credit.View, error)
	}); ok {
		return source.GetForSupplierContext(ctx, requestID, organizationID)
	}
	return r.Credit.GetForSupplier(requestID, organizationID)
}

func (r *Runtime) readCreditForSupplier(ctx context.Context, id string) ([]credit.View, error) {
	if source, ok := r.Credit.(interface {
		ReadForSupplier(ctx context.Context, id string) ([]credit.View, error)
	}); ok {
		return source.ReadForSupplier(ctx, id)
	}
	return r.Credit.ListForSupplier(id), nil
}
func (r *Runtime) readCreditForBuyer(ctx context.Context, id string) ([]credit.View, error) {
	if source, ok := r.Credit.(interface {
		ReadForBuyer(ctx context.Context, id string) ([]credit.View, error)
	}); ok {
		return source.ReadForBuyer(ctx, id)
	}
	return r.Credit.ListForBuyer(id), nil
}
func (r *Runtime) readPayments(ctx context.Context, id string) ([]payments.Payment, error) {
	if source, ok := r.Payments.(interface {
		ReadContext(context.Context, string) ([]payments.Payment, error)
	}); ok {
		return source.ReadContext(ctx, id)
	}
	if source, ok := r.Payments.(interface {
		Read(id string) ([]payments.Payment, error)
	}); ok {
		return source.Read(id)
	}
	return r.Payments.List(id)
}

func (r *Runtime) getPayment(ctx context.Context, id string) (payments.Payment, error) {
	if source, ok := r.Payments.(interface {
		GetContext(context.Context, string) (payments.Payment, error)
	}); ok {
		return source.GetContext(ctx, id)
	}
	return r.Payments.Get(id)
}
func (r *Runtime) readDisputesForOrganization(ctx context.Context, id string) ([]disputes.Dispute, error) {
	if source, ok := r.ScopedDisputes(ctx).(interface {
		ReadForOrganization(id string) ([]disputes.Dispute, error)
	}); ok {
		return source.ReadForOrganization(id)
	}
	return r.ScopedDisputes(ctx).ListForOrganization(id), nil
}
func (r *Runtime) readDisputesForBuyer(ctx context.Context, id string) ([]disputes.Dispute, error) {
	if source, ok := r.ScopedDisputes(ctx).(interface {
		ReadForBuyer(id string) ([]disputes.Dispute, error)
	}); ok {
		return source.ReadForBuyer(id)
	}
	return r.ScopedDisputes(ctx).ListForBuyer(id), nil
}
func (r *Runtime) readDisputesForObligation(ctx context.Context, id string) ([]disputes.Dispute, error) {
	if source, ok := r.ScopedDisputes(ctx).(interface {
		ReadForObligation(id string) ([]disputes.Dispute, error)
	}); ok {
		return source.ReadForObligation(id)
	}
	return r.ScopedDisputes(ctx).ListForObligation(id), nil
}
func (r *Runtime) readTradeLinesForSupplier(ctx context.Context, id string) ([]tradelines.TradeLine, error) {
	if source, ok := r.ScopedTradeLines(ctx).(interface {
		ReadForSupplier(id string) ([]tradelines.TradeLine, error)
	}); ok {
		return source.ReadForSupplier(id)
	}
	return r.ScopedTradeLines(ctx).ListForSupplier(id), nil
}
func (r *Runtime) readTradeLinesForBuyer(ctx context.Context, id string) ([]tradelines.TradeLine, error) {
	if source, ok := r.ScopedTradeLines(ctx).(interface {
		ReadForBuyer(id string) ([]tradelines.TradeLine, error)
	}); ok {
		return source.ReadForBuyer(id)
	}
	return r.ScopedTradeLines(ctx).ListForBuyer(id), nil
}
func (r *Runtime) readTradeLinesForBuyerOrganization(ctx context.Context, org string) ([]tradelines.TradeLine, error) {
	if source, ok := r.ScopedTradeLines(ctx).(interface {
		ReadForBuyerOrganization(org string) ([]tradelines.TradeLine, error)
	}); ok {
		return source.ReadForBuyerOrganization(org)
	}
	return r.ScopedTradeLines(ctx).ListForBuyerOrganization(org), nil
}
func (r *Runtime) readPaymentClaimsForBuyer(ctx context.Context, id string) ([]paymentclaims.Claim, error) {
	if source, ok := r.PaymentClaims.(interface {
		ReadForBuyer(ctx context.Context, id string) ([]paymentclaims.Claim, error)
	}); ok {
		return source.ReadForBuyer(ctx, id)
	}
	return r.PaymentClaims.ListForBuyer(ctx, id), nil
}
func (r *Runtime) readPaymentClaimsForSupplier(ctx context.Context, id string) ([]paymentclaims.Claim, error) {
	if source, ok := r.PaymentClaims.(interface {
		ReadForSupplier(ctx context.Context, id string) ([]paymentclaims.Claim, error)
	}); ok {
		return source.ReadForSupplier(ctx, id)
	}
	return r.PaymentClaims.ListForSupplier(ctx, id), nil
}
func (r *Runtime) readPaymentClaimsForObligation(ctx context.Context, id string) ([]paymentclaims.Claim, error) {
	if source, ok := r.PaymentClaims.(interface {
		ReadForObligation(ctx context.Context, id string) ([]paymentclaims.Claim, error)
	}); ok {
		return source.ReadForObligation(ctx, id)
	}
	return r.PaymentClaims.ListForObligation(ctx, id), nil
}
func (r *Runtime) collectionEligibilityContext(ctx context.Context, id string, now time.Time) (collections.Eligibility, error) {
	if source, ok := r.Collections.(interface {
		EligibilityContext(context.Context, string, time.Time) (collections.Eligibility, error)
	}); ok {
		return source.EligibilityContext(ctx, id, now)
	}
	return r.Collections.Eligibility(id, now)
}
func (r *Runtime) getCollectionAttemptContext(ctx context.Context, id string) (collections.Attempt, bool) {
	if source, ok := r.Collections.(interface {
		GetAttemptContext(context.Context, string) (collections.Attempt, bool)
	}); ok {
		return source.GetAttemptContext(ctx, id)
	}
	return r.Collections.GetAttempt(id)
}
func (r *Runtime) readCollectionsAttemptsContext(ctx context.Context, id string) ([]collections.Attempt, error) {
	if source, ok := r.Collections.(interface {
		ReadAttemptsContext(context.Context, string) ([]collections.Attempt, error)
	}); ok {
		return source.ReadAttemptsContext(ctx, id)
	}
	if source, ok := r.Collections.(interface {
		ReadAttempts(id string) ([]collections.Attempt, error)
	}); ok {
		return source.ReadAttempts(id)
	}
	return r.Collections.ListAttempts(id), nil
}
func financialReadError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	writeProblem(w, 503, "financial_data_unavailable", "Financial data could not be loaded; please retry")
	return true
}

func (r *Runtime) ScopedTradeLines(ctx context.Context) tradelines.Service {
	if scoped, ok := r.TradeLines.(interface {
		ForContext(context.Context) tradelines.Service
	}); ok {
		return scoped.ForContext(ctx)
	}
	return r.TradeLines
}

func (r *Runtime) ScopedCorrections(ctx context.Context) corrections.Service {
	if scoped, ok := r.Corrections.(interface {
		ForContext(context.Context) corrections.Service
	}); ok {
		return scoped.ForContext(ctx)
	}
	return r.Corrections
}

func (r *Runtime) ScopedDisputes(ctx context.Context) disputes.Service {
	if scoped, ok := r.Disputes.(interface {
		ForContext(context.Context) disputes.Service
	}); ok {
		return scoped.ForContext(ctx)
	}
	return r.Disputes
}
