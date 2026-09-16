package bankdebit

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

// Naira emits an exact JSON decimal; binary floats never enter the money path.
func Naira(kobo int64) json.Number { return json.Number(fmt.Sprintf("%d.%02d", kobo/100, kobo%100)) }
func Kobo(value json.Number) (int64, error) {
	n, ok := new(big.Rat).SetString(string(value))
	if !ok || n.Sign() < 0 {
		return 0, errors.New("invalid provider amount")
	}
	n.Mul(n, big.NewRat(100, 1))
	if !n.IsInt() || !n.Num().IsInt64() {
		return 0, errors.New("provider amount cannot be represented in kobo")
	}
	return n.Num().Int64(), nil
}
