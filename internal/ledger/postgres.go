package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"kredit/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the durable journal implementation. A single database
// transaction writes the journal header and every posting, and the unique
// idempotency key makes retries return the original transaction.
type PostgresStore struct {
	pool   *pgxpool.Pool
	outbox *outbox.Store
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func NewPostgresStoreWithOutbox(pool *pgxpool.Pool, events *outbox.Store) *PostgresStore {
	return &PostgresStore{pool: pool, outbox: events}
}

func (s *PostgresStore) PostPayment(paymentID string, amount Money, source string, effectiveAt time.Time, key string) (Transaction, error) {
	settlement, err := settlementAccount(source)
	if err != nil {
		return Transaction{}, err
	}
	return s.post(Transaction{EventType: "payment_recognized", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: settlement, Debit: amount}, {Account: AccountTradeReceivable, Credit: amount}}})
}

func (s *PostgresStore) PostPaymentReversal(paymentID string, amount Money, source string, effectiveAt time.Time, key string) (Transaction, error) {
	settlement, err := settlementAccount(source)
	if err != nil {
		return Transaction{}, err
	}
	return s.post(Transaction{EventType: "payment_reversed", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountTradeReceivable, Debit: amount}, {Account: settlement, Credit: amount}}})
}

func (s *PostgresStore) PostCollectionFee(paymentID string, amount Money, effectiveAt time.Time, key string) (Transaction, error) {
	return s.post(Transaction{EventType: "collection_fee_accrued", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountSupplierFeeReceivable, Debit: amount}, {Account: AccountPlatformCollectionRevenue, Credit: amount}}})
}

func (s *PostgresStore) PostCollectionFeeReversal(paymentID string, amount Money, effectiveAt time.Time, key string) (Transaction, error) {
	return s.post(Transaction{EventType: "collection_fee_reversed", ReferenceType: "payment", ReferenceID: paymentID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountPlatformCollectionRevenue, Debit: amount}, {Account: AccountSupplierFeeReceivable, Credit: amount}}})
}

func (s *PostgresStore) PostAdjustment(referenceID string, amount Money, adjustmentType string, effectiveAt time.Time, key string) (Transaction, error) {
	account := AccountReturnsAdjustment
	if adjustmentType == "write_off" {
		account = AccountWriteOff
	} else if adjustmentType != "dispute_adjustment" {
		return Transaction{}, errors.New("invalid adjustment type")
	}
	return s.post(Transaction{EventType: adjustmentType, ReferenceType: "dispute", ReferenceID: referenceID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: account, Debit: amount}, {Account: AccountTradeReceivable, Credit: amount}}})
}

func (s *PostgresStore) PostFeeWaiver(referenceID string, amount Money, effectiveAt time.Time, key string) (Transaction, error) {
	return s.post(Transaction{EventType: "fee_waived", ReferenceType: "obligation", ReferenceID: referenceID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountPlatformServiceRevenue, Debit: amount}, {Account: AccountSupplierFeeReceivable, Credit: amount}}})
}

func (s *PostgresStore) PostActivation(obligationID string, principal Money, effectiveAt time.Time, key string) (Transaction, error) {
	baseFee, err := BaseFee(principal)
	if err != nil {
		return Transaction{}, err
	}
	return s.post(Transaction{EventType: "principal_activated", ReferenceType: "obligation", ReferenceID: obligationID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountTradeReceivable, Debit: principal}, {Account: AccountPrincipalOriginated, Credit: principal}, {Account: AccountSupplierFeeReceivable, Debit: baseFee}, {Account: AccountPlatformServiceRevenue, Credit: baseFee}}})
}

// PostActivationTx writes activation postings and their outbox event inside a
// caller-owned transaction so a larger financial aggregate can commit once.
func (s *PostgresStore) PostActivationTx(ctx context.Context, tx pgx.Tx, obligationID string, principal Money, effectiveAt time.Time, key string) (Transaction, error) {
	baseFee, err := BaseFee(principal)
	if err != nil {
		return Transaction{}, err
	}
	return s.postTx(ctx, tx, Transaction{EventType: "principal_activated", ReferenceType: "obligation", ReferenceID: obligationID, IdempotencyKey: key, EffectiveAt: effectiveAt, Postings: []Posting{{Account: AccountTradeReceivable, Debit: principal}, {Account: AccountPrincipalOriginated, Credit: principal}, {Account: AccountSupplierFeeReceivable, Debit: baseFee}, {Account: AccountPlatformServiceRevenue, Credit: baseFee}}})
}

func settlementAccount(source string) (string, error) {
	switch source {
	case "integrated_voluntary", "supplier_recorded_transfer", "buyer_payment_claim", "cash_recorded", "adjustment":
		return AccountVoluntarySettlement, nil
	case "kredit_collection":
		return AccountCollectionSettlement, nil
	default:
		return "", errors.New("invalid payment source")
	}
}

func (s *PostgresStore) post(transaction Transaction) (Transaction, error) {
	if s == nil || s.pool == nil {
		return Transaction{}, errors.New("ledger database is not configured")
	}
	if err := validateTransaction(transaction); err != nil {
		return Transaction{}, err
	}
	ctx := context.Background()
	if transaction.EffectiveAt.IsZero() {
		transaction.EffectiveAt = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := s.postTx(ctx, tx, transaction)
	if err != nil {
		return Transaction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, err
	}
	return result, nil
}

func (s *PostgresStore) postTx(ctx context.Context, tx pgx.Tx, transaction Transaction) (Transaction, error) {
	if tx == nil {
		return Transaction{}, errors.New("ledger transaction is required")
	}
	if err := validateTransaction(transaction); err != nil {
		return Transaction{}, err
	}
	if transaction.EffectiveAt.IsZero() {
		transaction.EffectiveAt = time.Now().UTC()
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO ledger.transactions (event_type, reference_type, reference_id, idempotency_key, effective_at)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id::text`, transaction.EventType, transaction.ReferenceType, transaction.ReferenceID, transaction.IdempotencyKey, transaction.EffectiveAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(ctx, `SELECT id::text FROM ledger.transactions WHERE idempotency_key = $1`, transaction.IdempotencyKey).Scan(&id); err != nil {
			return Transaction{}, err
		}
		result, err := loadTransaction(ctx, tx, id)
		if err != nil {
			return Transaction{}, err
		}
		if !sameTransactionIntent(result, transaction) {
			return Transaction{}, errors.New("idempotency key was reused for a different ledger transaction")
		}
		if err := s.appendOutbox(ctx, tx, result); err != nil {
			return Transaction{}, err
		}
		return result, nil
	}
	if err != nil {
		return Transaction{}, err
	}
	for _, posting := range transaction.Postings {
		var inserted int64
		err = tx.QueryRow(ctx, `
			INSERT INTO ledger.postings (transaction_id, account_id, debit_kobo, credit_kobo)
			SELECT $1::uuid, id, $3, $4 FROM ledger.accounts WHERE code = $2
			RETURNING 1`, id, posting.Account, int64(posting.Debit), int64(posting.Credit)).Scan(&inserted)
		if err != nil {
			return Transaction{}, fmt.Errorf("insert ledger posting %s: %w", posting.Account, err)
		}
	}
	result, err := loadTransaction(ctx, tx, id)
	if err != nil {
		return Transaction{}, err
	}
	if err := s.appendOutbox(ctx, tx, result); err != nil {
		return Transaction{}, err
	}
	return result, nil
}

func (s *PostgresStore) appendOutbox(ctx context.Context, tx pgx.Tx, transaction Transaction) error {
	if s.outbox == nil {
		return nil
	}
	payload, err := json.Marshal(transaction)
	if err != nil {
		return err
	}
	_, err = s.outbox.AppendTx(ctx, tx, outbox.Event{
		AggregateType:  transaction.ReferenceType,
		AggregateID:    transaction.ReferenceID,
		EventType:      "ledger." + transaction.EventType,
		Payload:        payload,
		IdempotencyKey: "ledger:" + transaction.IdempotencyKey,
	})
	return err
}

func loadTransaction(ctx context.Context, tx pgx.Tx, id string) (Transaction, error) {
	var result Transaction
	if err := tx.QueryRow(ctx, `SELECT id::text, event_type, reference_type, reference_id, idempotency_key, effective_at, recorded_at FROM ledger.transactions WHERE id = $1::uuid`, id).Scan(&result.ID, &result.EventType, &result.ReferenceType, &result.ReferenceID, &result.IdempotencyKey, &result.EffectiveAt, &result.RecordedAt); err != nil {
		return Transaction{}, err
	}
	rows, err := tx.Query(ctx, `SELECT accounts.code, postings.debit_kobo, postings.credit_kobo FROM ledger.postings postings JOIN ledger.accounts accounts ON accounts.id = postings.account_id WHERE postings.transaction_id = $1::uuid ORDER BY postings.id`, id)
	if err != nil {
		return Transaction{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var posting Posting
		if err := rows.Scan(&posting.Account, &posting.Debit, &posting.Credit); err != nil {
			return Transaction{}, err
		}
		result.Postings = append(result.Postings, posting)
	}
	if err := rows.Err(); err != nil {
		return Transaction{}, err
	}
	return result, nil
}

func (s *PostgresStore) GetByReference(referenceID string) ([]Transaction, error) {
	if s == nil || s.pool == nil || referenceID == "" {
		return []Transaction{}, errors.New("ledger database is not configured")
	}
	ctx := context.Background()
	txRows, err := s.pool.Query(ctx, `
		SELECT id::text, event_type, reference_type, reference_id, idempotency_key, effective_at, recorded_at
		FROM ledger.transactions
		WHERE reference_id = $1
		ORDER BY effective_at, recorded_at`, referenceID)
	if err != nil {
		return nil, err
	}
	defer txRows.Close()

	transactions := make([]Transaction, 0)
	txMap := make(map[string]*Transaction)
	ids := make([]string, 0)

	for txRows.Next() {
		var t Transaction
		if err := txRows.Scan(&t.ID, &t.EventType, &t.ReferenceType, &t.ReferenceID, &t.IdempotencyKey, &t.EffectiveAt, &t.RecordedAt); err != nil {
			return nil, err
		}
		t.Postings = make([]Posting, 0)
		transactions = append(transactions, t)
		ids = append(ids, t.ID)
	}
	if err := txRows.Err(); err != nil {
		return nil, err
	}
	txRows.Close()

	if len(ids) == 0 {
		return transactions, nil
	}

	for i := range transactions {
		txMap[transactions[i].ID] = &transactions[i]
	}

	pRows, err := s.pool.Query(ctx, `
		SELECT postings.transaction_id::text, accounts.code, postings.debit_kobo, postings.credit_kobo
		FROM ledger.postings postings
		JOIN ledger.accounts accounts ON accounts.id = postings.account_id
		WHERE postings.transaction_id = ANY($1::uuid[])
		ORDER BY postings.id`, ids)
	if err != nil {
		return nil, err
	}
	defer pRows.Close()

	for pRows.Next() {
		var txID string
		var p Posting
		if err := pRows.Scan(&txID, &p.Account, &p.Debit, &p.Credit); err != nil {
			return nil, err
		}
		if t, ok := txMap[txID]; ok {
			t.Postings = append(t.Postings, p)
		}
	}
	if err := pRows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (s *PostgresStore) PostActivationWithFeeTx(ctx context.Context, tx pgx.Tx, id string, principal, fee Money, at time.Time, key string) (Transaction, error) {
	t, err := activationTransaction(id, principal, fee, at, key)
	if err != nil {
		return t, err
	}
	return s.postTx(ctx, tx, t)
}
