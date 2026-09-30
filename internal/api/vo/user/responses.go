package user

import "time"

type Profile struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Login struct {
	AccessToken string  `json:"accessToken"`
	ExpiresIn   int     `json:"expiresIn"`
	User        Profile `json:"user"`
}

type Message struct {
	Message string `json:"message"`
}
