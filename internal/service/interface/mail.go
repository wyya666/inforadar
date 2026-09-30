package serviceinterface

// Mailer delivers one-time account action links.
type Mailer interface {
	SendPasswordReset(email, token string) error
}
