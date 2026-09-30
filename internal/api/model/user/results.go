package user

import "time"

// Profile and SessionResult are internal service results. HTTP responses use api/vo/user.
type Profile struct {
	ID          string
	Email       string
	DisplayName string
	CreatedAt   time.Time
}

type SessionResult struct {
	AccessToken string
	ExpiresIn   int
	User        Profile
	Refresh     string
}
