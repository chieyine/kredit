package collections

import (
	"context"
	"testing"

	"kredit/internal/ledger"
)

func TestMockProviderContractScenarios(t *testing.T) {
	provider := NewMockProvider("contract-secret")
	request := Request{ExternalReference: "kredit-attempt-1", ObligationID: "obligation-1", BuyerUserID: "buyer-1", AmountKobo: 1000, Currency: "NGN"}
	for _, scenario := range []struct {
		name     string
		response Response
		want     string
	}{
		{name: "success", response: Response{State: ProviderSucceeded, SucceededAmountKobo: 1000}, want: ProviderSucceeded},
		{name: "pending", response: Response{State: ProviderPending}, want: ProviderPending},
		{name: "partial", response: Response{State: ProviderPartial, SucceededAmountKobo: 400}, want: ProviderPartial},
		{name: "retryable failure", response: Response{State: ProviderFailed, FailureCode: "timeout", Retryable: true}, want: ProviderFailed},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			provider.SetNextResponse(scenario.response)
			response, err := provider.Submit(context.Background(), request)
			if err != nil || response.State != scenario.want {
				t.Fatalf("response=%#v err=%v", response, err)
			}
		})
	}
	event := Webhook{EventID: "provider-event-1", ExternalReference: request.ExternalReference, State: ProviderSettled, SettlementState: ProviderSettled, SettlementReference: "settlement-1", SucceededAmountKobo: ledger.Money(1000)}
	event.Signature = provider.Sign(event)
	if !provider.VerifyWebhook(event) {
		t.Fatal("expected valid provider signature")
	}
	event.Signature = "forged"
	if provider.VerifyWebhook(event) {
		t.Fatal("expected forged signature rejection")
	}
}
