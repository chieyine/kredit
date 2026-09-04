package outbox

import (
	"testing"
	"time"
)

func TestRetryDelayIsBounded(t *testing.T) {
	got1 := retryDelay(1)
	if got1 < 5*time.Second || got1 > 7*time.Second {
		t.Fatalf("first retry out of bounded range: %s", got1)
	}
	maxBase := 10*time.Minute + 40*time.Second
	gotMax := retryDelay(100)
	if gotMax < maxBase || gotMax > maxBase+3*time.Minute {
		t.Fatalf("bounded retry out of bounded range: %s", gotMax)
	}
}
