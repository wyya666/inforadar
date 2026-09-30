package serviceinterface

import (
	"context"

	usermodel "inforadar/internal/api/model/user"
)

// UserService defines the account use cases used by the HTTP layer.
type UserService interface {
	Register(context.Context, string, string) error
	Login(context.Context, string, string) (usermodel.SessionResult, error)
	Refresh(context.Context, string) (usermodel.SessionResult, error)
	Logout(context.Context, string) error
	Authenticate(context.Context, string) (usermodel.Profile, uint64, error)
	RequestPasswordReset(context.Context, string) error
	ResetPassword(context.Context, string, string) error
	UpdateProfile(context.Context, uint64, string) (usermodel.Profile, error)
	ChangePassword(context.Context, uint64, string, string) error
}
