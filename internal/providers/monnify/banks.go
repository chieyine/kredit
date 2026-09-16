package monnify

import (
	"context"
	"kredit/internal/settlement"
)

func (c *Client) Banks(ctx context.Context) ([]settlement.Bank, error) {
	var banks []settlement.Bank
	if e := c.call(ctx, "GET", "/api/v1/banks", nil, &banks); e != nil {
		return nil, e
	}
	if e := settlement.ValidateBanks(banks); e != nil {
		return nil, e
	}
	return banks, nil
}
