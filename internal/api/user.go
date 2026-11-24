package api

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/Jack11843121/token-proxy/internal/model"
)

const userCols = `id, username, password, display_name, role, status, quota, used_quota, request_count, note, created_at, updated_at`

func scanUser(s scanner) (*model.User, error) {
	var u model.User
	err := s.Scan(&u.ID, &u.Username, &u.Password, &u.DisplayName, &u.Role, &u.Status, &u.Quota, &u.UsedQuota, &u.RequestCount, &u.Note, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

type userBody struct {
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	DisplayName string  `json:"display_name"`
	Quota       float64 `json:"quota"`
	Status      *int    `json:"status"`
	Note        string  `json:"note"`
}

// ListUsers 分页列出用户。
func (h *Handler) ListUsers(c *gin.Context) {
	offset, limit, kw := pageParams(c)
	where := " WHERE username LIKE ? OR display_name LIKE ?"
	args := []interface{}{"%" + kw + "%", "%" + kw + "%"}
	var total int
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM users"+where, args...).Scan(&total); err != nil {
		fail(c, 500, err.Error())
		return
	}
	rows, err := h.DB.Query("SELECT "+userCols+" FROM users"+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, limit, offset)...)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	list := make([]*model.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			fail(c, 500, err.Error())
			return
		}
		list = append(list, u)
	}
	ok(c, gin.H{"list": list, "total": total})
}

// CreateUser 创建用户（role 默认 user；admin 由首个账号 seed 生成）。
func (h *Handler) CreateUser(c *gin.Context) {
	var b userBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Username == "" || b.Password == "" {
		fail(c, 400, "用户名和密码必填")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(b.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, 500, "密码加密失败")
		return
	}
	if b.DisplayName == "" {
		b.DisplayName = b.Username
	}
	status := statusOr(b.Status)
	now := time.Now().Unix()
	res, err := h.DB.Exec(`INSERT INTO users (username, password, display_name, role, status, quota, used_quota, request_count, note, created_at, updated_at)
		VALUES (?, ?, ?, 'user', ?, ?, 0, 0, ?, ?, ?)`,
		b.Username, string(hash), b.DisplayName, status, b.Quota, b.Note, now, now)
	if err != nil {
		fail(c, 400, "用户名已存在")
		return
	}
	id, _ := res.LastInsertId()
	ok(c, gin.H{"id": id})
}

// GetUser 返回单个用户。
func (h *Handler) GetUser(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	u, err := scanUser(h.DB.QueryRow("SELECT "+userCols+" FROM users WHERE id = ?", id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(c, 404, "用户不存在")
			return
		}
		fail(c, 500, err.Error())
		return
	}
	ok(c, u)
}

// UpdateUser 更新用户（display_name/quota/status/note；password 非空则重置）。
func (h *Handler) UpdateUser(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	var b userBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	now := time.Now().Unix()
	status := statusOr(b.Status)
	if b.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(b.Password), bcrypt.DefaultCost)
		if _, err := h.DB.Exec(`UPDATE users SET password=?, display_name=?, quota=?, status=?, note=?, updated_at=? WHERE id=?`,
			string(hash), b.DisplayName, b.Quota, status, b.Note, now, id); err != nil {
			fail(c, 500, err.Error())
			return
		}
	} else {
		if _, err := h.DB.Exec(`UPDATE users SET display_name=?, quota=?, status=?, note=?, updated_at=? WHERE id=?`,
			b.DisplayName, b.Quota, status, b.Note, now, id); err != nil {
			fail(c, 500, err.Error())
			return
		}
	}
	ok(c, gin.H{"message": "已更新"})
}

// DeleteUser 删除用户及其令牌。
func (h *Handler) DeleteUser(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	tx, err := h.DB.Begin()
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM tokens WHERE user_id = ?`, id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM users WHERE id = ?`, id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"message": "已删除"})
}

type quotaBody struct {
	Amount float64 `json:"amount" binding:"required"`
}

// AdjustQuota 给用户充值/扣减额度（正=加，负=减，结果不低于 0）。
func (h *Handler) AdjustQuota(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	var b quotaBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	var cur float64
	if err := h.DB.QueryRow(`SELECT quota FROM users WHERE id = ?`, id).Scan(&cur); err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	next := cur + b.Amount
	if next < 0 {
		next = 0
	}
	h.DB.Exec(`UPDATE users SET quota = ?, updated_at = ? WHERE id = ?`, next, time.Now().Unix(), id)
	ok(c, gin.H{"quota": next})
}