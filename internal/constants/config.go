package constants

import "time"

const (
	ConfigName             = "config"
	ConfigEnvPrefix        = "INFORADAR"
	DefaultServerAddress   = ":8080"
	DefaultFrontendURL     = "http://127.0.0.1:5173"
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
	DefaultSMTPPort        = 587
	DefaultMySQLPort       = 3306
	MySQLNetworkTCP        = "tcp"
	MySQLCharset           = "utf8mb4"
	MySQLMaxOpenConns      = 20
	MySQLMaxIdleConns      = 10
	MySQLConnMaxLifetime   = time.Hour
	MySQLPingTimeout       = 5 * time.Second
	SMTPTLSStartTLS        = "starttls"
	SMTPTLSImplicit        = "implicit"
	SMTPTLSNone            = "none"
	JWTSecretMinBytes      = 32
	MaxAPIRequestBodyBytes = 1 << 20
	// The first example config contained this public value. Never accept it in production.
	LegacyExampleJWTSecret      = "f39f9110e6ebf7e04784b3b44bdfe6beed3b661eba31a989837d792cc08de52e"
	DevelopmentExampleJWTSecret = "inforadar-local-jwt-secret-do-not-use-in-production"
)
