package user_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	usermodel "inforadar/internal/api/model/user"
	"inforadar/internal/common"
	"inforadar/internal/config"
	"inforadar/internal/constants"
	repositoryinterface "inforadar/internal/repository/interface"
	userrepository "inforadar/internal/repository/user"
	userservice "inforadar/internal/service/user"
	cryptoutil "inforadar/internal/utils/crypto"
)

type captureMailer struct {
	reset string
	fail  bool
}

func (m *captureMailer) SendPasswordReset(_ string, token string) error {
	if m.fail {
		return errors.New("SMTP unavailable")
	}
	m.reset = token
	return nil
}

// Set INFORADAR_TEST_DATABASE_* to a disposable MySQL database with the manual SQL applied.
func openTestDatabase(t *testing.T) (repositoryinterface.UserStore, *gorm.DB) {
	t.Helper()
	dbName := os.Getenv("INFORADAR_TEST_DATABASE_DB")
	if dbName == "" {
		t.Skip("set INFORADAR_TEST_DATABASE_DB and connection fields to run the MySQL integration test")
	}
	port := constants.DefaultMySQLPort
	if raw := os.Getenv("INFORADAR_TEST_DATABASE_PORT"); raw != "" {
		var err error
		port, err = strconv.Atoi(raw)
		if err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Database{
		Host: os.Getenv("INFORADAR_TEST_DATABASE_HOST"), Port: port,
		Username: os.Getenv("INFORADAR_TEST_DATABASE_USERNAME"),
		Password: os.Getenv("INFORADAR_TEST_DATABASE_PASSWORD"), DB: dbName,
	}
	if cfg.Host == "" || cfg.Username == "" {
		t.Fatal("test database host and username are required")
	}
	repository, err := userrepository.OpenMySQL(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	driverConfig := drivermysql.NewConfig()
	driverConfig.User = cfg.Username
	driverConfig.Passwd = cfg.Password
	driverConfig.Net = constants.MySQLNetworkTCP
	driverConfig.Addr = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	driverConfig.DBName = cfg.DB
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	driverConfig.Params = map[string]string{"charset": constants.MySQLCharset}
	connector, err := drivermysql.NewConnector(driverConfig)
	if err != nil {
		t.Fatal(err)
	}
	inspectionDB := sql.OpenDB(connector)
	t.Cleanup(func() { _ = inspectionDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: inspectionDB, DSNConfig: driverConfig}), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return repository, db
}

func TestMySQLStoreOwnsConnection(t *testing.T) {
	store, _ := openTestDatabase(t)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UserByEmail(context.Background(), "nobody@example.test"); err == nil {
		t.Fatal("closed repository still accepts database queries")
	}
}

func TestMySQLAccountLifecycle(t *testing.T) {
	repository, db := openTestDatabase(t)
	mailer := &captureMailer{}
	service, err := userservice.NewService(repository, mailer, "integration-test-secret-at-least-32-bytes")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	email := fmt.Sprintf("account-test-%d@example.test", time.Now().UnixNano())
	t.Cleanup(func() { _ = db.Where("email = ?", email).Delete(&usermodel.User{}).Error })
	if err := service.Register(ctx, email, "first password 123"); err != nil {
		t.Fatal(err)
	}
	var row usermodel.User
	if err := db.Where("email = ?", email).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Account != email || row.PasswordHash == "" || row.PasswordHash == "first password 123" || row.Role != "user" || row.DisplayName == "" || row.CreatedAt.IsZero() || row.EditTime.IsZero() || row.UpdatedAt.IsZero() || row.IsDelete {
		t.Fatalf("user row does not match the account schema: %+v", row)
	}
	var registrationTokens int64
	if err := db.Model(&usermodel.ActionToken{}).Where("user_id = ?", row.ID).Count(&registrationTokens).Error; err != nil || registrationTokens != 0 {
		t.Fatalf("registration created action tokens: count=%d, error=%v", registrationTokens, err)
	}
	if err := service.Register(ctx, email, "first password 123"); !errors.Is(err, common.ErrDuplicateEmail) {
		t.Fatalf("duplicate registration = %v", err)
	}
	session, err := service.Login(ctx, email, "first password 123")
	if err != nil {
		t.Fatal(err)
	}
	profile, userID, err := service.Authenticate(ctx, session.AccessToken)
	if err != nil || profile.Email != email {
		t.Fatalf("authenticate = %v, profile = %+v", err, profile)
	}
	renewed, err := service.Refresh(ctx, session.Refresh)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.Refresh == session.Refresh {
		t.Fatal("refresh token was not rotated")
	}
	if _, err := service.Refresh(ctx, session.Refresh); !errors.Is(err, common.ErrNotLogin) {
		t.Fatalf("old refresh token = %v", err)
	}
	profile, err = service.UpdateProfile(ctx, userID, "Radar Observer")
	if err != nil || profile.DisplayName != "Radar Observer" {
		t.Fatalf("update profile = %v, profile = %+v", err, profile)
	}
	if err := service.RequestPasswordReset(ctx, "unknown@example.test"); err != nil || mailer.reset != "" {
		t.Fatalf("unknown account reset = %v", err)
	}
	if err := service.RequestPasswordReset(ctx, email); err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(ctx, mailer.reset, "second password 456"); err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(ctx, mailer.reset, "another password"); !errors.Is(err, common.ErrInvalidToken) {
		t.Fatalf("reused reset token = %v", err)
	}
	if _, _, err := service.Authenticate(ctx, renewed.AccessToken); !errors.Is(err, common.ErrNotLogin) {
		t.Fatalf("access token after reset = %v", err)
	}
	if _, err := service.Refresh(ctx, renewed.Refresh); !errors.Is(err, common.ErrNotLogin) {
		t.Fatalf("refresh token after reset = %v", err)
	}
	if _, err := service.Login(ctx, email, "first password 123"); !errors.Is(err, common.ErrInvalidCredentials) {
		t.Fatalf("old password login = %v", err)
	}
	session, err = service.Login(ctx, email, "second password 456")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(ctx, userID, "wrong password", "third password 789"); !errors.Is(err, common.ErrCurrentPassword) {
		t.Fatalf("wrong current password = %v", err)
	}
	if err := service.ChangePassword(ctx, userID, "second password 456", "third password 789"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Authenticate(ctx, session.AccessToken); !errors.Is(err, common.ErrNotLogin) {
		t.Fatalf("access token after password change = %v", err)
	}
	session, err = service.Login(ctx, email, "third password 789")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Logout(ctx, session.Refresh); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Authenticate(ctx, session.AccessToken); !errors.Is(err, common.ErrNotLogin) {
		t.Fatalf("access token after logout = %v", err)
	}
}

func TestMySQLExpiredResetAndFailedEmail(t *testing.T) {
	repository, db := openTestDatabase(t)
	mailer := &captureMailer{fail: true}
	service, err := userservice.NewService(repository, mailer, "integration-test-secret-at-least-32-bytes")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	email := fmt.Sprintf("mail-test-%d@example.test", time.Now().UnixNano())
	t.Cleanup(func() { _ = db.Where("email = ?", email).Delete(&usermodel.User{}).Error })
	if err := service.Register(ctx, email, "first password 123"); err != nil {
		t.Fatalf("registration should not use SMTP: %v", err)
	}
	if err := service.RequestPasswordReset(ctx, email); !errors.Is(err, common.ErrMailUnavailable) {
		t.Fatalf("SMTP failure during password reset = %v", err)
	}
	mailer.fail = false
	if err := service.RequestPasswordReset(ctx, email); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&usermodel.ActionToken{}).Where("token_hash = ?", cryptoutil.TokenHash(mailer.reset)).Update("expires_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(ctx, mailer.reset, "second password 456"); !errors.Is(err, common.ErrInvalidToken) {
		t.Fatalf("expired reset token = %v", err)
	}
}
