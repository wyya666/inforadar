package constants

import "time"

const (
	RateLimitCacheMaxEntries = 10000
	RateLimitCacheTTL        = time.Hour
	RegisterIPLimit          = 10
	LoginIPLimit             = 20
	ForgotIPLimit            = 20
	ResetIPLimit             = 30
	LoginEmailLimit          = 10
	ForgotEmailLimit         = 3
	RegisterIPPeriod         = time.Hour
	LoginIPPeriod            = 15 * time.Minute
	ForgotIPPeriod           = time.Hour
	ResetIPPeriod            = 30 * time.Minute
	LoginEmailPeriod         = 15 * time.Minute
	ForgotEmailPeriod        = time.Hour
)
