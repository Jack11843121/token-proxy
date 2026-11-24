// Package api 实现后台管理端 REST API。
package api

import (
	"database/sql"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/config"
	"github.com/Jack11843121/token-proxy/internal/middleware"
)

// statusOr 解析状态字段：nil（未传）默认启用，显式 0 = 禁用。
func statusOr(p *int) int {
	if p == nil {
		return 1 // 启用
	}
	return *p
}

// Handler 持有管理 API 所需的依赖。
type Handler struct {
	DB  *sql.DB
	Cfg *config.Config
}

// Register 挂载后台管理路由。
func Register(r *gin.Engine, h *Handler) {
	// 登录接口（无需鉴权）
	auth := r.Group("/api/auth")
	auth.POST("/login", h.Login)
	auth.POST("/password", middleware.AdminAuth(h.Cfg.JWTSecret), h.ChangePassword)
	auth.GET("/me", middleware.AdminAuth(h.Cfg.JWTSecret), h.Me)

	// 管理接口（需后台鉴权）
	adm := r.Group("/api")
	adm.Use(middleware.AdminAuth(h.Cfg.JWTSecret))
	adm.GET("/users", h.ListUsers)
	adm.POST("/users", h.CreateUser)
	adm.GET("/users/:id", h.GetUser)
	adm.PUT("/users/:id", h.UpdateUser)
	adm.DELETE("/users/:id", h.DeleteUser)
	adm.POST("/users/:id/quota", h.AdjustQuota)

	adm.GET("/tokens", h.ListTokens)
	adm.POST("/tokens", h.CreateToken)
	adm.PUT("/tokens/:id", h.UpdateToken)
	adm.DELETE("/tokens/:id", h.DeleteToken)

	adm.GET("/channels", h.ListChannels)
	adm.POST("/channels", h.CreateChannel)
	adm.PUT("/channels/:id", h.UpdateChannel)
	adm.DELETE("/channels/:id", h.DeleteChannel)
	adm.POST("/channels/:id/test", h.TestChannel)

	adm.GET("/models", h.ListModels)
	adm.POST("/models", h.CreateModel)
	adm.PUT("/models/:id", h.UpdateModel)
	adm.DELETE("/models/:id", h.DeleteModel)

	adm.GET("/logs", h.ListLogs)
	adm.GET("/dashboard", h.Dashboard)
}

// ---- 公共响应与辅助 ----

// scanner 抽象 *sql.Row 与 *sql.Rows，供各资源的 scan 函数复用。
type scanner interface {
	Scan(dest ...interface{}) error
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"ok": true, "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"ok": false, "error": msg})
}

func paramID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, 400, "非法的 ID")
		return 0, false
	}
	return id, true
}

func pageParams(c *gin.Context) (offset, limit int, keyword string) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	return (page - 1) * size, size, c.Query("keyword")
}