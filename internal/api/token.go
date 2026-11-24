package api

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/model"
	"github.com/Jack11843121/token-proxy/internal/store"
)

const tokenCols = `id, user_id, name, key, status, expired_time, unlimited_quota, quota, used_quota, model_scope, request_count, created_at, updated_at, accessed_at`

func scanToken(s scanner) (*model.Token, error) {
	var t model.Token
	err := s.Scan(&t.ID, &t.UserID, &t.Name, &t.Key, &t.Status, &t.ExpiredTime, &t.UnlimitedQuota, &t.Quota, &t.UsedQuota, &t.ModelScope, &t.RequestCount, &t.CreatedAt, &t.UpdatedAt, &t.AccessedAt)
	return &t, err
}

// ListTokens 列出令牌，可按用户过滤。
func (h *Handler) ListTokens(c *gin.Context) {
	offset, limit, kw := pageParams(c)
	where := " WHERE (name LIKE ?)"
	args := []interface{}{"%" + kw + "%"}
	if uid := c.Query("user_id"); uid != "" {
		where += " AND user_id = ?"
		args = append(args, uid)
	}
	var total int
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM tokens"+where, args...).Scan(&total); err != nil {
		fail(c, 500, err.Error())
		return
	}
	rows, err := h.DB.Query("SELECT "+tokenCols+" FROM tokens"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	list := make([]*model.Token, 0)
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			fail(c, 500, err.Error())
			return
		}
		list = append(list, t)
	}
	ok(c, gin.H{"list": list, "total": total})
}

type tokenBody struct {
	UserID         int64   `json:"user_id" binding:"required"`
	Name           string  `json:"name"`
	Status         *int    `json:"status"`
	ExpiredTime    int64   `json:"expired_time"`
	UnlimitedQuota int     `json:"unlimited_quota"`
	Quota          float64 `json:"quota"`
	ModelScope     string  `json:"model_scope"`
}

// CreateToken 创建令牌；Quota 传 -1 表示跟随用户额度。
func (h *Handler) CreateToken(c *gin.Context) {
	var b tokenBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误：需要 user_id")
		return
	}
	status := statusOr(b.Status)
	if b.Quota == 0 {
		b.Quota = -1 // 跟随用户额度
	}
	key, err := store.TokenKey()
	if err != nil {
		fail(c, 500, "生成密钥失败")
		return
	}
	now := time.Now().Unix()
	res, err := h.DB.Exec(`INSERT INTO tokens (user_id, name, key, status, expired_time, unlimited_quota, quota, used_quota, model_scope, request_count, created_at, updated_at, accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, 0, ?, ?, 0)`,
		b.UserID, b.Name, key, status, b.ExpiredTime, b.UnlimitedQuota, b.Quota, b.ModelScope, now, now)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	ok(c, gin.H{"id": id, "key": key})
}

// UpdateToken 更新令牌（name/status/expired_time/unlimited_quota/quota/model_scope）。
func (h *Handler) UpdateToken(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	var b tokenBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if _, err := h.DB.Exec(`UPDATE tokens SET name=?, status=?, expired_time=?, unlimited_quota=?, quota=?, model_scope=?, updated_at=? WHERE id=?`,
		b.Name, statusOr(b.Status), b.ExpiredTime, b.UnlimitedQuota, b.Quota, b.ModelScope, time.Now().Unix(), id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"message": "已更新"})
}

// DeleteToken 删除令牌。
func (h *Handler) DeleteToken(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	if _, err := h.DB.Exec(`DELETE FROM tokens WHERE id = ?`, id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"message": "已删除"})
}

// GetTokenForRelay 供中继层使用：按 key 查询令牌并校验状态。
func (h *Handler) GetTokenForRelay(key string) (*model.Token, error) {
	var t model.Token
	err := h.DB.QueryRow("SELECT "+tokenCols+" FROM tokens WHERE key = ?", key).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Key, &t.Status, &t.ExpiredTime, &t.UnlimitedQuota, &t.Quota, &t.UsedQuota, &t.ModelScope, &t.RequestCount, &t.CreatedAt, &t.UpdatedAt, &t.AccessedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("无效的 API Key")
	}
	return &t, err
}