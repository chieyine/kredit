package web

import (
	"context"
	"kredit/internal/config"
	"kredit/internal/mandates"
	"testing"
)

func TestDisabledMonoCannotCreateMockActiveMandate(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		runtime := NewRuntime(config.Config{Environment: "development", CollectionProvider: "mono-sweep", MonoSweepEnabled: enabled})
		if runtime.Mono != nil {
			t.Fatal("Mono configured without database and credentials")
		}
		if _, err := runtime.Mandates.CreateAuthorizationSession(context.Background(), mandates.AuthorizationInput{UserID: "buyer", BusinessID: "business", AmountCeiling: 50000000}); err == nil {
			t.Fatal("disabled Mono fell back to a mock mandate")
		}
		if runtime.Collections.ProviderStatus().FeatureEnabled {
			t.Fatal("unconfigured Mono collections enabled")
		}
	}
}
