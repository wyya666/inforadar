package serviceinterface

import "errors"

var (
	ErrProviderUnauthorized   = errors.New("provider authentication failed")
	ErrProviderInvalidRequest = errors.New("provider rejected request")
	ErrProviderLimited        = errors.New("provider rate limited")
	ErrProviderUnavailable    = errors.New("provider unavailable")
)
