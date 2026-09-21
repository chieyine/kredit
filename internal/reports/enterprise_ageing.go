package reports

import (
	"errors"
	"math"
	"time"

	"kredit/internal/ledger"
	"kredit/internal/schedules"
)

// Use each remaining instalment, not the obligation's single oldest bucket.
// The first bucket includes current/future amounts; the others are days overdue.
func enterpriseAgeing(outstanding ledger.Money, items []schedules.Item, grace int, now time.Time) (map[string]ledger.Money, error) {
	buckets := map[string]ledger.Money{"0-30": 0, "31-60": 0, "61-90": 0, "90+": 0}
	if outstanding < 0 || now.IsZero() {
		return nil, errors.New("invalid ageing inputs")
	}
	if outstanding == 0 {
		return buckets, nil
	}
	if grace < 0 || int64(grace) > math.MaxInt64/int64(time.Hour) {
		return nil, errors.New("invalid schedule grace interval")
	}
	var classified ledger.Money
	for _, item := range items {
		if item.PrincipalDueKobo < 0 || item.AllocatedKobo < 0 || item.AllocatedKobo > item.PrincipalDueKobo {
			return nil, errors.New("invalid schedule amounts")
		}
		if item.State == schedules.ItemCancelled {
			continue
		}
		amount := item.PrincipalDueKobo - item.AllocatedKobo
		if amount == 0 {
			continue
		}
		if item.State == schedules.ItemPaid {
			return nil, errors.New("paid instalment has an unpaid balance")
		}
		deadline := item.CollectionAt
		if deadline.IsZero() {
			if item.DueAt.IsZero() {
				return nil, errors.New("instalment due date is missing")
			}
			deadline = item.DueAt.Add(time.Duration(grace) * time.Hour)
		}
		bucket := "0-30"
		if days := int64(now.Sub(deadline) / (24 * time.Hour)); days > 90 {
			bucket = "90+"
		} else if days > 60 {
			bucket = "61-90"
		} else if days > 30 {
			bucket = "31-60"
		}
		var err error
		buckets[bucket], err = ledger.CheckedAdd(buckets[bucket], amount)
		if err != nil {
			return nil, err
		}
		classified, err = ledger.CheckedAdd(classified, amount)
		if err != nil {
			return nil, err
		}
	}
	if classified != outstanding {
		return nil, errors.New("schedule does not reconcile to portfolio outstanding")
	}
	return buckets, nil
}
