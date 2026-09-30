package constants

import "time"

const (
	AccessTTL            = 15 * time.Minute
	RefreshTTL           = 30 * 24 * time.Hour
	ResetPasswordTTL     = 30 * time.Minute
	AccessIssuer         = "inforadar"
	AccessAudience       = "inforadar-web"
	ResetPasswordPurpose = "reset_password"
	RoleUser             = "user"
	RoleAdmin            = "admin"
	RefreshCookieName    = "inforadar_refresh"
	RefreshCookiePath    = "/api/v1/auth"
	EmailMaxBytes        = 254
	PasswordMinBytes     = 8
	PasswordMaxBytes     = 128
	DisplayNameMaxRunes  = 256
	ActionTokenLength    = 64
)
