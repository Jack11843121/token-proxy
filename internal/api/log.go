package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/model"
)

const logCols = `id, user_id, token_id, channel_id, token_name, channel_name, model, upstream_model, type, prompt_tokens, completion_tokens, total_tokens, quota, status, error, ip, latency_ms, created_at`

func scanLog(s scanner) (*model.Log, error) {
	var l model.Log
	err := s.Scan(&l.ID, &l.UserID, &l.TokenID, &l.ChannelID, &l.TokenName, &l.ChannelName, &l.Model, &l.UpstreamModel, &l.Type, &l.PromptTokens, &l.CompletionTokens, &l.TotalTokens, &l.Quota, &l.Status, &l.Error, &l.IP, &l.LatencyMs, &l.CreatedAt)
	return &l, err
}

// ListLogs 分页查询请求日志，支持多条件过滤。
func (h *Handler) ListLogs(c *gin.Context) {
	offset, limit, kw := pageParams(c)
	where := " WHERE 1=1"
	args := []interface{}{}
	if kw != "" {
		where += " AND (model LIKE ? OR token_name LIKE ? OR channel_name LIKE ?)"
		args = append(args, "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	for _, f := range []struct{ param, col string }{
		{"user_id", "user_id"}, {"token_id", "token_id"}, {"channel_id", "channel_id"},
		{"model", "model"}, {"status", "status"},
	} {
		if v := c.Query(f.param); v != "" {
			where += " AND " + f.col + " = ?"
			args = append(args, v)
		}
	}
	if v := c.Query("start"); v != "" {
		where += " AND created_at >= ?"
		args = append(args, v)
	}
	if v := c.Query("end"); v != "" {
		where += " AND created_at <= ?"
		args = append(args, v)
	}
	var total int
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM logs"+where, args...).Scan(&total); err != nil {
		fail(c, 500, err.Error())
		return
	}
	rows, err := h.DB.Query("SELECT "+logCols+" FROM logs"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	list := make([]*model.Log, 0)
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			fail(c, 500, err.Error())
			return
		}
		list = append(list, l)
	}
	ok(c, gin.H{"list": list, "total": total})
}

// Dashboard 聚合用量统计。
func (h *Handler) Dashboard(c *gin.Context) {
	now := time.Now()
	todayStart := dayStart(now)
	yStart := todayStart - 86400

	var totalReq, totalTok, totalPrompt, totalComp int64
	var totalQuota float64
	h.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(prompt_tokens),0), COALESCE(SUM(completion_tokens),0), COALESCE(SUM(quota),0) FROM logs`).
		Scan(&totalReq, &totalTok, &totalPrompt, &totalComp, &totalQuota)

	aggr := func(from, to int64) (int64, int64, float64) {
		var n, tok int64
		var q float64
		h.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(quota),0) FROM logs WHERE created_at >= ? AND created_at < ?`, from, to).
			Scan(&n, &tok, &q)
		return n, tok, q
	}
	todayN, todayTok, todayQuota := aggr(todayStart, todayStart+86400)
	yN, yTok, yQuota := aggr(yStart, todayStart)

	var users, tokens, channels int
	h.DB.QueryRow(`SELECT (SELECT COUNT(*) FROM users), (SELECT COUNT(*) FROM tokens), (SELECT COUNT(*) FROM channels)`).
		Scan(&users, &tokens, &channels)

	// 7 日趋势
	trend := make([]gin.H, 0, 7)
	rows, err := h.DB.Query(`SELECT strftime('%Y-%m-%d', created_at, 'unixepoch', 'localtime') d, COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(quota),0)
		FROM logs WHERE created_at >= ? GROUP BY d ORDER BY d`, yStart-5*86400)
	if err == nil {
		for rows.Next() {
			var d string
			var n, tok int64
			var q float64
			if rows.Scan(&d, &n, &tok, &q) == nil {
				trend = append(trend, gin.H{"date": d, "requests": n, "tokens": tok, "quota": q})
			}
		}
		rows.Close()
	}

	ok(c, gin.H{
		"today":      gin.H{"requests": todayN, "tokens": todayTok, "quota": todayQuota},
		"yesterday":  gin.H{"requests": yN, "tokens": yTok, "quota": yQuota},
		"total":      gin.H{"requests": totalReq, "tokens": totalTok, "prompt_tokens": totalPrompt, "completion_tokens": totalComp, "quota": totalQuota},
		"counts":     gin.H{"users": users, "tokens": tokens, "channels": channels},
		"trend":      trend,
	})
}

func dayStart(t time.Time) int64 {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).Unix()
}