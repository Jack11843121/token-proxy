package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/model"
)

const priceCols = `id, name, price_input, price_output, status, tags, created_at, updated_at`

func scanPrice(s scanner) (*model.Price, error) {
	var p model.Price
	err := s.Scan(&p.ID, &p.Name, &p.PriceInput, &p.PriceOutput, &p.Status, &p.Tags, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

type priceBody struct {
	Name        string  `json:"name" binding:"required"`
	PriceInput  float64 `json:"price_input"`
	PriceOutput float64 `json:"price_output"`
	Status      *int    `json:"status"`
	Tags        string  `json:"tags"`
}

// ListModels 列出模型价格。
func (h *Handler) ListModels(c *gin.Context) {
	offset, limit, kw := pageParams(c)
	where := " WHERE (name LIKE ? OR tags LIKE ?)"
	args := []interface{}{"%" + kw + "%", "%" + kw + "%"}
	var total int
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM models"+where, args...).Scan(&total); err != nil {
		fail(c, 500, err.Error())
		return
	}
	rows, err := h.DB.Query("SELECT "+priceCols+" FROM models"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	list := make([]*model.Price, 0)
	for rows.Next() {
		p, err := scanPrice(rows)
		if err != nil {
			fail(c, 500, err.Error())
			return
		}
		list = append(list, p)
	}
	ok(c, gin.H{"list": list, "total": total})
}

// CreateModel 新增模型价格。
func (h *Handler) CreateModel(c *gin.Context) {
	var b priceBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误：需要 name")
		return
	}
	status := statusOr(b.Status)
	now := time.Now().Unix()
	res, err := h.DB.Exec(`INSERT INTO models (name, price_input, price_output, status, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.Name, b.PriceInput, b.PriceOutput, status, b.Tags, now, now)
	if err != nil {
		fail(c, 400, "模型已存在")
		return
	}
	id, _ := res.LastInsertId()
	ok(c, gin.H{"id": id})
}

// UpdateModel 更新模型价格。
func (h *Handler) UpdateModel(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	var b priceBody
	if err := c.ShouldBindJSON(&b); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if _, err := h.DB.Exec(`UPDATE models SET name=?, price_input=?, price_output=?, status=?, tags=?, updated_at=? WHERE id=?`,
		b.Name, b.PriceInput, b.PriceOutput, statusOr(b.Status), b.Tags, time.Now().Unix(), id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"message": "已更新"})
}

// DeleteModel 删除模型价格。
func (h *Handler) DeleteModel(c *gin.Context) {
	id, okk := paramID(c)
	if !okk {
		return
	}
	if _, err := h.DB.Exec(`DELETE FROM models WHERE id = ?`, id); err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"message": "已删除"})
}