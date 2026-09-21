package credit

import "errors"

// ErrRequestNotFound deliberately does not disclose whether another tenant owns the request.
var ErrRequestNotFound = errors.New("credit request not found")
