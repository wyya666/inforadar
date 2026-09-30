package repositoryinterface

import (
	"context"
	"time"

	usermodel "inforadar/internal/api/model/user"
)

// UserRepository defines the account data operations needed by the service.
// Transaction callbacks receive the same contract backed by one DB transaction.
type UserRepository interface {
	WithinTransaction(ctx context.Context, fn func(UserRepository) error) error
	CreateUser(ctx context.Context, user *usermodel.User) error
	UserByEmail(ctx context.Context, email string) (usermodel.User, error)
	UserByID(ctx context.Context, id uint64) (usermodel.User, error)
	SetPasswordHash(ctx context.Context, userID uint64, hash string) error
	SetPasswordHashIfMatch(ctx context.Context, userID uint64, oldHash, newHash string) (bool, error)
	SetDisplayName(ctx context.Context, userID uint64, name string) error
	CreateActionToken(ctx context.Context, token *usermodel.ActionToken) error
	ActionTokenByHashForUpdate(ctx context.Context, hash, purpose string) (usermodel.ActionToken, error)
	ConsumeActionToken(ctx context.Context, id uint64, at time.Time) error
	ConsumeAllActionTokens(ctx context.Context, userID uint64, purpose string, at time.Time) error
	CreateSession(ctx context.Context, session *usermodel.Session) error
	SessionByRefreshHashForUpdate(ctx context.Context, hash string) (usermodel.Session, error)
	SessionByID(ctx context.Context, id string, userID uint64) (usermodel.Session, error)
	RotateRefreshToken(ctx context.Context, id, newHash string) error
	RevokeSessionByRefreshHash(ctx context.Context, hash string, at time.Time) error
	RevokeUserSessions(ctx context.Context, userID uint64, at time.Time) error
}

// UserStore owns the database connection used by the repository.
// Transaction-scoped repositories only implement UserRepository.
type UserStore interface {
	UserRepository
	Radar() RadarRepository
	Close() error
}
