package user

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"inforadar/internal/config"
	"inforadar/internal/constants"
	repositoryinterface "inforadar/internal/repository/interface"
	radarr "inforadar/internal/repository/radar"
)

// OpenMySQL connects to an existing schema. It never creates or alters tables.
func OpenMySQL(ctx context.Context, cfg config.Database) (repositoryinterface.UserStore, error) {
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
		return nil, fmt.Errorf("configure MySQL connection: %w", err)
	}
	sqlDB := sql.OpenDB(connector)
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: sqlDB, DSNConfig: driverConfig}), &gorm.Config{
		TranslateError: true,
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("connect to MySQL: %w", err)
	}
	sqlDB.SetMaxOpenConns(constants.MySQLMaxOpenConns)
	sqlDB.SetMaxIdleConns(constants.MySQLMaxIdleConns)
	sqlDB.SetConnMaxLifetime(constants.MySQLConnMaxLifetime)
	checkCtx, cancel := context.WithTimeout(ctx, constants.MySQLPingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(checkCtx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	for _, table := range []struct{ name, probe string }{
		{constants.TableUser, "SELECT id, user_account, email, user_password, user_name, user_avatar, user_profile, user_role, edit_time, create_time, update_time, is_delete FROM `user` LIMIT 0"},
		{constants.TableSessions, "SELECT id, user_id, refresh_token_hash, expires_at, revoked_at FROM auth_sessions LIMIT 0"},
		{constants.TableActionTokens, "SELECT id, user_id, purpose, token_hash, expires_at, consumed_at FROM action_tokens LIMIT 0"},
		{constants.TableRadarSettings, "SELECT user_id, model_provider, model_id, search_provider, interval_hours FROM radar_settings LIMIT 0"},
		{constants.TableProviderCredentials, "SELECT user_id, provider, ciphertext, nonce FROM provider_credentials LIMIT 0"},
		{constants.TableMonitorTopics, "SELECT id, user_id, next_run_at, lease_until FROM monitor_topics LIMIT 0"},
		{constants.TableMonitorRuns, "SELECT id, topic_id, status FROM monitor_runs LIMIT 0"},
		{constants.TableMonitorCandidates, "SELECT id, topic_id, url_hash FROM monitor_candidates LIMIT 0"},
	} {
		rows, err := sqlDB.QueryContext(checkCtx, table.probe)
		if err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("MySQL table %s is missing or outdated; execute docs/mysql/001_init.sql and docs/mysql/002_radar.sql manually: %w", table.name, err)
		}
		rows.Close()
	}
	return &Store{Repository: &Repository{db: db}, radar: radarr.New(db), pool: sqlDB}, nil
}

// Store is the root repository and owns the SQL connection pool.
type Store struct {
	*Repository
	radar repositoryinterface.RadarRepository
	pool  *sql.DB
}

func (s *Store) Close() error                               { return s.pool.Close() }
func (s *Store) Radar() repositoryinterface.RadarRepository { return s.radar }
