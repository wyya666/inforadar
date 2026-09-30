-- Run manually after 001_init.sql against the same MySQL 8.0+ database.
-- The application never migrates or creates these tables.
USE inforadar;
CREATE TABLE IF NOT EXISTS radar_settings (
  user_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  model_provider VARCHAR(16) NOT NULL,
  model_id VARCHAR(128) NOT NULL,
  search_provider VARCHAR(16) NOT NULL,
  interval_hours TINYINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  CONSTRAINT fk_radar_settings_user FOREIGN KEY (user_id) REFERENCES `user` (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS provider_credentials (
  user_id BIGINT UNSIGNED NOT NULL,
  provider VARCHAR(16) NOT NULL,
  ciphertext VARBINARY(4096) NOT NULL,
  nonce VARBINARY(12) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  PRIMARY KEY (user_id, provider),
  CONSTRAINT fk_provider_credentials_user FOREIGN KEY (user_id) REFERENCES `user` (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS monitor_topics (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  description VARCHAR(1000) NOT NULL,
  search_query VARCHAR(70) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  next_run_at DATETIME(3) NULL,
  last_run_at DATETIME(3) NULL,
  last_manual_at DATETIME(3) NULL,
  lease_until DATETIME(3) NULL,
  retry_count TINYINT UNSIGNED NOT NULL DEFAULT 0,
  last_error VARCHAR(256) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  KEY idx_monitor_topics_user (user_id, id),
  KEY idx_monitor_topics_due (status, next_run_at, lease_until),
  CONSTRAINT fk_monitor_topics_user FOREIGN KEY (user_id) REFERENCES `user` (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS monitor_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  topic_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(16) NOT NULL,
  result_count INT UNSIGNED NOT NULL DEFAULT 0,
  error_message VARCHAR(256) NULL,
  started_at DATETIME(3) NOT NULL,
  finished_at DATETIME(3) NULL,
  KEY idx_monitor_runs_user_topic (user_id, topic_id, id),
  CONSTRAINT fk_monitor_runs_user FOREIGN KEY (user_id) REFERENCES `user` (id) ON DELETE CASCADE,
  CONSTRAINT fk_monitor_runs_topic FOREIGN KEY (topic_id) REFERENCES monitor_topics (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS monitor_candidates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  topic_id BIGINT UNSIGNED NOT NULL,
  url_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  url VARCHAR(2048) NOT NULL,
  title VARCHAR(512) NOT NULL,
  snippet TEXT NULL,
  published_at DATETIME(3) NULL,
  important TINYINT(1) NOT NULL,
  summary VARCHAR(1000) NULL,
  first_seen_at DATETIME(3) NOT NULL,
  UNIQUE KEY uq_monitor_candidates_topic_url (topic_id, url_hash),
  KEY idx_monitor_candidates_feed (user_id, important, id),
  CONSTRAINT fk_monitor_candidates_user FOREIGN KEY (user_id) REFERENCES `user` (id) ON DELETE CASCADE,
  CONSTRAINT fk_monitor_candidates_topic FOREIGN KEY (topic_id) REFERENCES monitor_topics (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
