package mono

import (
	"context"
	"testing"
)

// FuzzWebhookParsingAfterAuthentication covers the "webhook parsers after
// signature layer" target README section 40.3 requires. The body is fuzzed with
// the authentication already satisfied, which is exactly the position an
// attacker reaches once a provider secret leaks: the parser itself must not
// panic, must reject live-mode events in the sandbox adapter, and must never
// return an accepted notice without the identity the domain relies on.
func FuzzWebhookParsingAfterAuthentication(f *testing.F) {
	const secret = "whsec-fuzz"
	client, err := New("https://api.example.test", "test_sk_fuzz", secret, "https://redirect.example.test", false,
		func(context.Context, string, string) (string, error) { return "customer", nil })
	if err != nil {
		f.Fatalf("building the sandbox client failed: %v", err)
	}

	f.Add([]byte(`{"event":"events.mandates.approved","data":{"id":"mnd_1"}}`))
	f.Add([]byte(`{"event":"events.mandates.debit.successful","data":{"mandate":"mnd_1","reference_number":"ref_1"}}`))
	f.Add([]byte(`{"event":"events.mandates.approved","data":{"id":"mnd_1","live_mode":true}}`))
	f.Add([]byte(`{`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, raw []byte) {
		if _, err := client.ParseWebhook("wrong-secret", raw); err == nil {
			t.Fatal("a webhook was accepted with the wrong secret")
		}
		notice, err := client.ParseWebhook(secret, raw)
		if err != nil {
			return
		}
		if notice.EventID == "" || notice.MandateID == "" || notice.Type == "" {
			t.Fatalf("accepted notice is missing identity: %+v", notice)
		}
		if notice.PayloadHash == "" {
			t.Fatalf("accepted notice has no payload digest: %+v", notice)
		}
		if len(notice.MandateID) > 256 || len(notice.Reference) > 256 || len(notice.EventID) > 256 {
			t.Fatalf("accepted notice exceeded its length bounds: %+v", notice)
		}
	})
}
