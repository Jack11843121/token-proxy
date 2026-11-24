// Package relay 实现 OpenAI 兼容的模型转发（中继）核心：
// API Key 鉴权 → 额度校验 → 渠道选择/故障切换 → 协议适配 → 计费扣费 → 日志。
package relay

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/billing"
	"github.com/Jack11843121/token-proxy/internal/model"
)

// Service 持有中继层依赖。
type Service struct {
	DB *sql.DB
}

// New 构建中继服务。
func New(db *sql.DB) *Service { return &Service{DB: db} }

// Register 挂载 /v1 中继路由（OpenAI 兼容）。
func Register(r *gin.Engine, s *Service) {
	v1 := r.Group("/v1")
	v1.POST("/chat/completions", s.ChatCompletions)
	v1.POST("/completions", s.Completions)
	v1.POST("/embeddings", s.Embeddings)
	v1.GET("/models", s.ListModels)
	v1.GET("/models/:model", s.GetModel)
}

// apiError 是中继层返回给调用方的错误。
type apiError struct {
	status int
	msg    string
}

func rerr(status int, msg string) *apiError { return &apiError{status: status, msg: msg} }

// result 是一次转发尝试的结果。
type result struct {
	promptT int
	compT   int
	status  int    // 上游/响应状态码（成功=200）
	err     string // 空 = 成功
	started bool   // 是否已向客户端写入任何字节（流式中途失败则无法回退）
	body    []byte // 非流式成功时的响应体
	ct      string // 非流式成功时的 Content-Type
}

// preflight 携带鉴权与渠道候选信息。
type preflight struct {
	token        model.Token
	user         model.User
	requestModel string
	channels     []model.Channel
}

// ChatCompletions 处理 /v1/chat/completions。
func (s *Service) ChatCompletions(c *gin.Context) { s.relay(c, "chat") }

// Completions 处理 /v1/completions（文本补全）。
func (s *Service) Completions(c *gin.Context) { s.relay(c, "completion") }

// Embeddings 处理 /v1/embeddings（向量）。
func (s *Service) Embeddings(c *gin.Context) { s.relay(c, "embedding") }

func (s *Service) relay(c *gin.Context, kind string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无法读取请求体")
		return
	}
	m := map[string]any{}
	if err := json.Unmarshal(body, &m); err != nil {
		writeError(c, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	reqModel, _ := m["model"].(string)
	if reqModel == "" {
		writeError(c, http.StatusBadRequest, "缺少 model 字段")
		return
	}
	stream := false
	if v, ok := m["stream"].(bool); ok {
		stream = v
	}

	pre, aerr := s.preflight(c, reqModel)
	if aerr != nil {
		writeError(c, aerr.status, aerr.msg)
		return
	}

	ip := c.ClientIP()
	start := time.Now()

	var (
		usedCh   model.Channel
		usedUp   string
		last     result
		attempts int
	)
	for _, ch := range pre.channels {
		up := upstreamModel(ch, reqModel)
		m["model"] = up
		var r result
		if ch.Type == model.ChannelTypeAnthropic {
			if kind != "chat" {
				r = result{status: http.StatusBadRequest, err: "Claude 渠道仅支持 chat/completions，请改配 OpenAI 兼容渠道处理 embeddings/completions"}
				break
			}
			r = relayAnthropic(c, ch, m, stream, up, reqModel)
		} else {
			r = relayOpenAI(c, ch, m, stream, kind)
		}
		usedCh, usedUp, last, attempts = ch, up, r, attempts+1
		if r.err == "" || r.started {
			break
		}
		// 未开始流式且失败 → 故障切换下一渠道
	}

	if last.err != "" {
		if !last.started {
			writeUpstreamError(c, last.status, last.err)
		}
		// 流式中途已回写，无法再写错误响应
	} else if !stream {
		if last.ct != "" {
			c.Header("Content-Type", last.ct)
		}
		c.Status(last.status)
		if len(last.body) > 0 {
			c.Writer.Write(last.body)
		}
	}
	s.finish(pre, usedCh, usedUp, reqModel, kind, last, ip, time.Since(start).Milliseconds())
	_ = attempts
}

// preflight 校验 API Key、用户状态、模型授权与额度，并选出候选渠道。
func (s *Service) preflight(c *gin.Context, requestModel string) (*preflight, *apiError) {
	key := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if key == "" {
		return nil, rerr(http.StatusUnauthorized, "缺少 API Key：请在 Authorization 头携带 Bearer sk-xxx")
	}
	var t model.Token
	err := s.DB.QueryRow(`SELECT id, user_id, name, key, status, expired_time, unlimited_quota, quota, used_quota, model_scope, request_count, created_at, updated_at, accessed_at
		FROM tokens WHERE key = ?`, key).
		Scan(&t.ID, &t.UserID, &t.Name, &t.Key, &t.Status, &t.ExpiredTime, &t.UnlimitedQuota, &t.Quota, &t.UsedQuota, &t.ModelScope, &t.RequestCount, &t.CreatedAt, &t.UpdatedAt, &t.AccessedAt)
	if err != nil {
		return nil, rerr(http.StatusUnauthorized, "无效的 API Key")
	}
	if t.Status != model.StatusEnabled {
		return nil, rerr(http.StatusForbidden, "该 API Key 已被禁用")
	}
	if t.ExpiredTime != 0 && t.ExpiredTime < time.Now().Unix() {
		return nil, rerr(http.StatusForbidden, "该 API Key 已过期")
	}
	var u model.User
	if err := s.DB.QueryRow(`SELECT id, username, status, quota, used_quota, request_count FROM users WHERE id = ?`, t.UserID).
		Scan(&u.ID, &u.Username, &u.Status, &u.Quota, &u.UsedQuota, &u.RequestCount); err != nil {
		return nil, rerr(http.StatusUnauthorized, "该 API Key 关联的用户不存在")
	}
	if u.Status != model.StatusEnabled {
		return nil, rerr(http.StatusForbidden, "账号已被禁用")
	}
	if t.ModelScope != "" && t.ModelScope != "*" && !containsItem(t.ModelScope, requestModel) {
		return nil, rerr(http.StatusNotFound, "该 API Key 无权访问模型 "+requestModel)
	}
	avail := u.Quota
	if t.Quota >= 0 {
		avail = t.Quota
	}
	if t.UnlimitedQuota != 1 && avail <= 0 {
		return nil, rerr(http.StatusTooManyRequests, "额度已用尽，请联系管理员充值")
	}
	channels := s.candidates(requestModel)
	if len(channels) == 0 {
		return nil, rerr(http.StatusNotFound, "模型 "+requestModel+" 暂未开通（无可用渠道）")
	}
	return &preflight{token: t, user: u, requestModel: requestModel, channels: channels}, nil
}

// candidates 取所有启用渠道，按模型过滤后依据 priority(升序) + weight(降序) 排序。
func (s *Service) candidates(requestModel string) []model.Channel {
	rows, err := s.DB.Query(`SELECT id, name, type, base_url, key, models, model_mapping, priority, weight, status, groups, created_at, updated_at
		FROM channels WHERE status = 1`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	list := make([]model.Channel, 0)
	for rows.Next() {
		var ch model.Channel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.BaseURL, &ch.Key, &ch.Models, &ch.ModelMapping, &ch.Priority, &ch.Weight, &ch.Status, &ch.Groups, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			continue
		}
		if supportsModel(ch.Models, requestModel) {
			list = append(list, ch)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Priority != list[j].Priority {
			return list[i].Priority < list[j].Priority
		}
		return list[i].Weight > list[j].Weight
	})
	return list
}

// finish 记录日志并（成功时）执行计费扣费与计数。
func (s *Service) finish(pre *preflight, ch model.Channel, upModel, reqModel, kind string, r result, ip string, latency int64) {
	inP, outP := billing.Price(s.DB, reqModel)
	cost := billing.Spend(inP, outP, r.promptT, r.compT)
	status := 1
	if r.err != "" {
		status = 0
	}
	if r.status == 0 && r.err != "" {
		r.status = http.StatusBadGateway
	}
	now := time.Now().Unix()
	s.DB.Exec(`INSERT INTO logs (user_id, token_id, channel_id, token_name, channel_name, model, upstream_model, type, prompt_tokens, completion_tokens, total_tokens, quota, status, error, ip, latency_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		pre.user.ID, pre.token.ID, ch.ID, pre.token.Name, ch.Name, reqModel, upModel, kind,
		r.promptT, r.compT, r.promptT+r.compT, cost, status, r.err, ip, latency, now)

	if status == 1 {
		nq := pre.user.Quota - cost
		if nq < 0 {
			nq = 0
		}
		s.DB.Exec(`UPDATE users SET quota = ?, used_quota = used_quota + ?, request_count = request_count + 1, updated_at = ? WHERE id = ?`,
			nq, cost, now, pre.user.ID)
		if pre.token.Quota >= 0 {
			nt := pre.token.Quota - cost
			if nt < 0 {
				nt = 0
			}
			s.DB.Exec(`UPDATE tokens SET quota = ?, used_quota = used_quota + ?, request_count = request_count + 1, accessed_at = ? WHERE id = ?`,
				nt, cost, now, pre.token.ID)
		} else {
			s.DB.Exec(`UPDATE tokens SET used_quota = used_quota + ?, request_count = request_count + 1, accessed_at = ? WHERE id = ?`,
				cost, now, pre.token.ID)
		}
	}
}

// ListModels 返回中继站可提供的模型列表（OpenAI /v1/models 格式）。
func (s *Service) ListModels(c *gin.Context) {
	type m struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	}
	seen := map[string]bool{}
	data := make([]m, 0)
	rows, err := s.DB.Query(`SELECT models, model_mapping FROM channels WHERE status = 1`)
	if err == nil {
		for rows.Next() {
			var modelsStr, mappingStr string
			rows.Scan(&modelsStr, &mappingStr)
			for _, name := range strings.Split(modelsStr, ",") {
				name = strings.TrimSpace(name)
				if name == "" || name == "*" {
					// "*" 表示全部；落到下方 models 定价表统一提供
					continue
				}
				if !seen[name] {
					seen[name] = true
					data = append(data, m{ID: name, Object: "model", OwnedBy: "token-proxy"})
				}
			}
		}
		rows.Close()
	}
	prows, err := s.DB.Query(`SELECT name FROM models WHERE status = 1`)
	if err == nil {
		for prows.Next() {
			var name string
			prows.Scan(&name)
			if !seen[name] {
				seen[name] = true
				data = append(data, m{ID: name, Object: "model", OwnedBy: "token-proxy"})
			}
		}
		prows.Close()
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
}

// GetModel 返回单个模型信息。
func (s *Service) GetModel(c *gin.Context) {
	name := c.Param("model")
	c.JSON(http.StatusOK, gin.H{"id": name, "object": "model", "owned_by": "token-proxy", "permission": []any{}})
}

// ---- 通用辅助 ----

func supportsModel(modelsStr, m string) bool {
	if modelsStr == "" || modelsStr == "*" {
		return true
	}
	return containsItem(modelsStr, m)
}

func containsItem(csv, item string) bool {
	for _, s := range strings.Split(csv, ",") {
		if strings.TrimSpace(s) == item {
			return true
		}
	}
	return false
}

func upstreamModel(ch model.Channel, requestModel string) string {
	if ch.ModelMapping == "" {
		return requestModel
	}
	var mm map[string]string
	if json.Unmarshal([]byte(ch.ModelMapping), &mm) == nil {
		if v, ok := mm[requestModel]; ok && v != "" {
			return v
		}
	}
	return requestModel
}

func writeError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": gin.H{"message": msg, "type": "invalid_request_error", "code": status}})
}

// writeUpstreamError 把上游或内部失败包装为 OpenAI 风格错误返回。
func writeUpstreamError(c *gin.Context, status int, msg string) {
	if status == 0 {
		status = http.StatusBadGateway
	}
	c.JSON(status, gin.H{"error": gin.H{"message": msg, "type": "upstream_error", "code": status}})
}