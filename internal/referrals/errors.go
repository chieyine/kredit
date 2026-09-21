package referrals

import "errors"

var (
	ErrUnavailable     = errors.New("referral persistence is unavailable")
	ErrCodeUnavailable = errors.New("referral code is unavailable")
)
