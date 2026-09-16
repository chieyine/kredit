package flutterwave

import (
	"context"
	"kredit/internal/settlement"
)

// Direct-debit support is narrower than Flutterwave's general bank directory.
// Source: https://developer.flutterwave.com/docs/direct-debit (2026-09-16).
func (c *Client) Banks(context.Context) ([]settlement.Bank, error) {
	return []settlement.Bank{
		{Code: "011", Name: "First Bank"}, {Code: "023", Name: "Citi Bank"}, {Code: "032", Name: "Union Bank"}, {Code: "033", Name: "United Bank for Africa"}, {Code: "035", Name: "Wema Bank"}, {Code: "044", Name: "Access Bank"}, {Code: "050", Name: "Ecobank"}, {Code: "057", Name: "Zenith Bank"}, {Code: "058", Name: "Guaranty Trust Bank"}, {Code: "068", Name: "Standard Chartered Bank"}, {Code: "070", Name: "Fidelity Bank"}, {Code: "076", Name: "Polaris Bank"}, {Code: "082", Name: "Keystone Bank"}, {Code: "100", Name: "Suntrust Bank"}, {Code: "101", Name: "ProvidusBank"}, {Code: "214", Name: "First City Monument Bank"}, {Code: "215", Name: "Unity Bank"}, {Code: "221", Name: "Stanbic IBTC Bank"}, {Code: "232", Name: "Sterling Bank"}, {Code: "301", Name: "Jaiz Bank"}, {Code: "000025", Name: "Titan Trust Bank"}, {Code: "000027", Name: "Globus Bank"}, {Code: "000031", Name: "PremiumTrust Bank"},
	}, nil
}
