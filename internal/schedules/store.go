package schedules

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"kredit/internal/ledger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TypeEqual          = "equal"
	TypeCustom         = "custom"
	CadenceWeekly      = "weekly"
	CadenceFortnightly = "fortnightly"
	CadenceMonthly     = "monthly"
	CadenceCustom      = "custom"
	PolicyLastDay      = "last_day"
	PolicyCap          = "cap"
	ItemOpen           = "OPEN"
	ItemInGrace        = "IN_GRACE"
	ItemOverdue        = "OVERDUE"
	ItemPartiallyPaid  = "PARTIALLY_PAID"
	ItemPaid           = "PAID"
	ItemCancelled      = "CANCELLED"
)

type CustomItem struct {
	AmountKobo ledger.Money
	DueDate    time.Time
}
type CreateInput struct {
	ObligationID         string
	PrincipalKobo        ledger.Money
	ScheduleType         string
	Count                int
	InstalmentAmountKobo ledger.Money
	StartDate            time.Time
	DueHour              int
	DueMinute            int
	Timezone             string
	GraceHours           int
	Cadence              string
	MonthEndPolicy       string
	CustomItems          []CustomItem
	AllocationPolicy     string
}
type Schedule struct {
	ID               string    `json:"id"`
	ObligationID     string    `json:"obligation_id"`
	ScheduleType     string    `json:"schedule_type"`
	Timezone         string    `json:"timezone"`
	AllocationPolicy string    `json:"allocation_policy"`
	Cadence          string    `json:"cadence"`
	GraceHours       int       `json:"grace_hours"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}
type Item struct {
	ID                    string       `json:"id"`
	ScheduleID            string       `json:"schedule_id"`
	Sequence              int          `json:"sequence"`
	PrincipalDueKobo      ledger.Money `json:"principal_due_kobo"`
	DueAt                 time.Time    `json:"due_at"`
	GraceHours            int          `json:"grace_hours"`
	CollectionAt          time.Time    `json:"collection_at"`
	AllocatedKobo         ledger.Money `json:"allocated_kobo"`
	CollectedKobo         ledger.Money `json:"collected_kobo"`
	State                 string       `json:"state"`
	DisputedKobo          ledger.Money `json:"disputed_kobo"`
	CollectionBlockReason string       `json:"collection_block_reason,omitempty"`
}
type AllocationTarget struct {
	ScheduleItemID string
	AmountKobo     ledger.Money
}

type Store struct {
	mu           sync.RWMutex
	schedules    map[string]*Schedule
	items        map[string][]*Item
	byObligation map[string]string
	now          func() time.Time
	newID        func() string
	pool         *pgxpool.Pool
}

func NewStore() *Store {
	return &Store{schedules: map[string]*Schedule{}, items: map[string][]*Item{}, byObligation: map[string]string{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier}
}

func NewPostgresStore(pool *pgxpool.Pool) *Store {
	return &Store{schedules: map[string]*Schedule{}, items: map[string][]*Item{}, byObligation: map[string]string{}, now: func() time.Time { return time.Now().UTC() }, newID: newIdentifier, pool: pool}
}

func (s *Store) Create(input CreateInput) (Schedule, []Item, error) {
	if s.pool != nil {
		return s.createPostgres(input)
	}
	if input.ObligationID == "" || input.PrincipalKobo <= 0 {
		return Schedule{}, nil, errors.New("obligation and positive principal are required")
	}
	if input.Timezone == "" {
		input.Timezone = "Africa/Lagos"
	}
	loc, err := time.LoadLocation(input.Timezone)
	if err != nil {
		return Schedule{}, nil, errors.New("valid timezone is required")
	}
	if input.GraceHours < 0 || input.GraceHours > 720 {
		return Schedule{}, nil, errors.New("grace hours must be between 0 and 720")
	}
	if input.StartDate.IsZero() {
		return Schedule{}, nil, errors.New("start date is required")
	}
	if input.ScheduleType != TypeEqual && input.ScheduleType != TypeCustom {
		return Schedule{}, nil, errors.New("schedule type must be equal or custom")
	}
	if input.AllocationPolicy == "" {
		input.AllocationPolicy = "due_date_order"
	}
	if input.Cadence == "" {
		input.Cadence = CadenceMonthly
	}
	if input.MonthEndPolicy == "" {
		input.MonthEndPolicy = PolicyCap
	}
	if input.Cadence != CadenceWeekly && input.Cadence != CadenceFortnightly && input.Cadence != CadenceMonthly && input.Cadence != CadenceCustom {
		return Schedule{}, nil, errors.New("invalid cadence")
	}
	amounts := []ledger.Money{}
	dates := []time.Time{}
	if input.ScheduleType == TypeEqual {
		if input.Count < 1 || input.Count > 60 {
			return Schedule{}, nil, errors.New("equal schedule count must be between 1 and 60")
		}
		if input.InstalmentAmountKobo > 0 {
			if input.InstalmentAmountKobo > input.PrincipalKobo || input.InstalmentAmountKobo != input.PrincipalKobo/ledger.Money(input.Count) || input.PrincipalKobo%ledger.Money(input.Count) != 0 {
				return Schedule{}, nil, errors.New("instalment amount and count must equal principal")
			}
		}
		base := input.PrincipalKobo / ledger.Money(input.Count)
		remainder := input.PrincipalKobo - (base * ledger.Money(input.Count))
		for i := 0; i < input.Count; i++ {
			amount := base
			if i == input.Count-1 {
				amount += remainder
			}
			amounts = append(amounts, amount)
			date, err := cadenceDate(input.StartDate, i, input.Cadence, input.MonthEndPolicy, loc)
			if err != nil {
				return Schedule{}, nil, err
			}
			dates = append(dates, date)
		}
	} else {
		if len(input.CustomItems) == 0 || len(input.CustomItems) > 60 {
			return Schedule{}, nil, errors.New("custom schedule must contain 1 to 60 items")
		}
		var total ledger.Money
		for _, item := range input.CustomItems {
			if item.AmountKobo <= 0 || item.DueDate.IsZero() {
				return Schedule{}, nil, errors.New("custom amounts and dates must be positive")
			}
			amounts = append(amounts, item.AmountKobo)
			dates = append(dates, item.DueDate.In(loc))
			var addErr error
			total, addErr = ledger.CheckedAdd(total, item.AmountKobo)
			if addErr != nil {
				return Schedule{}, nil, errors.New("schedule total is too large")
			}
		}
		if total != input.PrincipalKobo {
			return Schedule{}, nil, errors.New("schedule amounts must equal principal")
		}
	}
	for i := 1; i < len(dates); i++ {
		if !dates[i].After(dates[i-1]) {
			return Schedule{}, nil, errors.New("schedule dates must be ordered")
		}
	}
	schedule := &Schedule{ID: "schedule-" + s.newID(), ObligationID: input.ObligationID, ScheduleType: input.ScheduleType, Timezone: input.Timezone, AllocationPolicy: input.AllocationPolicy, Cadence: input.Cadence, GraceHours: input.GraceHours, Status: "ACTIVE", CreatedAt: s.now()}
	items := make([]*Item, 0, len(amounts))
	for i, amount := range amounts {
		due := time.Date(dates[i].Year(), dates[i].Month(), dates[i].Day(), input.DueHour, input.DueMinute, 0, 0, loc)
		items = append(items, &Item{ID: "schedule-item-" + s.newID(), ScheduleID: schedule.ID, Sequence: i + 1, PrincipalDueKobo: amount, DueAt: due, GraceHours: input.GraceHours, CollectionAt: due.Add(time.Duration(input.GraceHours) * time.Hour), State: ItemOpen})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byObligation[input.ObligationID]; exists {
		return Schedule{}, nil, errors.New("obligation already has a schedule")
	}
	s.schedules[schedule.ID] = schedule
	s.items[schedule.ID] = items
	s.byObligation[input.ObligationID] = schedule.ID
	return cloneSchedule(*schedule), cloneItems(items), nil
}

func (s *Store) CreateDefault(obligationID string, principal ledger.Money, dueDate string, collectionAt time.Time, graceHours int) (Schedule, []Item, error) {
	date, err := time.ParseInLocation("2006-01-02", dueDate, time.UTC)
	if err != nil {
		return Schedule{}, nil, err
	}
	return s.Create(CreateInput{ObligationID: obligationID, PrincipalKobo: principal, ScheduleType: TypeEqual, Count: 1, StartDate: date, DueHour: collectionAt.Hour(), DueMinute: collectionAt.Minute(), Timezone: collectionAt.Location().String(), GraceHours: graceHours, Cadence: CadenceCustom})
}

func (s *Store) GetForObligation(obligationID string) (Schedule, []Item, error) {
	if s.pool != nil {
		return s.getPostgres(obligationID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id := s.byObligation[obligationID]
	if id == "" {
		return Schedule{}, nil, errors.New("schedule not found")
	}
	return cloneSchedule(*s.schedules[id]), cloneItems(s.items[id]), nil
}

func (s *Store) DeleteIfEmpty(obligationID string) error {
	if s.pool != nil {
		ctx := context.Background()
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		var scheduleID string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM app.repayment_schedules WHERE obligation_id = $1::uuid FOR UPDATE`, obligationID).Scan(&scheduleID); errors.Is(err, pgx.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		var allocated int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(allocated_kobo),0) FROM app.schedule_items WHERE schedule_id = $1::uuid`, scheduleID).Scan(&allocated); err != nil {
			return err
		}
		if allocated > 0 {
			return errors.New("schedule has allocated payments")
		}
		if _, err := tx.Exec(ctx, `DELETE FROM app.repayment_schedules WHERE id = $1::uuid`, scheduleID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.byObligation[obligationID]
	if id == "" {
		return nil
	}
	for _, item := range s.items[id] {
		if item.AllocatedKobo > 0 {
			return errors.New("schedule has allocated payments")
		}
	}
	delete(s.items, id)
	delete(s.schedules, id)
	delete(s.byObligation, obligationID)
	return nil
}
func (s *Store) ListForObligation(obligationID string) []Item {
	_, items, err := s.GetForObligation(obligationID)
	if err != nil {
		return []Item{}
	}
	return items
}

func (s *Store) Allocate(obligationID string, amount ledger.Money) ([]AllocationTarget, error) {
	if s.pool != nil {
		return s.allocatePostgres(obligationID, amount)
	}
	if amount <= 0 {
		return nil, errors.New("allocation amount must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.byObligation[obligationID]
	if id == "" {
		return []AllocationTarget{{AmountKobo: amount}}, nil
	}
	items := s.items[id]
	capacity := ledger.Money(0)
	for _, item := range items {
		remainingForItem := item.PrincipalDueKobo - item.AllocatedKobo
		if remainingForItem > 0 {
			capacity += remainingForItem
		}
	}
	if amount > capacity {
		return nil, errors.New("allocation exceeds schedule outstanding")
	}
	remaining := amount
	targets := []AllocationTarget{}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Sequence < items[j].Sequence })
	for _, item := range items {
		open := item.PrincipalDueKobo - item.AllocatedKobo
		if open <= 0 {
			continue
		}
		take := open
		if take > remaining {
			take = remaining
		}
		item.AllocatedKobo += take
		if item.AllocatedKobo == item.PrincipalDueKobo {
			item.State = ItemPaid
		} else {
			item.State = ItemPartiallyPaid
		}
		targets = append(targets, AllocationTarget{ScheduleItemID: item.ID, AmountKobo: take})
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining > 0 {
		return nil, errors.New("allocation exceeds schedule outstanding")
	}
	return targets, nil
}
func (s *Store) ReverseAllocations(targets []AllocationTarget) error {
	if s.pool != nil {
		return s.reverseAllocationsPostgres(targets)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, target := range targets {
		for _, items := range s.items {
			for _, item := range items {
				if item.ID == target.ScheduleItemID {
					if target.AmountKobo > item.AllocatedKobo {
						return errors.New("schedule allocation reversal exceeds allocated amount")
					}
					item.AllocatedKobo -= target.AmountKobo
					if item.AllocatedKobo == 0 {
						item.State = ItemOpen
					} else {
						item.State = ItemPartiallyPaid
					}
				}
			}
		}
	}
	return nil
}
func (s *Store) Evaluate(now time.Time) []Item {
	if s.pool != nil {
		return s.evaluatePostgres(now)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Item{}
	for _, items := range s.items {
		for _, item := range items {
			if item.State == ItemPaid || item.State == ItemCancelled {
				continue
			}
			if item.AllocatedKobo > 0 {
				continue
			}
			if now.Before(item.DueAt) {
				item.State = ItemOpen
			} else if now.Before(item.CollectionAt) {
				item.State = ItemInGrace
			} else {
				item.State = ItemOverdue
			}
			out = append(out, *item)
		}
	}
	return out
}

func (s *Store) CollectionTarget(obligationID string, now time.Time) (ledger.Money, error) {
	if s.pool != nil {
		return s.collectionTargetPostgres(obligationID, now)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.byObligation[obligationID]
	if id == "" {
		return 0, errors.New("schedule not found")
	}
	total := ledger.Money(0)
	for _, item := range s.items[id] {
		if item.State != ItemPaid && item.State != ItemCancelled {
			if now.Before(item.DueAt) {
				item.State = ItemOpen
			} else if now.Before(item.CollectionAt) {
				item.State = ItemInGrace
			} else if item.State != ItemPartiallyPaid {
				item.State = ItemOverdue
			}
			if !now.Before(item.CollectionAt) {
				remaining := item.PrincipalDueKobo - item.AllocatedKobo
				if remaining > 0 {
					total += remaining
				}
			}
		}
	}
	return total, nil
}
func (s *Store) MarkCollected(obligationID string, amount ledger.Money) error {
	if s.pool != nil {
		return s.markCollectedPostgres(obligationID, amount, false)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.byObligation[obligationID]
	if id == "" {
		return errors.New("schedule not found")
	}
	remaining := amount
	for _, item := range s.items[id] {
		available := item.AllocatedKobo - item.CollectedKobo
		if available <= 0 {
			continue
		}
		take := available
		if take > remaining {
			take = remaining
		}
		item.CollectedKobo += take
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining > 0 {
		return errors.New("collected amount exceeds allocated schedule amount")
	}
	return nil
}
func (s *Store) ReverseCollected(obligationID string, amount ledger.Money) error {
	if s.pool != nil {
		return s.markCollectedPostgres(obligationID, amount, true)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.byObligation[obligationID]
	if id == "" {
		return errors.New("schedule not found")
	}
	remaining := amount
	for i := len(s.items[id]) - 1; i >= 0; i-- {
		item := s.items[id][i]
		if item.CollectedKobo <= 0 {
			continue
		}
		take := item.CollectedKobo
		if take > remaining {
			take = remaining
		}
		item.CollectedKobo -= take
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining > 0 {
		return errors.New("collection reversal exceeds collected amount")
	}
	return nil
}

func cadenceDate(start time.Time, index int, cadence, monthEnd string, loc *time.Location) (time.Time, error) {
	base := start.In(loc)
	switch cadence {
	case CadenceWeekly:
		return base.AddDate(0, 0, index*7), nil
	case CadenceFortnightly:
		return base.AddDate(0, 0, index*14), nil
	case CadenceMonthly:
		monthIndex := int(base.Month()) - 1 + index
		year := base.Year() + monthIndex/12
		month := time.Month(monthIndex%12 + 1)
		day := base.Day()
		last := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
		if day > last {
			if monthEnd == PolicyLastDay || monthEnd == PolicyCap {
				return time.Date(year, month, last, 0, 0, 0, 0, loc), nil
			}
			return time.Time{}, errors.New("monthly date exceeds month end without policy")
		}
		return time.Date(year, month, day, 0, 0, 0, 0, loc), nil
	case CadenceCustom:
		return base, nil
	default:
		return time.Time{}, errors.New("invalid cadence")
	}
}
func cloneSchedule(v Schedule) Schedule { return v }
func cloneItems(items []*Item) []Item {
	out := make([]Item, 0, len(items))
	for _, item := range items {
		out = append(out, *item)
	}
	return out
}

var identifierCounter int64

func newIdentifier() string {
	return fmt.Sprintf("%d-%d", time.Now().UTC().UnixNano(), atomic.AddInt64(&identifierCounter, 1))
}

func (s *Store) createPostgres(input CreateInput) (Schedule, []Item, error) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Schedule{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	schedule, items, err := s.CreateTx(ctx, tx, input)
	if err != nil {
		return Schedule{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Schedule{}, nil, err
	}
	return schedule, items, nil
}

// CreateTx persists a validated schedule inside a caller-owned transaction.
func (s *Store) CreateTx(ctx context.Context, tx pgx.Tx, input CreateInput) (Schedule, []Item, error) {
	if tx == nil {
		return Schedule{}, nil, errors.New("schedule transaction is required")
	}
	var existing Schedule
	err := tx.QueryRow(ctx, `SELECT id::text,obligation_id::text,schedule_type,timezone,allocation_policy,cadence,grace_hours,status,created_at FROM app.repayment_schedules WHERE obligation_id=$1::uuid`, input.ObligationID).Scan(&existing.ID, &existing.ObligationID, &existing.ScheduleType, &existing.Timezone, &existing.AllocationPolicy, &existing.Cadence, &existing.GraceHours, &existing.Status, &existing.CreatedAt)
	if err == nil {
		rows, err := tx.Query(ctx, `SELECT id::text,schedule_id::text,sequence,principal_due_kobo,due_at,grace_hours,collection_at,allocated_kobo,collected_kobo,state,disputed_kobo,COALESCE(collection_block_reason,'') FROM app.schedule_items WHERE schedule_id=$1::uuid ORDER BY sequence`, existing.ID)
		if err != nil {
			return Schedule{}, nil, err
		}
		defer rows.Close()
		items := []Item{}
		for rows.Next() {
			var item Item
			if err := rows.Scan(&item.ID, &item.ScheduleID, &item.Sequence, &item.PrincipalDueKobo, &item.DueAt, &item.GraceHours, &item.CollectionAt, &item.AllocatedKobo, &item.CollectedKobo, &item.State, &item.DisputedKobo, &item.CollectionBlockReason); err != nil {
				return Schedule{}, nil, err
			}
			items = append(items, item)
		}
		return existing, items, rows.Err()
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, nil, err
	}
	// Reuse the thoroughly-tested validation and date generation path without
	// putting the temporary identifiers into PostgreSQL.
	validated, generated, err := (&Store{schedules: map[string]*Schedule{}, items: map[string][]*Item{}, byObligation: map[string]string{}, now: s.now, newID: s.newID}).Create(input)
	if err != nil {
		return Schedule{}, nil, err
	}
	var schedule Schedule
	err = tx.QueryRow(ctx, `INSERT INTO app.repayment_schedules (obligation_id, schedule_type, timezone, allocation_policy, cadence, grace_hours, status) VALUES ($1::uuid,$2,$3,$4,$5,$6,'ACTIVE') RETURNING id::text, obligation_id::text, schedule_type, timezone, allocation_policy, cadence, grace_hours, status, created_at`, input.ObligationID, validated.ScheduleType, validated.Timezone, validated.AllocationPolicy, validated.Cadence, validated.GraceHours).
		Scan(&schedule.ID, &schedule.ObligationID, &schedule.ScheduleType, &schedule.Timezone, &schedule.AllocationPolicy, &schedule.Cadence, &schedule.GraceHours, &schedule.Status, &schedule.CreatedAt)
	if err != nil {
		return Schedule{}, nil, err
	}
	items := make([]Item, 0, len(generated))
	for _, generatedItem := range generated {
		var item Item
		err = tx.QueryRow(ctx, `INSERT INTO app.schedule_items (schedule_id, sequence, principal_due_kobo, due_at, grace_hours, collection_at, state) VALUES ($1::uuid,$2,$3,$4,$5,$6,'OPEN') RETURNING id::text, schedule_id::text, sequence, principal_due_kobo, due_at, grace_hours, collection_at, allocated_kobo, collected_kobo, state, disputed_kobo, COALESCE(collection_block_reason,'')`, schedule.ID, generatedItem.Sequence, int64(generatedItem.PrincipalDueKobo), generatedItem.DueAt, generatedItem.GraceHours, generatedItem.CollectionAt).
			Scan(&item.ID, &item.ScheduleID, &item.Sequence, &item.PrincipalDueKobo, &item.DueAt, &item.GraceHours, &item.CollectionAt, &item.AllocatedKobo, &item.CollectedKobo, &item.State, &item.DisputedKobo, &item.CollectionBlockReason)
		if err != nil {
			return Schedule{}, nil, err
		}
		items = append(items, item)
	}
	return schedule, items, nil
}

func (s *Store) getPostgres(obligationID string) (Schedule, []Item, error) {
	ctx := context.Background()
	var schedule Schedule
	err := s.pool.QueryRow(ctx, `SELECT id::text, obligation_id::text, schedule_type, timezone, allocation_policy, cadence, grace_hours, status, created_at FROM app.repayment_schedules WHERE obligation_id = $1::uuid`, obligationID).
		Scan(&schedule.ID, &schedule.ObligationID, &schedule.ScheduleType, &schedule.Timezone, &schedule.AllocationPolicy, &schedule.Cadence, &schedule.GraceHours, &schedule.Status, &schedule.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, nil, errors.New("schedule not found")
	}
	if err != nil {
		return Schedule{}, nil, err
	}
	items, err := s.itemsPostgres(schedule.ID)
	return schedule, items, err
}

func (s *Store) itemsPostgres(scheduleID string) ([]Item, error) {
	rows, err := s.pool.Query(context.Background(), `SELECT id::text, schedule_id::text, sequence, principal_due_kobo, due_at, grace_hours, collection_at, allocated_kobo, collected_kobo, state, disputed_kobo, COALESCE(collection_block_reason,'') FROM app.schedule_items WHERE schedule_id = $1::uuid ORDER BY sequence`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.ScheduleID, &item.Sequence, &item.PrincipalDueKobo, &item.DueAt, &item.GraceHours, &item.CollectionAt, &item.AllocatedKobo, &item.CollectedKobo, &item.State, &item.DisputedKobo, &item.CollectionBlockReason); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) allocatePostgres(obligationID string, amount ledger.Money) ([]AllocationTarget, error) {
	if amount <= 0 {
		return nil, errors.New("allocation amount must be positive")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var scheduleID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM app.repayment_schedules WHERE obligation_id = $1::uuid FOR UPDATE`, obligationID).Scan(&scheduleID); errors.Is(err, pgx.ErrNoRows) {
		return []AllocationTarget{{AmountKobo: amount}}, nil
	} else if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text, principal_due_kobo, allocated_kobo, state FROM app.schedule_items WHERE schedule_id = $1::uuid ORDER BY sequence FOR UPDATE`, scheduleID)
	if err != nil {
		return nil, err
	}
	type row struct {
		id             string
		due, allocated ledger.Money
		state          string
	}
	items := make([]row, 0)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.due, &item.allocated, &item.state); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	var capacity ledger.Money
	for _, item := range items {
		if remaining := item.due - item.allocated; remaining > 0 {
			capacity += remaining
		}
	}
	if amount > capacity {
		return nil, errors.New("allocation exceeds schedule outstanding")
	}
	remaining := amount
	targets := make([]AllocationTarget, 0)
	for _, item := range items {
		open := item.due - item.allocated
		if open <= 0 {
			continue
		}
		take := open
		if take > remaining {
			take = remaining
		}
		newAllocated := item.allocated + take
		state := ItemPartiallyPaid
		if newAllocated == item.due {
			state = ItemPaid
		}
		if _, err := tx.Exec(ctx, `UPDATE app.schedule_items SET allocated_kobo = $2, state = $3 WHERE id = $1::uuid`, item.id, int64(newAllocated), state); err != nil {
			return nil, err
		}
		targets = append(targets, AllocationTarget{ScheduleItemID: item.id, AmountKobo: take})
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return targets, nil
}

func (s *Store) reverseAllocationsPostgres(targets []AllocationTarget) error {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, target := range targets {
		var allocated, due int64
		if err := tx.QueryRow(ctx, `SELECT allocated_kobo, principal_due_kobo FROM app.schedule_items WHERE id = $1::uuid FOR UPDATE`, target.ScheduleItemID).Scan(&allocated, &due); errors.Is(err, pgx.ErrNoRows) {
			return errors.New("schedule item not found")
		} else if err != nil {
			return err
		}
		if target.AmountKobo <= 0 || int64(target.AmountKobo) > allocated {
			return errors.New("schedule allocation reversal exceeds allocated amount")
		}
		newAllocated := allocated - int64(target.AmountKobo)
		var state string
		switch newAllocated {
		case 0:
			state = ItemOpen
		case due:
			state = ItemPaid
		default:
			state = ItemPartiallyPaid
		}
		if _, err := tx.Exec(ctx, `UPDATE app.schedule_items SET allocated_kobo = $2, state = $3 WHERE id = $1::uuid`, target.ScheduleItemID, newAllocated, state); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) evaluatePostgres(now time.Time) []Item {
	ctx := context.Background()
	_, _ = s.pool.Exec(ctx, `UPDATE app.schedule_items SET state = CASE WHEN now() < due_at THEN 'OPEN' WHEN now() < collection_at THEN 'IN_GRACE' ELSE 'OVERDUE' END WHERE state NOT IN ('PAID','CANCELLED') AND allocated_kobo = 0 AND due_at <= $1`, now)
	rows, err := s.pool.Query(ctx, `SELECT id::text, schedule_id::text, sequence, principal_due_kobo, due_at, grace_hours, collection_at, allocated_kobo, collected_kobo, state, disputed_kobo, COALESCE(collection_block_reason,'') FROM app.schedule_items WHERE state NOT IN ('PAID','CANCELLED') AND allocated_kobo = 0 ORDER BY collection_at, sequence`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]Item, 0)
	for rows.Next() {
		var item Item
		if rows.Scan(&item.ID, &item.ScheduleID, &item.Sequence, &item.PrincipalDueKobo, &item.DueAt, &item.GraceHours, &item.CollectionAt, &item.AllocatedKobo, &item.CollectedKobo, &item.State, &item.DisputedKobo, &item.CollectionBlockReason) == nil {
			result = append(result, item)
		}
	}
	if rows.Err() != nil {
		return nil
	}
	return result
}

func (s *Store) collectionTargetPostgres(obligationID string, now time.Time) (ledger.Money, error) {
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var scheduleID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM app.repayment_schedules WHERE obligation_id = $1::uuid FOR UPDATE`, obligationID).Scan(&scheduleID); errors.Is(err, pgx.ErrNoRows) {
		return 0, errors.New("schedule not found")
	} else if err != nil {
		return 0, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text, principal_due_kobo, allocated_kobo, state, collection_at, due_at FROM app.schedule_items WHERE schedule_id = $1::uuid ORDER BY sequence FOR UPDATE`, scheduleID)
	if err != nil {
		return 0, err
	}
	var total ledger.Money
	for rows.Next() {
		var id string
		var due, allocated int64
		var state string
		var collectionAt, dueAt time.Time
		if err := rows.Scan(&id, &due, &allocated, &state, &collectionAt, &dueAt); err != nil {
			rows.Close()
			return 0, err
		}
		if state == ItemPaid || state == ItemCancelled {
			continue
		}
		if now.Before(dueAt) {
			state = ItemOpen
		} else if now.Before(collectionAt) {
			state = ItemInGrace
		} else if state != ItemPartiallyPaid {
			state = ItemOverdue
		}
		if _, err := tx.Exec(ctx, `UPDATE app.schedule_items SET state = $2 WHERE id = $1::uuid`, id, state); err != nil {
			rows.Close()
			return 0, err
		}
		if !now.Before(collectionAt) && due > allocated {
			total += ledger.Money(due - allocated)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	return total, tx.Commit(ctx)
}

func (s *Store) markCollectedPostgres(obligationID string, amount ledger.Money, reverse bool) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	order := "sequence ASC"
	if reverse {
		order = "sequence DESC"
	}
	rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT id::text, allocated_kobo, collected_kobo FROM app.schedule_items WHERE schedule_id = (SELECT id FROM app.repayment_schedules WHERE obligation_id = $1::uuid FOR UPDATE) ORDER BY %s FOR UPDATE`, order), obligationID)
	if err != nil {
		return err
	}
	type item struct {
		id                   string
		allocated, collected int64
	}
	items := make([]item, 0)
	for rows.Next() {
		var value item
		if err := rows.Scan(&value.id, &value.allocated, &value.collected); err != nil {
			rows.Close()
			return err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	remaining := int64(amount)
	for _, value := range items {
		available := value.allocated - value.collected
		if reverse {
			available = value.collected
		}
		if available <= 0 {
			continue
		}
		take := available
		if take > remaining {
			take = remaining
		}
		newCollected := value.collected + take
		if reverse {
			newCollected = value.collected - take
		}
		if _, err := tx.Exec(ctx, `UPDATE app.schedule_items SET collected_kobo = $2 WHERE id = $1::uuid`, value.id, newCollected); err != nil {
			return err
		}
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining > 0 {
		if reverse {
			return errors.New("collection reversal exceeds collected amount")
		}
		return errors.New("collected amount exceeds allocated schedule amount")
	}
	return tx.Commit(ctx)
}
