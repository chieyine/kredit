package buyers

import (
	"kredit/internal/identity"
	"sync"
	"testing"
)

func TestImportedInvitationConcurrentReplayAndChangedDetails(t *testing.T) {
	store := NewStore("import-fixture", identity.NewMockProvider())
	input := CreateInvitationInput{SourceReference: "roster:customer-1", Target: "owner@example.test", TargetType: "email", LegalName: "Distributor", BusinessType: "limited_company", BusinessAddress: "Lagos", Industry: "food"}
	results := make(chan CreateInvitationResult, 8)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			r, err := store.CreateInvitation("owner", "manufacturer", input)
			if err != nil {
				t.Error(err)
				return
			}
			results <- r
		}()
	}
	workers.Wait()
	close(results)
	token := ""
	created := 0
	for result := range results {
		if token == "" {
			token = result.RawToken
		}
		if token != result.RawToken {
			t.Fatal("replay created another invitation")
		}
		if !result.Replayed {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("created %d invitations", created)
	}
	changed := input
	changed.LegalName = "Different business"
	if _, err := store.CreateInvitation("owner", "manufacturer", changed); err == nil {
		t.Fatal("reused source reference with different payload")
	}
	other, err := store.CreateInvitation("other-owner", "another-manufacturer", input)
	if err != nil || other.RawToken == token {
		t.Fatalf("import identity crossed supplier boundary: %v", err)
	}
}
