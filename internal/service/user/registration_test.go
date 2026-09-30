package user_test

import (
	"context"
	"errors"
	"testing"

	usermodel "inforadar/internal/api/model/user"
	"inforadar/internal/common"
	repositoryinterface "inforadar/internal/repository/interface"
	userservice "inforadar/internal/service/user"
)

type registrationRepository struct {
	repositoryinterface.UserRepository
	user *usermodel.User
}

func (r *registrationRepository) CreateUser(_ context.Context, user *usermodel.User) error {
	if r.user != nil {
		return common.ErrDuplicateEmail
	}
	user.ID = 42
	copy := *user
	r.user = &copy
	return nil
}

func (r *registrationRepository) UserByEmail(_ context.Context, email string) (usermodel.User, error) {
	if r.user == nil || r.user.Email != email {
		return usermodel.User{}, common.ErrNotFound
	}
	return *r.user, nil
}

func (r *registrationRepository) CreateSession(_ context.Context, _ *usermodel.Session) error {
	return nil
}

type noMailOnRegistration struct{}

func (noMailOnRegistration) SendPasswordReset(string, string) error {
	panic("registration must not send email")
}

func TestRegisterWithoutEmailVerification(t *testing.T) {
	repository := &registrationRepository{}
	service, err := userservice.NewService(repository, noMailOnRegistration{}, "unit-test-secret-at-least-32-bytes")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := service.Register(ctx, "not-an-email", "password 123"); !errors.Is(err, common.ErrInvalidInput) {
		t.Fatalf("invalid email = %v", err)
	}
	if repository.user != nil {
		t.Fatal("invalid email created an account")
	}
	if err := service.Register(ctx, " User@Example.Test ", "password 123"); err != nil {
		t.Fatal(err)
	}
	if repository.user.Email != "user@example.test" || repository.user.PasswordHash == "password 123" {
		t.Fatalf("registered account was not normalized and hashed: %+v", repository.user)
	}
	if err := service.Register(ctx, "user@example.test", "password 123"); !errors.Is(err, common.ErrDuplicateEmail) {
		t.Fatalf("duplicate email = %v", err)
	}
	result, err := service.Login(ctx, "user@example.test", "password 123")
	if err != nil || result.AccessToken == "" {
		t.Fatalf("login immediately after registration = %v", err)
	}
}
