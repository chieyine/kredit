package identity

import "strings"

// SafeVerificationResult is the persisted evidence contract, independent of a
// buyer profile. Each case already records provider, reference and timestamps;
// these facts distinguish checks without retaining raw identity/account data.
// Directors/shareholders remain provider evidence, not an imported PII roster.
func SafeVerificationResult(input map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range input {
		switch key {
		case "bvn_status", "nin_status", "cac_status", "bank_account_status", "directors_status", "shareholders_status", "credit_history_status":
			switch value {
			case "pending", "in_progress", "verified", "failed", "expired", "review", "not_applicable":
				result[key] = value
			}
		case "verified_name", "verified_role":
			// Retain only bounded display evidence. A number accidentally returned in
			// one of these fields is not safe merely because the key is allowlisted.
			if len(value) <= 200 && !hasLongDigitRun(value) {
				result[key] = strings.TrimSpace(value)
			}
		}
	}
	return result
}
func hasLongDigitRun(value string) bool {
	count := 0
	for _, r := range value {
		if r >= '0' && r <= '9' {
			count++
			if count >= 10 {
				return true
			}
		} else {
			count = 0
		}
	}
	return false
}
