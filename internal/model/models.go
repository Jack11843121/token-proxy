// Package model 定义 token-proxy 的数据模型。
package model

// 渠道类型：openai 表示「OpenAI 兼容」协议（覆盖 OpenAI/DeepSeek/通义/Kimi/GLM/豆包/Gemini 兼容端点 及任意自定义中转），
// anthropic 表示 Claude Messages API（需要做 OpenAI ↔ Claude 协议转换）。
const (
	ChannelTypeOpenAI    = "openai"
	ChannelTypeAnthropic = "anthropic"
)

// 角色与状态常量。
const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	StatusEnabled  = 1
	StatusDisabled = 0
)

// User 中转站里的最终用户（配额单位 = 美元等价额度）。
type User struct {
	ID           int64   `json:"id"`
	Username     string  `json:"username"`
	Password     string  `json:"-"` // bcrypt 哈希，永不输出
	DisplayName  string  `json:"display_name"`
	Role         string  `json:"role"`
	Status       int     `json:"status"`
	Quota        float64 `json:"quota"`      // 剩余额度
	UsedQuota    float64 `json:"used_quota"` // 累计已用额度
	RequestCount int64   `json:"request_count"`
	Note         string  `json:"note"`
	CreatedAt    int64   `json:"created_at"`
	UpdatedAt    int64   `json:"updated_at"`
}

// Token 是下发给使用者的 API 令牌（sk- 开头）。
type Token struct {
	ID             int64   `json:"id"`
	UserID         int64   `json:"user_id"`
	Name           string  `json:"name"`
	Key            string  `json:"key"`             // 完整 sk- 密钥（管理员可查看）
	Status         int     `json:"status"`
	ExpiredTime    int64   `json:"expired_time"`    // 0 = 永不过期
	UnlimitedQuota int     `json:"unlimited_quota"` // 1 = 不限额度
	Quota          float64 `json:"quota"`           // -1 = 跟随用户额度
	UsedQuota      float64 `json:"used_quota"`
	ModelScope     string  `json:"model_scope"` // 允许访问的模型，逗号分隔；空 = 全部
	RequestCount   int64   `json:"request_count"`
	CreatedAt      int64   `json:"created_at"`
	UpdatedAt      int64   `json:"updated_at"`
	AccessedAt     int64   `json:"accessed_at"`
}

// Channel 表示一个上游大模型厂商渠道。
type Channel struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`       // openai | anthropic
	BaseURL      string `json:"base_url"`   // 上游接口地址
	Key          string `json:"key"`        // 上游 API Key
	Models       string `json:"models"`     // 该渠道支持的模型，逗号分隔
	ModelMapping string `json:"model_mapping"` // JSON：{"用户模型名":"上游模型名"}
	Priority     int    `json:"priority"`   // 越小越优先
	Weight       int    `json:"weight"`     // 同级权重（分流）
	Status       int    `json:"status"`
	Groups       string `json:"groups"` // 分组，逗号分隔；空 = 全组
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// Price 定义模型价格（单位：额度 / 每 100 万 tokens）。
type Price struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	PriceInput  float64 `json:"price_input"`  // 输入（prompt）每 1M tokens 的额度
	PriceOutput float64 `json:"price_output"` // 输出（completion）每 1M tokens 的额度
	Status      int     `json:"status"`
	Tags        string  `json:"tags"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

// Log 记录一次转发请求的用量与结果。
type Log struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"user_id"`
	TokenID         int64   `json:"token_id"`
	ChannelID       int64   `json:"channel_id"`
	TokenName       string  `json:"token_name"`
	ChannelName     string  `json:"channel_name"`
	Model           string  `json:"model"`
	UpstreamModel   string  `json:"upstream_model"`
	Type            string  `json:"type"`
	PromptTokens    int     `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	Quota           float64 `json:"quota"`
	Status          int     `json:"status"` // 1 成功 0 失败
	Error           string  `json:"error"`
	IP              string  `json:"ip"`
	LatencyMs       int64   `json:"latency_ms"`
	CreatedAt       int64   `json:"created_at"`
}