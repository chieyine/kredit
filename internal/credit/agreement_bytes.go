package credit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

// UnmarshalJSON restores accepted bytes after a JSONB snapshot round trip.
// Base64 survives database whitespace/key normalization without changing the
// hash or trusting mutable display fields beside the canonical agreement.
func (a *AgreementVersion) UnmarshalJSON(data []byte) error {
	type plain AgreementVersion
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*a = AgreementVersion(decoded)
	matches := func(value []byte) bool {
		hash := sha256.Sum256(value)
		return hex.EncodeToString(hash[:]) == a.DocumentHash
	}
	if len(a.CanonicalBytes) > 0 {
		if !json.Valid(a.CanonicalBytes) || !matches(a.CanonicalBytes) {
			return errors.New("accepted agreement bytes do not match the recorded hash")
		}
		a.CanonicalJSON = append(json.RawMessage(nil), a.CanonicalBytes...)
		return nil
	}
	if matches(a.CanonicalJSON) {
		a.CanonicalBytes = append([]byte(nil), a.CanonicalJSON...)
		return nil
	}
	// Older snapshots have only normalized JSON. Recover the known canonical
	// serialization only when it independently matches the original hash. Never
	// replace an accepted hash to make a damaged or unknown version appear valid.
	var terms agreementCanonical
	decoder := json.NewDecoder(bytes.NewReader(a.CanonicalJSON))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&terms) == nil {
		original, err := json.Marshal(terms)
		if err == nil && matches(original) {
			a.CanonicalJSON = original
			a.CanonicalBytes = append([]byte(nil), original...)
		}
	}
	return nil
}
