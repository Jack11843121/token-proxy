package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/Jack11843121/token-proxy/internal/auth"
	"github.com/Jack11843121/token-proxy/internal/model"
)

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 后台登录，成功返回 JWT 与用户信息。
func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误：需要 username 和 password")
		return
	}
	var u model.User
	err := h.DB.QueryRow(`SELECT id, username, password, display_name, role, status, quota, used_quota, request_count, note, created_at, updated_at
		FROM users WHERE username = ?`, req.Username).
		Scan(&u.ID, &u.Username, &u.Password, &u.DisplayName, &u.Role, &u.Status, &u.Quota, &u.UsedQuota, &u.RequestCount, &u.Note, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		fail(c, 401, "用户名或密码错误")
		return
	}
	if u.Status != model.StatusEnabled {
		fail(c, 403, "账号已被禁用")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		fail(c, 401, "用户名或密码错误")
		return
	}
	token, err := auth.Generate(u.ID, u.Role, h.Cfg.JWTSecret, h.Cfg.SessionSeconds)
	if err != nil {
		fail(c, 500, "生成登录令牌失败")
		return
	}
	ok(c, gin.H{"token": token, "user": u})
}

// Me 返回当前登录用户。
func (h *Handler) Me(c *gin.Context) {
	id := c.GetInt64("user_id")
	var u model.User
	err := h.DB.QueryRow(`SELECT id, username, password, display_name, role, status, quota, used_quota, request_count, note, created_at, updated_at
		FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.Password, &u.DisplayName, &u.Role, &u.Status, &u.Quota, &u.UsedQuota, &u.RequestCount, &u.Note, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	ok(c, u)
}

type passwordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改当前管理员密码。
func (h *Handler) ChangePassword(c *gin.Context) {
	var req passwordReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.NewPassword) < 6 {
		fail(c, 400, "新密码至少 6 位")
		return
	}
	id := c.GetInt64("user_id")
	var old string
	if err := h.DB.QueryRow(`SELECT password FROM users WHERE id = ?`, id).Scan(&old); err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(old), []byte(req.OldPassword)) != nil {
		fail(c, 400, "原密码错误")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	h.DB.Exec(`UPDATE users SET password = ?, updated_at = ? WHERE id = ?`, string(hash), time.Now().Unix(), id)
	ok(c, gin.H{"message": "密码已修改"})
}