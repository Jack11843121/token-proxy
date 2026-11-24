package api

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/model"
	"github.com/Jack11843121/token-proxy/internal/relay"
)

const channelCols = `id, name, type, base_url, key, models, model_mapping, priority, weight, status, groups, created_at, updated_at`

func scanChannel(s scanner) (*model.Channel, error) {
	var ch model.Channel
	err := s.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.BaseURL, &ch.Key, &ch.Models, &ch.ModelMapping, &ch.Priority, &ch.Weight, &ch.Status, &ch.Groups, &ch.CreatedAt, &ch.UpdatedAt)
	return &ch, err
}

type channelBody struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	BaseURL      string `json:"base_url" binding:"required"`
	Key          string `json:"key"`
	Models       string `json:"models"`
	ModelMapping string `json:"model_mapping"`
	Priority     int    `json:"priority"`
	Weight       int    `json:"weight"`
	Status       *int   `json:"status"`
	Groups       string `json:"groups"`
}

// ListChannels 列出渠道。
func (h *Handler) ListChannels(c *gin.Context) {
	offset, limit, kw := pageParams(c)
	where := " WHERE (name LIKE ? OR base_url LIKE ?)"
	args := []interface{}{"%" + kw + "%", "%" + kw + "%"}
	var total int
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM channels"+where, args...).Scan(&total); err != nil {
		fail(c, 500, err.Error())
		return
	}
	rows, err := h.DB.Query("SELECT "+channelCols+" FROM channels"+where+" ORDER BY priority ASC, id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	list := make([]*model.Channel, 0)
	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			fail(c, 500, err.Error())
			return
		}
		list = append(list, ch)
	}
	ok(c, gin.H{"list": list, "total": total})
}

// CreateChannel 新增渠道。
func (h *Handler) CreateChannel(c *gin.Context) {
	var b channelBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误：需要 base_url")
		return
	}
	if b.Type == "" {
		b.Type = model.ChannelTypeOpenAI
	}
	status := statusOr(b.Status)
	if b.Weight <= 0 {
		b.Weight = 1
	}
	now := time.Now().Unix()
	res, err := h.DB.Exec(`INSERT INTO channels (name, type, base_url, key, models, model_mapping, priority, weight, status, groups, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Name, b.Type, b.BaseURL, b.Key, b.Models, b.ModelMapping, b.Priority, b.Weight, status, b.Groups, now, now)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	ok(c, gin.H{"id": id})
}

// UpdateChannel 更新渠道。
func (h *Handler) UpdateChannel(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	var b channelBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if b.Weight <= 0 {
		b.Weight = 1
	}
	if _, err := h.DB.Exec(`UPDATE channels SET name=?, type=?, base_url=?, key=?, models=?, model_mapping=?, priority=?, weight=?, status=?, groups=?, updated_at=? WHERE id=?`,
		b.Name, b.Type, b.BaseURL, b.Key, b.Models, b.ModelMapping, b.Priority, b.Weight, statusOr(b.Status), b.Groups, time.Now().Unix(), id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"message": "已更新"})
}

// DeleteChannel 删除渠道。
func (h *Handler) DeleteChannel(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	if _, err := h.DB.Exec(`DELETE FROM channels WHERE id = ?`, id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"message": "已删除"})
}

// TestChannel 测试渠道连通性并返回其可用模型。
func (h *Handler) TestChannel(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	ch, err := scanChannel(h.DB.QueryRow("SELECT "+channelCols+" FROM channels WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		fail(c, 404, "渠道不存在")
		return
	}
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	models, err := relay.TestChannel(ctx, *ch)
	if err != nil {
		ok(c, gin.H{"success": false, "error": err.Error()})
		return
	}
	ok(c, gin.H{"success": true, "models": models})
}