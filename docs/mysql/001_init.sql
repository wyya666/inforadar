-- Execute this file manually against MySQL 8.0+ before starting the API.
-- The application never creates or alters tables.
-- This script targets a fresh database; CREATE TABLE does not change an older schema.
CREATE DATABASE IF NOT EXISTS inforadar CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE inforadar;

CREATE TABLE IF NOT EXISTS `user` (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_account VARCHAR(256) NOT NULL COMMENT '账号，当前使用邮箱',
  email VARCHAR(254) NOT NULL,
  user_password VARCHAR(512) NOT NULL COMMENT 'Argon2id 密码哈希',
  user_name VARCHAR(256) NULL COMMENT '用户昵称',
  user_avatar VARCHAR(1024) NULL COMMENT '用户头像',
  user_profile VARCHAR(512) NULL COMMENT '用户简介',
  user_role VARCHAR(256) NOT NULL DEFAULT 'user' COMMENT '用户角色：user/admin',
  edit_time DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  create_time DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  update_time DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  is_delete TINYINT NOT NULL DEFAULT 0 COMMENT '是否删除',
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_account (user_account),
  UNIQUE KEY uk_user_email (email),
  KEY idx_user_name (user_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户';

CREATE TABLE IF NOT EXISTS auth_sessions (
  id CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  refresh_token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  revoked_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_auth_sessions_refresh_token_hash (refresh_token_hash),
  KEY idx_auth_sessions_user_id (user_id),
  CONSTRAINT fk_auth_sessions_user FOREIGN KEY (user_id) REFERENCES `user` (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS action_tokens (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  purpose VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  consumed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_action_tokens_token_hash (token_hash),
  KEY idx_action_tokens_user_purpose (user_id, purpose),
  CONSTRAINT fk_action_tokens_user FOREIGN KEY (user_id) REFERENCES `user` (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
