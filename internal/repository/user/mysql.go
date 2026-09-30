package user

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	usermodel "inforadar/internal/api/model/user"
	"inforadar/internal/common"
	repositoryinterface "inforadar/internal/repository/interface"
)

type Repository struct {
	db *gorm.DB
}

var _ repositoryinterface.UserRepository = (*Repository)(nil)

func New(db *gorm.DB) repositoryinterface.UserRepository { return &Repository{db: db} }

func (r *Repository) WithinTransaction(ctx context.Context, fn func(repositoryinterface.UserRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repository{db: tx})
	})
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return common.ErrNotFound
	}
	return err
}

func (r *Repository) CreateUser(ctx context.Context, user *usermodel.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return common.ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *Repository) UserByEmail(ctx context.Context, email string) (usermodel.User, error) {
	var row usermodel.User
	if err := r.db.WithContext(ctx).Where("email = ? AND is_delete = 0", email).First(&row).Error; err != nil {
		return usermodel.User{}, notFound(err)
	}
	return row, nil
}

func (r *Repository) UserByID(ctx context.Context, id uint64) (usermodel.User, error) {
	var row usermodel.User
	if err := r.db.WithContext(ctx).Where("is_delete = 0").First(&row, id).Error; err != nil {
		return usermodel.User{}, notFound(err)
	}
	return row, nil
}

func (r *Repository) SetPasswordHash(ctx context.Context, userID uint64, hash string) error {
	return r.db.WithContext(ctx).Model(&usermodel.User{}).Where("id = ? AND is_delete = 0", userID).
		Updates(map[string]any{"user_password": hash, "edit_time": time.Now().UTC()}).Error
}

func (r *Repository) SetPasswordHashIfMatch(ctx context.Context, userID uint64, oldHash, newHash string) (bool, error) {
	result := r.db.WithContext(ctx).Model(&usermodel.User{}).
		Where("id = ? AND user_password = ? AND is_delete = 0", userID, oldHash).
		Updates(map[string]any{"user_password": newHash, "edit_time": time.Now().UTC()})
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) SetDisplayName(ctx context.Context, userID uint64, name string) error {
	return r.db.WithContext(ctx).Model(&usermodel.User{}).Where("id = ? AND is_delete = 0", userID).
		Updates(map[string]any{"user_name": name, "edit_time": time.Now().UTC()}).Error
}

func (r *Repository) CreateActionToken(ctx context.Context, token *usermodel.ActionToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *Repository) ActionTokenByHashForUpdate(ctx context.Context, hash, purpose string) (usermodel.ActionToken, error) {
	var row usermodel.ActionToken
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("token_hash = ? AND purpose = ?", hash, purpose).First(&row).Error
	if err != nil {
		return usermodel.ActionToken{}, notFound(err)
	}
	return row, nil
}

func (r *Repository) ConsumeActionToken(ctx context.Context, id uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&usermodel.ActionToken{}).Where("id = ?", id).Update("consumed_at", at).Error
}

func (r *Repository) ConsumeAllActionTokens(ctx context.Context, userID uint64, purpose string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&usermodel.ActionToken{}).
		Where("user_id = ? AND purpose = ? AND consumed_at IS NULL", userID, purpose).
		Update("consumed_at", at).Error
}

func (r *Repository) CreateSession(ctx context.Context, session *usermodel.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *Repository) SessionByRefreshHashForUpdate(ctx context.Context, hash string) (usermodel.Session, error) {
	var row usermodel.Session
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("refresh_token_hash = ?", hash).First(&row).Error
	if err != nil {
		return usermodel.Session{}, notFound(err)
	}
	return row, nil
}

func (r *Repository) SessionByID(ctx context.Context, id string, userID uint64) (usermodel.Session, error) {
	var row usermodel.Session
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		return usermodel.Session{}, notFound(err)
	}
	return row, nil
}

func (r *Repository) RotateRefreshToken(ctx context.Context, id, newHash string) error {
	return r.db.WithContext(ctx).Model(&usermodel.Session{}).Where("id = ?", id).
		Update("refresh_token_hash", newHash).Error
}

func (r *Repository) RevokeSessionByRefreshHash(ctx context.Context, hash string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&usermodel.Session{}).
		Where("refresh_token_hash = ? AND revoked_at IS NULL", hash).Update("revoked_at", at).Error
}

func (r *Repository) RevokeUserSessions(ctx context.Context, userID uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&usermodel.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", at).Error
}
