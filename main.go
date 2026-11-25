// token-proxy —— 开源多模型 Token 网关 / 大模型中转站。
// 单二进制部署：内置 SQLite，后台管理 REST API + OpenAI 兼容中继 + 嵌入的前端界面。
package main

import (
	"embed"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/api"
	"github.com/Jack11843121/token-proxy/internal/config"
	"github.com/Jack11843121/token-proxy/internal/middleware"
	"github.com/Jack11843121/token-proxy/internal/relay"
	"github.com/Jack11843121/token-proxy/internal/store"
)

//go:embed all:ui/dist
var uiFS embed.FS

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()
	if err := store.EnsureAdmin(db, cfg.AdminUser, cfg.AdminPass); err != nil {
		log.Fatalf("初始化管理员失败: %v", err)
	}
	if err := store.EnsureDefaultModels(db); err != nil {
		log.Fatalf("初始化默认模型失败: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "product": "token-proxy"})
	})

	api.Register(r, &api.Handler{DB: db, Cfg: cfg})
	relay.Register(r, relay.New(db))
	serveFrontend(r)

	log.Printf("token-proxy 管理后台: http://localhost%s  (默认 %s / %s)", cfg.Listen, cfg.AdminUser, cfg.AdminPass)
	log.Printf("token-proxy 中继接口: http://localhost%s/v1 (OpenAI 兼容，Authorization: Bearer sk-xxx)", cfg.Listen)
	if err := r.Run(cfg.Listen); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

// serveFrontend 托管嵌入的前端资源并做 SPA 回退。
func serveFrontend(r *gin.Engine) {
	sub, err := fs.Sub(uiFS, "ui/dist")
	if err != nil {
		return
	}
	assets, err := fs.Sub(sub, "assets")
	if err == nil {
		r.StaticFS("/assets", http.FS(assets))
	}
	index, _ := fs.ReadFile(sub, "index.html")

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/v1/") {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "not found", "type": "invalid_request_error"}})
			return
		}
		// 优先返回嵌入的静态资源（favicon 等），再回退 index.html。
		if p != "/" {
			if b, err := fs.ReadFile(sub, strings.TrimPrefix(p, "/")); err == nil {
				ct := mime.TypeByExtension(filepath.Ext(p))
				if ct == "" {
					ct = "application/octet-stream"
				}
				c.Data(http.StatusOK, ct, b)
				return
			}
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
}