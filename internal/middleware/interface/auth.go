package middlewareinterface

import (
	"context"

	usermodel "inforadar/internal/api/model/user"
)

// Authenticator checks an access JWT and its backing session.
type Authenticator interface {
	Authenticate(context.Context, string) (usermodel.Profile, uint64, error)
}
