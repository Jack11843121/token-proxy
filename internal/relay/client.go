package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// httpClient 同时用于非流式与流式转发：仅限制响应头超时，不限制整体 body 读取时长（流式需要长时间保持）。
var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 60 * time.Second,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
	},
}

// paths 把逻辑端点映射为 OpenAI 兼容路径。
var paths = map[string]string{
	"chat":       "/chat/completions",
	"completion": "/completions",
	"embedding":  "/embeddings",
}

type usageJSON struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// usageFromData 从一行 SSE `data: {...}` 中解析 usage（用于流式 include_usage）。
func usageFromData(line string) (usageJSON, bool) {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "data:") {
		return usageJSON{}, false
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
	if s == "" || s == "[DONE]" {
		return usageJSON{}, false
	}
	var obj struct {
		Usage *usageJSON `json:"usage"`
	}
	if json.Unmarshal([]byte(s), &obj) == nil && obj.Usage != nil {
		return *obj.Usage, true
	}
	return usageJSON{}, false
}

// usageFromBody 从完整 JSON 响应体中解析 usage。
func usageFromBody(body []byte) usageJSON {
	var obj struct {
		Usage usageJSON `json:"usage"`
	}
	_ = json.Unmarshal(body, &obj)
	return obj.Usage
}

// extractUpstreamMessage 从上游错误响应中提取可读信息。
func extractUpstreamMessage(body []byte) string {
	var obj struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &obj) == nil {
		if obj.Error.Message != "" {
			return obj.Error.Message
		}
		if obj.Message != "" {
			return obj.Message
		}
	}
	s := strings.TrimSpace(string(body))
	if len(s) > 500 {
		s = s[:500]
	}
	if s == "" {
		return "上游返回空响应"
	}
	return s
}

// randID 生成随机十六进制 ID（用于流式响应的 chatcmpl id）。
func randID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return prefix + "-stream"
	}
	return prefix + "-" + hex.EncodeToString(b)
}