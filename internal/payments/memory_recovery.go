package payments

import (
	"errors"
	"fmt"

	"kredit/internal/ledger"
)

// ErrReconciliationRequired is not a retryable payment error. A callback may
// change state and then fail; a compensating callback can do the same. Without
// a common transaction there is no sound way to infer that either had no effect.
// Keep the evidence and refuse further mutation instead of issuing a guessed
// inverse or generating fresh journal keys. Real financial state belongs in
// PostgresStore, not in this process-local development adapter.
var ErrReconciliationRequired = errors.New("in-memory payment operation requires reconciliation; automatic replay is blocked")

type memoryOperation struct {
	payment   Payment
	kind      string
	key       string
	stage     string
	completed bool
	cause     error
}

func (op *memoryOperation) recoveryError() error {
	return errors.Join(ErrReconciliationRequired,
		fmt.Errorf("%s payment %s stopped at %s", op.kind, op.payment.ID, op.stage), op.cause)
}

// All operation state is protected by Store.mu. The block is installed BEFORE
// the first side effect. It survives returned errors and panic unwinding; there
// is deliberately no generic "clear and retry" API that could erase uncertainty.
// This defer does not recover a panic or replace its original value.
func (s *Store) finishMemoryOperation(op *memoryOperation, result *error) {
	if op.completed {
		delete(s.blocked, op.payment.ObligationID)
		if op.key != "" {
			delete(s.recording, op.key)
		}
		return
	}
	op.cause = *result
	if op.cause == nil {
		op.cause = errors.New("operation interrupted before publication")
	}
	*result = op.recoveryError()
}

func (s *Store) memoryBlock(obligationID string) error {
	if op := s.blocked[obligationID]; op != nil {
		return op.recoveryError()
	}
	return nil
}

// Nonempty allocator output must represent this entire payment exactly once.
// Subtracting from the expected amount avoids an overflowing accumulated sum.
func validateMemoryAllocations(targets []AllocationTarget, amount ledger.Money) error {
	if len(targets) == 0 {
		return nil
	}
	remaining := amount
	seen := make(map[string]bool, len(targets))
	for _, target := range targets {
		if target.ScheduleItemID == "" || seen[target.ScheduleItemID] || target.AmountKobo <= 0 || target.AmountKobo > remaining {
			return errors.New("allocation callback returned inconsistent payment targets")
		}
		seen[target.ScheduleItemID] = true
		remaining -= target.AmountKobo
	}
	if remaining != 0 {
		return errors.New("allocation callback did not allocate the full payment")
	}
	return nil
}
