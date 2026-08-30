package whatsapp

import "testing"

func FuzzParseAmountNeverPanics(f *testing.F) {
	for _, seed := range []string{"1", "₦500,000", "1.25m", "0.001", "-1", "", "999999999999999999999999"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = parseAmount(input)
	})
}

func FuzzParseCommandNeverPanics(f *testing.F) {
	for _, seed := range []string{"How much am I owed?", "Royal Pharmacy paid ₦500,000", "create credit Royal Pharmacy, ₦1.2m, due 30 September 2026"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseCommand(input)
	})
}
