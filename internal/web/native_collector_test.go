package web

import (
	"context"
	"kredit/internal/config"
	"kredit/internal/mandates"
	"testing"
)

func TestNativeCollectorsNeverFallBackToMockAuthorization(t *testing.T) {
	for _, adapter := range []string{"paystack", "flutterwave", "monnify"} {
		t.Run(adapter, func(t *testing.T) {
			runtime := NewRuntime(config.Config{Environment: "development", CollectionAdapter: adapter, CollectionProvider: adapter + "-main", RealCollections: true})
			if _, err := runtime.Mandates.CreateAuthorizationSession(context.Background(), mandates.AuthorizationInput{UserID: "buyer", BusinessID: "business", AmountCeiling: 10000}); err == nil {
				t.Fatal("unconfigured collector created a mock permission")
			}
			if runtime.Collections.ProviderStatus().FeatureEnabled {
				t.Fatal("unconfigured native collector enabled money movement")
			}
		})
	}
}
