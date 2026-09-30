package config

import (
	"inforadar/internal/constants"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionRejectsExampleKeys(t *testing.T) {
	cfg := Config{
		Server:   Server{Address: ":8080", FrontendURL: "https://radar.example.test", Environment: constants.EnvironmentProduction},
		Database: Database{Host: "127.0.0.1", Port: 3306, Username: "user", DB: "inforadar"},
		Auth:     Auth{JWTSecret: constants.LegacyExampleJWTSecret, CookieSecure: true},
		SMTP:     SMTP{Host: "smtp.example.test", Port: 465, From: "no-reply@example.test", TLSMode: constants.SMTPTLSImplicit},
		Radar:    Radar{EncryptionKey: strings.Repeat("a", 64)},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "example values") {
		t.Fatalf("production accepted public example key: %v", err)
	}
	cfg.Auth.JWTSecret = constants.DevelopmentExampleJWTSecret
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "example values") {
		t.Fatalf("production accepted development example key: %v", err)
	}
	cfg.Auth.JWTSecret = strings.Repeat("b", constants.JWTSecretMinBytes)
	cfg.Radar.EncryptionKey = strings.Repeat("0", 64)
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "private radar.encryption_key") {
		t.Fatalf("production accepted public radar key: %v", err)
	}
	cfg.Radar.EncryptionKey = strings.Repeat("a", 64)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("production rejected replacement JWT secret: %v", err)
	}
}

func TestExampleConfigHasDevelopmentKeys(t *testing.T) {
	path, err := filepath.Abs("../../config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("INFORADAR_CONFIG", path)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("example config should start in development: %v", err)
	}
	if cfg.Auth.JWTSecret != constants.DevelopmentExampleJWTSecret || cfg.Radar.EncryptionKey != strings.Repeat("0", 64) {
		t.Fatal("example config no longer uses the documented development keys")
	}
}

func TestLoadDatabaseFieldsAndEnvironmentOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := `server:
  address: ":8080"
  frontend_url: "http://127.0.0.1:5173"
database:
  host: "127.0.0.1"
  port: 3306
  username: "file-user"
  password: "file-password"
  db: "inforadar"
auth:
  jwt_secret: "01234567890123456789012345678901"
smtp:
  host: "127.0.0.1"
  port: 1025
  from: "no-reply@example.test"
  tls_mode: none
radar:
  encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("INFORADAR_CONFIG", path)
	t.Setenv("INFORADAR_DATABASE_PORT", "3307")
	t.Setenv("INFORADAR_DATABASE_USERNAME", "env-user")
	t.Setenv("INFORADAR_DATABASE_PASSWORD", "env:p@ss/word")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Host != "127.0.0.1" || cfg.Database.Port != 3307 || cfg.Database.Username != "env-user" || cfg.Database.Password != "env:p@ss/word" || cfg.Database.DB != "inforadar" {
		t.Fatalf("unexpected database configuration: host=%q port=%d username=%q db=%q", cfg.Database.Host, cfg.Database.Port, cfg.Database.Username, cfg.Database.DB)
	}

	t.Setenv("INFORADAR_DATABASE_DB", "env-inforadar")
	cfg, err = Load()
	if err != nil || cfg.Database.DB != "env-inforadar" {
		t.Fatalf("database.db environment override = %q, error = %v", cfg.Database.DB, err)
	}
	cfg.Database.DB = ""
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "database.db") {
		t.Fatalf("missing database db should fail validation, got %v", err)
	}
}
