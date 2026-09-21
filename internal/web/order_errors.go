package web

import (
	"errors"
	"kredit/internal/credit"
	"kredit/internal/orders"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Server) writeOrderProblem(w http.ResponseWriter, r *http.Request, err error) {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, orders.ErrNotFound), errors.Is(err, pgx.ErrNoRows), errors.Is(err, credit.ErrRequestNotFound):
		writeProblem(w, 404, "order_record_not_found", "Order record was not found.")
	case errors.Is(err, orders.ErrAuthority), errors.Is(err, orders.ErrDualControl):
		writeProblem(w, 403, "order_forbidden", "Current authority and independent approval are required.")
	case errors.Is(err, orders.ErrConflict), errors.Is(err, orders.ErrAlreadyDecided):
		writeProblem(w, 409, "order_conflict", "This action conflicts with recorded evidence. Refresh the record before retrying.")
	case errors.Is(err, orders.ErrInvalidInput):
		writeProblem(w, 422, "invalid_order_input", "Check the order, quantities, amount and supporting evidence.")
	case errors.As(err, &pgErr) && pgErr.Code == "42501":
		writeProblem(w, 403, "order_forbidden", "You are not authorized to change this order evidence.")
	case pgErr != nil && (pgErr.Code == "23503" || pgErr.Code == "23514" || pgErr.Code == "22P02"):
		writeProblem(w, 422, "invalid_order_input", "The submitted records or quantities do not match this order.")
	case pgErr != nil && pgErr.Code == "23505":
		writeProblem(w, 409, "order_conflict", "This evidence has already been recorded.")
	default:
		// Avoid returning SQL diagnostics, identifiers, or submitted personal
		// information. The request ID is sufficient to correlate an outage.
		s.logger.ErrorContext(r.Context(), "order operation unavailable", "request_id", requestIDFromContext(r.Context()))
		writeProblem(w, 503, "order_unavailable", "Order records are temporarily unavailable. Please retry.")
	}
}
