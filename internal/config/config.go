// Package config 加载 token-proxy 运行时配置（环境变量 + 默认值）。
package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config 描述网关的运行时配置。
type Config struct {
	Listen         string // HTTP 监听地址
	DataDir        string // 数据目录
	DBPath         string // SQLite 数据库文件路径
	AdminUser      string // 初始管理员用户名（首次启动 seed）
	AdminPass      string // 初始管理员密码（首次启动 seed）
	JWTSecret      string // 后台登录 JWT 签名密钥
	SessionSeconds int    // 后台登录会话有效期（秒）
}

// Load 从环境变量读取配置，未设置时使用默认值。
func Load() *Config {
	c := &Config{
		Listen:         getenv("TOKEN_PROXY_LISTEN", ":3000"),
		DataDir:        getenv("TOKEN_PROXY_DATA_DIR", "./data"),
		AdminUser:      getenv("TOKEN_PROXY_ADMIN_USER", "root"),
		AdminPass:      getenv("TOKEN_PROXY_ADMIN_PASS", "123456"),
		JWTSecret:      getenv("TOKEN_PROXY_JWT_SECRET", "token-proxy-change-me-secret"),
		SessionSeconds: getenvInt("TOKEN_PROXY_SESSION_SECONDS", 7*24*3600),
	}
	if c.DBPath = getenv("TOKEN_PROXY_DB", ""); c.DBPath == "" {
		c.DBPath = filepath.Join(c.DataDir, "token-proxy.db")
	}
	return c
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}