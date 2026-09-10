package feedback

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresFeedbackReturnsStoredMonthlyAnswer(t *testing.T) {
	if os.Getenv("KREDIT_INTEGRATION") != "1" || os.Getenv("DATABASE_URL") == "" {
		t.Skip("isolated integration database required")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	user := uuid.NewString()
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM app.analytics_events WHERE name=$1 AND subject_id_hash=encode(digest($2,'sha256'),'hex')`, EventName, user)
	}()
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["role"] = "kredit_app"
	runtimePool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtimePool.Close()
	store := NewPostgresStore(runtimePool)
	input := Input{UserID: user, Area: "buyer", Screen: "overview", Answer: "yes"}
	original, err := store.Submit(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.Answer = "no"
	replay, err := store.Submit(ctx, input)
	if err != nil || replay.Answer != original.Answer || !replay.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("replay differs from stored feedback: %+v, %v", replay, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Submit(cancelled, input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("database failure lost its classification: %v", err)
	}
}
