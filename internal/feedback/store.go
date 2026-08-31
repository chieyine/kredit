package feedback

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const EventName = "feedback.clarity_submitted"

type Input struct {
	UserID         string `json:"-"`
	OrganizationID string `json:"organization_id,omitempty"`
	Area           string `json:"area"`
	Screen         string `json:"screen"`
	Answer         string `json:"answer"`
}

type Entry struct {
	Area           string    `json:"area"`
	Screen         string    `json:"screen"`
	Answer         string    `json:"answer"`
	OrganizationID string    `json:"organization_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type Store struct {
	pool *pgxpool.Pool
	mu   sync.RWMutex
	rows []Entry
	seen map[string]bool
	now  func() time.Time
}

func NewStore() *Store {
	return &Store{seen: make(map[string]bool), now: func() time.Time { return time.Now().UTC() }}
}

func NewPostgresStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, seen: make(map[string]bool), now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) Submit(ctx context.Context, input Input) (Entry, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.OrganizationID = strings.TrimSpace(input.OrganizationID)
	input.Area = strings.TrimSpace(input.Area)
	input.Screen = strings.TrimSpace(input.Screen)
	input.Answer = strings.TrimSpace(input.Answer)
	if _, err := uuid.Parse(input.UserID); err != nil {
		return Entry{}, errors.New("a valid user is required")
	}
	if input.OrganizationID != "" {
		if _, err := uuid.Parse(input.OrganizationID); err != nil {
			return Entry{}, errors.New("organization_id must be a UUID")
		}
	}
	if input.Area != "seller" && input.Area != "buyer" {
		return Entry{}, errors.New("area must be seller or buyer")
	}
	if input.Screen != "overview" {
		return Entry{}, errors.New("screen must be overview")
	}
	if input.Answer != "yes" && input.Answer != "partly" && input.Answer != "no" {
		return Entry{}, errors.New("answer must be yes, partly or no")
	}
	if input.Area == "seller" && input.OrganizationID == "" {
		return Entry{}, errors.New("organization_id is required for seller feedback")
	}
	if input.Area == "buyer" && input.OrganizationID != "" {
		return Entry{}, errors.New("organization_id is not used for buyer feedback")
	}
	entry := Entry{Area: input.Area, Screen: input.Screen, Answer: input.Answer, OrganizationID: input.OrganizationID, CreatedAt: s.now()}
	digest := sha256.Sum256([]byte(strings.Join([]string{input.UserID, input.OrganizationID, input.Area, input.Screen, entry.CreatedAt.Format("2006-01")}, "|")))
	deduplication := "feedback:monthly:" + hex.EncodeToString(digest[:])
	if s.pool != nil {
		_, err := s.pool.Exec(ctx, `SELECT app.record_product_event($1,$2::uuid,NULLIF($3,'')::uuid,'product_improvement',$4,$5,jsonb_build_object('area',$6::text,'screen',$7::text,'answer',$8::text))`, EventName, input.UserID, input.OrganizationID, entry.CreatedAt, deduplication, input.Area, input.Screen, input.Answer)
		if err != nil {
			return Entry{}, err
		}
		return entry, nil
	}
	s.mu.Lock()
	if !s.seen[deduplication] {
		s.rows = append(s.rows, entry)
		s.seen[deduplication] = true
	}
	s.mu.Unlock()
	return entry, nil
}
