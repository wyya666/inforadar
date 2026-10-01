package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	usermodel "inforadar/internal/api/model/user"
	"inforadar/internal/common"
	"inforadar/internal/constants"
	repositoryinterface "inforadar/internal/repository/interface"
	serviceinterface "inforadar/internal/service/interface"
	cryptoutil "inforadar/internal/utils/crypto"
)

type Service struct {
	repository repositoryinterface.UserRepository
	mailer     serviceinterface.Mailer
	jwtSecret  []byte
	now        func() time.Time
	dummyHash  string
}

func NewService(repository repositoryinterface.UserRepository, mailer serviceinterface.Mailer, jwtSecret string) (serviceinterface.UserService, error) {
	dummy, err := cryptoutil.HashPassword("not-a-real-password")
	if err != nil {
		return nil, err
	}
	return &Service{repository: repository, mailer: mailer, jwtSecret: []byte(jwtSecret), now: time.Now, dummyHash: dummy}, nil
}

func profile(user usermodel.User) usermodel.Profile {
	return usermodel.Profile{
		ID: strconv.FormatUint(user.ID, 10), Email: user.Email,
		DisplayName: user.DisplayName, CreatedAt: user.CreatedAt,
	}
}

func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) == 0 || len(email) > constants.EmailMaxBytes || strings.ContainsAny(email, " \t\r\n") {
		return "", common.ErrInvalidInput
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || !strings.Contains(email, "@") {
		return "", common.ErrInvalidInput
	}
	return email, nil
}

func validatePassword(password string) error {
	if n := len([]byte(password)); n < constants.PasswordMinBytes || n > constants.PasswordMaxBytes {
		return errors.New(constants.MessagePasswordLength)
	}
	return nil
}

func (s *Service) Register(ctx context.Context, rawEmail, password string) error {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return fmt.Errorf("%w: %v", common.ErrInvalidInput, err)
	}
	hash, err := cryptoutil.HashPassword(password)
	if err != nil {
		return err
	}
	name := strings.SplitN(email, "@", 2)[0]
	if utf8.RuneCountInString(name) > constants.DisplayNameMaxRunes {
		name = string([]rune(name)[:constants.DisplayNameMaxRunes])
	}
	user := usermodel.User{Account: email, Email: email, DisplayName: name, PasswordHash: hash, Role: constants.RoleUser}
	return s.repository.CreateUser(ctx, &user)
}

func (s *Service) validActionToken(ctx context.Context, tx repositoryinterface.UserRepository, raw, purpose string) (usermodel.ActionToken, error) {
	if len(raw) != constants.ActionTokenLength {
		return usermodel.ActionToken{}, common.ErrInvalidToken
	}
	entry, err := tx.ActionTokenByHashForUpdate(ctx, cryptoutil.TokenHash(raw), purpose)
	if errors.Is(err, common.ErrNotFound) {
		return usermodel.ActionToken{}, common.ErrInvalidToken
	}
	if err != nil {
		return usermodel.ActionToken{}, err
	}
	if entry.ConsumedAt != nil || !entry.ExpiresAt.After(s.now().UTC()) {
		return usermodel.ActionToken{}, common.ErrInvalidToken
	}
	return entry, nil
}

func (s *Service) Login(ctx context.Context, rawEmail, password string) (usermodel.SessionResult, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return usermodel.SessionResult{}, common.ErrInvalidCredentials
	}
	user, err := s.repository.UserByEmail(ctx, email)
	if errors.Is(err, common.ErrNotFound) {
		cryptoutil.VerifyPassword(s.dummyHash, password)
		return usermodel.SessionResult{}, common.ErrInvalidCredentials
	}
	if err != nil {
		return usermodel.SessionResult{}, err
	}
	if !cryptoutil.VerifyPassword(user.PasswordHash, password) {
		return usermodel.SessionResult{}, common.ErrInvalidCredentials
	}
	sessionID, err := cryptoutil.NewSessionID()
	if err != nil {
		return usermodel.SessionResult{}, err
	}
	refresh, err := cryptoutil.NewRandomToken()
	if err != nil {
		return usermodel.SessionResult{}, err
	}
	now := s.now().UTC()
	access, err := cryptoutil.SignAccess(s.jwtSecret, user.ID, sessionID, now)
	if err != nil {
		return usermodel.SessionResult{}, err
	}
	session := usermodel.Session{ID: sessionID, UserID: user.ID, RefreshTokenHash: cryptoutil.TokenHash(refresh), ExpiresAt: now.Add(constants.RefreshTTL)}
	if err := s.repository.CreateSession(ctx, &session); err != nil {
		return usermodel.SessionResult{}, err
	}
	return usermodel.SessionResult{AccessToken: access, ExpiresIn: int(constants.AccessTTL.Seconds()), User: profile(user), Refresh: refresh}, nil
}

func (s *Service) Refresh(ctx context.Context, raw string) (usermodel.SessionResult, error) {
	if len(raw) != constants.ActionTokenLength {
		return usermodel.SessionResult{}, common.ErrNotLogin
	}
	var result usermodel.SessionResult
	err := s.repository.WithinTransaction(ctx, func(tx repositoryinterface.UserRepository) error {
		session, err := tx.SessionByRefreshHashForUpdate(ctx, cryptoutil.TokenHash(raw))
		if errors.Is(err, common.ErrNotFound) {
			return common.ErrNotLogin
		}
		if err != nil {
			return err
		}
		now := s.now().UTC()
		if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
			return common.ErrNotLogin
		}
		user, err := tx.UserByID(ctx, session.UserID)
		if err != nil {
			return err
		}
		newRefresh, err := cryptoutil.NewRandomToken()
		if err != nil {
			return err
		}
		access, err := cryptoutil.SignAccess(s.jwtSecret, user.ID, session.ID, now)
		if err != nil {
			return err
		}
		if err := tx.RotateRefreshToken(ctx, session.ID, cryptoutil.TokenHash(newRefresh)); err != nil {
			return err
		}
		result = usermodel.SessionResult{AccessToken: access, ExpiresIn: int(constants.AccessTTL.Seconds()), User: profile(user), Refresh: newRefresh}
		return nil
	})
	return result, err
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	if len(raw) != constants.ActionTokenLength {
		return nil
	}
	return s.repository.RevokeSessionByRefreshHash(ctx, cryptoutil.TokenHash(raw), s.now().UTC())
}

func (s *Service) Authenticate(ctx context.Context, raw string) (usermodel.Profile, uint64, error) {
	userID, sessionID, err := cryptoutil.ParseAccess(s.jwtSecret, raw)
	if err != nil {
		return usermodel.Profile{}, 0, common.ErrNotLogin
	}
	session, err := s.repository.SessionByID(ctx, sessionID, userID)
	if errors.Is(err, common.ErrNotFound) {
		return usermodel.Profile{}, 0, common.ErrNotLogin
	}
	if err != nil {
		return usermodel.Profile{}, 0, err
	}
	if session.RevokedAt != nil || !session.ExpiresAt.After(s.now().UTC()) {
		return usermodel.Profile{}, 0, common.ErrNotLogin
	}
	user, err := s.repository.UserByID(ctx, userID)
	if errors.Is(err, common.ErrNotFound) {
		return usermodel.Profile{}, 0, common.ErrNotLogin
	}
	if err != nil {
		return usermodel.Profile{}, 0, err
	}
	return profile(user), user.ID, nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, rawEmail string) error {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	if s.mailer == nil {
		return common.ErrPasswordResetUnavailable
	}
	user, err := s.repository.UserByEmail(ctx, email)
	if errors.Is(err, common.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	rawToken, err := cryptoutil.NewRandomToken()
	if err != nil {
		return err
	}
	entry := usermodel.ActionToken{UserID: user.ID, Purpose: constants.ResetPasswordPurpose, TokenHash: cryptoutil.TokenHash(rawToken), ExpiresAt: s.now().UTC().Add(constants.ResetPasswordTTL)}
	if err := s.repository.CreateActionToken(ctx, &entry); err != nil {
		return err
	}
	if err := s.mailer.SendPasswordReset(email, rawToken); err != nil {
		return fmt.Errorf("%w: %v", common.ErrMailUnavailable, err)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if err := validatePassword(password); err != nil {
		return fmt.Errorf("%w: %v", common.ErrInvalidInput, err)
	}
	hash, err := cryptoutil.HashPassword(password)
	if err != nil {
		return err
	}
	return s.repository.WithinTransaction(ctx, func(tx repositoryinterface.UserRepository) error {
		entry, err := s.validActionToken(ctx, tx, token, constants.ResetPasswordPurpose)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		if err := tx.SetPasswordHash(ctx, entry.UserID, hash); err != nil {
			return err
		}
		if err := tx.ConsumeAllActionTokens(ctx, entry.UserID, constants.ResetPasswordPurpose, now); err != nil {
			return err
		}
		return tx.RevokeUserSessions(ctx, entry.UserID, now)
	})
}

func (s *Service) UpdateProfile(ctx context.Context, userID uint64, displayName string) (usermodel.Profile, error) {
	name := strings.TrimSpace(displayName)
	if name == "" || utf8.RuneCountInString(name) > constants.DisplayNameMaxRunes {
		return usermodel.Profile{}, common.ErrInvalidInput
	}
	var user usermodel.User
	err := s.repository.WithinTransaction(ctx, func(tx repositoryinterface.UserRepository) error {
		if err := tx.SetDisplayName(ctx, userID, name); err != nil {
			return err
		}
		var err error
		user, err = tx.UserByID(ctx, userID)
		return err
	})
	if err != nil {
		return usermodel.Profile{}, err
	}
	return profile(user), nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return fmt.Errorf("%w: %v", common.ErrInvalidInput, err)
	}
	user, err := s.repository.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !cryptoutil.VerifyPassword(user.PasswordHash, oldPassword) {
		return common.ErrCurrentPassword
	}
	if oldPassword == newPassword {
		return common.ErrSamePassword
	}
	hash, err := cryptoutil.HashPassword(newPassword)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	return s.repository.WithinTransaction(ctx, func(tx repositoryinterface.UserRepository) error {
		updated, err := tx.SetPasswordHashIfMatch(ctx, userID, user.PasswordHash, hash)
		if err != nil {
			return err
		}
		if !updated {
			return common.ErrCurrentPassword
		}
		return tx.RevokeUserSessions(ctx, userID, now)
	})
}
