package outbox

import (
	"testing"
	"time"
)

func TestRetryDelayIsBounded(t *testing.T) {
	if got := retryDelay(1); got != 5*time.Second {
		t.Fatalf("first retry = %s", got)
	}
	if got := retryDelay(100); got != 10*time.Minute+40*time.Second {
		t.Fatalf("bounded retry = %s", got)
	}
}
