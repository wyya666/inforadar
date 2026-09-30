package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
	"inforadar/internal/constants"
)

type Config struct {
	Server   Server   `mapstructure:"server"`
	Database Database `mapstructure:"database"`
	Auth     Auth     `mapstructure:"auth"`
	SMTP     SMTP     `mapstructure:"smtp"`
	Radar    Radar    `mapstructure:"radar"`
}

type Server struct {
	Address        string   `mapstructure:"address"`
	FrontendURL    string   `mapstructure:"frontend_url"`
	Environment    string   `mapstructure:"environment"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

type Database struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

type Auth struct {
	JWTSecret    string `mapstructure:"jwt_secret"`
	CookieSecure bool   `mapstructure:"cookie_secure"`
}

type SMTP struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	TLSMode  string `mapstructure:"tls_mode"`
}

type Radar struct {
	EncryptionKey string `mapstructure:"encryption_key"`
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName(constants.ConfigName)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	if path := os.Getenv("INFORADAR_CONFIG"); path != "" {
		v.SetConfigFile(path)
	}
	v.SetEnvPrefix(constants.ConfigEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	defaults := map[string]any{
		"server.address":      constants.DefaultServerAddress,
		"server.frontend_url": constants.DefaultFrontendURL,
		"server.environment":  constants.EnvironmentDevelopment,
		"database.port":       constants.DefaultMySQLPort,
		"auth.cookie_secure":  false,
		"smtp.port":           constants.DefaultSMTPPort,
		"smtp.tls_mode":       constants.SMTPTLSStartTLS,
	}
	for key, value := range defaults {
		v.SetDefault(key, value)
	}
	for _, key := range []string{
		"server.address", "server.frontend_url", "server.environment", "server.trusted_proxies",
		"database.host", "database.port", "database.username", "database.password", "database.db",
		"auth.jwt_secret", "auth.cookie_secure",
		"smtp.host", "smtp.port", "smtp.username", "smtp.password", "smtp.from", "smtp.tls_mode",
		"radar.encryption_key",
	} {
		if err := v.BindEnv(key); err != nil {
			return Config{}, err
		}
	}
	if err := v.ReadInConfig(); err != nil {
		var missing viper.ConfigFileNotFoundError
		if !errors.As(err, &missing) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Database.Host == "" || c.Database.Port < 1 || c.Database.Port > 65535 || c.Database.Username == "" || c.Database.DB == "" {
		return errors.New("database.host, database.username, database.db and a valid database.port are required")
	}
	if len(c.Auth.JWTSecret) < constants.JWTSecretMinBytes {
		return errors.New("auth.jwt_secret must be at least 32 bytes")
	}
	if c.Server.Environment == constants.EnvironmentProduction && (c.Auth.JWTSecret == constants.LegacyExampleJWTSecret || c.Auth.JWTSecret == constants.DevelopmentExampleJWTSecret) {
		return errors.New("production requires a private auth.jwt_secret; example values are public")
	}
	if len(c.Radar.EncryptionKey) != 64 {
		return errors.New("radar.encryption_key must be 64 hexadecimal characters; generate with openssl rand -hex 32")
	}
	for _, char := range c.Radar.EncryptionKey {
		if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
			return errors.New("radar.encryption_key must be hexadecimal")
		}
	}
	if c.Server.Environment == constants.EnvironmentProduction && c.Radar.EncryptionKey == strings.Repeat("0", 64) {
		return errors.New("production requires a private radar.encryption_key")
	}
	if c.Server.Address == "" {
		return errors.New("server.address is required")
	}
	u, err := url.Parse(c.Server.FrontendURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("server.frontend_url must be an http(s) origin")
	}
	if u.Path != "" && u.Path != "/" {
		return errors.New("server.frontend_url must not contain a path")
	}
	if c.Server.Environment != constants.EnvironmentDevelopment && c.Server.Environment != constants.EnvironmentProduction {
		return errors.New("server.environment must be development or production")
	}
	if c.Server.Environment == constants.EnvironmentProduction && (u.Scheme != "https" || !c.Auth.CookieSecure) {
		return errors.New("production requires an https frontend_url and a secure cookie")
	}
	if c.SMTP.Host == "" || c.SMTP.From == "" || c.SMTP.Port < 1 || c.SMTP.Port > 65535 {
		return errors.New("smtp.host, smtp.from and a valid smtp.port are required")
	}
	if c.SMTP.TLSMode != constants.SMTPTLSStartTLS && c.SMTP.TLSMode != constants.SMTPTLSImplicit && c.SMTP.TLSMode != constants.SMTPTLSNone {
		return errors.New("smtp.tls_mode must be starttls, implicit or none")
	}
	if c.Server.Environment == constants.EnvironmentProduction && c.SMTP.TLSMode == constants.SMTPTLSNone {
		return errors.New("production SMTP requires TLS")
	}
	if (c.SMTP.Username == "") != (c.SMTP.Password == "") {
		return errors.New("smtp.username and smtp.password must be set together")
	}
	return nil
}

func (c Config) FrontendOrigin() string {
	u, _ := url.Parse(c.Server.FrontendURL)
	return u.Scheme + "://" + u.Host
}
