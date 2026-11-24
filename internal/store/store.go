// Package store 负责 SQLite 连接、建表迁移与初始数据。
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，无需 CGO

	"golang.org/x/crypto/bcrypt"
)

// Open 打开（必要时创建）SQLite 数据库并执行迁移，返回连接池。
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite 单写者；用单连接串行化写入，规避并发锁冲突（转发的长流不持有连接，不影响吞吐）。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'user',
			status INTEGER NOT NULL DEFAULT 1,
			quota REAL NOT NULL DEFAULT 0,
			used_quota REAL NOT NULL DEFAULT 0,
			request_count INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			key TEXT NOT NULL UNIQUE,
			status INTEGER NOT NULL DEFAULT 1,
			expired_time INTEGER NOT NULL DEFAULT 0,
			unlimited_quota INTEGER NOT NULL DEFAULT 0,
			quota REAL NOT NULL DEFAULT -1,
			used_quota REAL NOT NULL DEFAULT 0,
			model_scope TEXT NOT NULL DEFAULT '',
			request_count INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0,
			accessed_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'openai',
			base_url TEXT NOT NULL DEFAULT '',
			key TEXT NOT NULL DEFAULT '',
			models TEXT NOT NULL DEFAULT '',
			model_mapping TEXT NOT NULL DEFAULT '',
			priority INTEGER NOT NULL DEFAULT 0,
			weight INTEGER NOT NULL DEFAULT 1,
			status INTEGER NOT NULL DEFAULT 1,
			groups TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			price_input REAL NOT NULL DEFAULT 0,
			price_output REAL NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1,
			tags TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			token_id INTEGER NOT NULL DEFAULT 0,
			channel_id INTEGER NOT NULL DEFAULT 0,
			token_name TEXT NOT NULL DEFAULT '',
			channel_name TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			upstream_model TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT '',
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			quota REAL NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1,
			error TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			latency_ms INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_created ON logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_user ON logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_token ON logs(token_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_user ON tokens(user_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tokens_key ON tokens(key)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// EnsureAdmin 保证至少存在一个管理员；不存在则用 username/password 创建。
// 仅当 users 表中还没有任何 admin 时生效，避免覆盖用户已改过的密码。
func EnsureAdmin(db *sql.DB, username, password string) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = db.Exec(`INSERT INTO users (username, password, display_name, role, status, quota, used_quota, request_count, created_at, updated_at)
		VALUES (?, ?, '管理员', 'admin', 1, 100.0, 0, 0, ?, ?)`, username, string(hash), now, now)
	return err
}

// EnsureDefaultModels 预置常用模型价格（仅当 models 表为空时），单位 = 额度 / 100 万 tokens。
// 这是「美元等价」的参考价，管理员可在后台自行调整。
func EnsureDefaultModels(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM models`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now().Unix()
	seeds := []struct {
		name                    string
		in, out                 float64
		tags                    string
	}{
		{"gpt-4o", 2.5, 10, "OpenAI"},
		{"gpt-4o-mini", 0.15, 0.6, "OpenAI"},
		{"o3-mini", 1.1, 4.4, "OpenAI"},
		{"claude-sonnet-4", 3, 15, "Anthropic"},
		{"claude-haiku-4", 0.8, 4, "Anthropic"},
		{"deepseek-chat", 0.27, 1.1, "DeepSeek"},
		{"deepseek-reasoner", 0.55, 2.19, "DeepSeek"},
		{"qwen-plus", 0.4, 1.2, "通义千问"},
		{"glm-4-plus", 0.5, 1.5, "智谱"},
		{"moonshot-v1-8k", 0.6, 1.2, "Kimi"},
		{"doubao-1.5-pro-32k", 0.8, 2, "豆包"},
		{"gemini-2.0-flash", 0.1, 0.4, "Google"},
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO models (name, price_input, price_output, status, tags, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, s := range seeds {
		if _, err := stmt.Exec(s.name, s.in, s.out, s.tags, now, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// TokenKey 生成一个 sk- 开头的随机令牌密钥。
func TokenKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-" + hex.EncodeToString(b), nil
}