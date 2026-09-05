package web

import (
	"context"
	"kredit/internal/collections"
	"kredit/internal/credit"
	"kredit/internal/disputes"
	"kredit/internal/paymentclaims"
	"kredit/internal/payments"
	"kredit/internal/tradelines"
	"net/http"
)

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
func (r *Runtime) readDisputesForOrganization(id string) ([]disputes.Dispute, error) {
	if source, ok := r.Disputes.(interface {
		ReadForOrganization(id string) ([]disputes.Dispute, error)
	}); ok {
		return source.ReadForOrganization(id)
	}
	return r.Disputes.ListForOrganization(id), nil
}
func (r *Runtime) readDisputesForBuyer(id string) ([]disputes.Dispute, error) {
	if source, ok := r.Disputes.(interface {
		ReadForBuyer(id string) ([]disputes.Dispute, error)
	}); ok {
		return source.ReadForBuyer(id)
	}
	return r.Disputes.ListForBuyer(id), nil
}
func (r *Runtime) readDisputesForObligation(id string) ([]disputes.Dispute, error) {
	if source, ok := r.Disputes.(interface {
		ReadForObligation(id string) ([]disputes.Dispute, error)
	}); ok {
		return source.ReadForObligation(id)
	}
	return r.Disputes.ListForObligation(id), nil
}
func (r *Runtime) readTradeLinesForSupplier(id string) ([]tradelines.TradeLine, error) {
	if source, ok := r.TradeLines.(interface {
		ReadForSupplier(id string) ([]tradelines.TradeLine, error)
	}); ok {
		return source.ReadForSupplier(id)
	}
	return r.TradeLines.ListForSupplier(id), nil
}
func (r *Runtime) readTradeLinesForBuyer(id string) ([]tradelines.TradeLine, error) {
	if source, ok := r.TradeLines.(interface {
		ReadForBuyer(id string) ([]tradelines.TradeLine, error)
	}); ok {
		return source.ReadForBuyer(id)
	}
	return r.TradeLines.ListForBuyer(id), nil
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
func (r *Runtime) readCollectionsAttempts(id string) ([]collections.Attempt, error) {
	return r.readCollectionsAttemptsContext(context.Background(), id)
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
