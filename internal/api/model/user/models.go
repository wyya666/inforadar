package user

import (
	"time"

	"inforadar/internal/constants"
)

type User struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Account      string    `gorm:"column:user_account;size:256;not null;uniqueIndex:uk_user_account"`
	Email        string    `gorm:"column:email;size:254;not null;uniqueIndex:uk_user_email"`
	PasswordHash string    `gorm:"column:user_password;size:512;not null"`
	DisplayName  string    `gorm:"column:user_name;size:256;index:idx_user_name"`
	Avatar       *string   `gorm:"column:user_avatar;size:1024"`
	Bio          *string   `gorm:"column:user_profile;size:512"`
	Role         string    `gorm:"column:user_role;size:256;not null;default:user"`
	EditTime     time.Time `gorm:"column:edit_time;autoCreateTime"`
	CreatedAt    time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:update_time;autoUpdateTime"`
	IsDelete     bool      `gorm:"column:is_delete;not null;default:false"`
}

func (User) TableName() string { return constants.TableUser }

type Session struct {
	ID               string     `gorm:"primaryKey;size:32"`
	UserID           uint64     `gorm:"not null;index"`
	RefreshTokenHash string     `gorm:"size:64;not null;uniqueIndex:uq_auth_sessions_refresh_token_hash"`
	ExpiresAt        time.Time  `gorm:"not null"`
	RevokedAt        *time.Time `gorm:"column:revoked_at"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Session) TableName() string { return constants.TableSessions }

type ActionToken struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement"`
	UserID     uint64     `gorm:"not null;index"`
	Purpose    string     `gorm:"size:32;not null"`
	TokenHash  string     `gorm:"size:64;not null;uniqueIndex:uq_action_tokens_token_hash"`
	ExpiresAt  time.Time  `gorm:"not null"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time
}

func (ActionToken) TableName() string { return constants.TableActionTokens }
