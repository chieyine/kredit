package identity

import "testing"

func TestVerificationFactsDiscardRawIdentityAndInvalidStatus(t *testing.T) {
	result := SafeVerificationResult(map[string]string{
		"bvn": "12345678901", "account_number": "0123456789", "nin": "12345678901",
		"bvn_status": "verified", "nin_status": "pending", "cac_status": "verified",
		"directors_status": "review", "shareholders_status": "verified", "bank_account_status": "verified",
		"credit_history_status": "12345678901", "verified_name": "12345678901", "verified_role": "Director",
	})
	if len(result) != 7 || result["bvn_status"] != "verified" || result["nin_status"] != "pending" {
		t.Fatalf("unexpected safe facts: %#v", result)
	}
	for _, key := range []string{"bvn", "account_number", "nin", "credit_history_status", "verified_name"} {
		if _, ok := result[key]; ok {
			t.Fatalf("unsafe field retained: %s", key)
		}
	}
}
