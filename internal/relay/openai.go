package relay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/model"
)

// relayOpenAI 转发到 OpenAI 兼容协议的上游（覆盖 OpenAI/DeepSeek/通义/Kimi/GLM/豆包/Gemini 兼容端点及任意自定义中转）。
func relayOpenAI(c *gin.Context, ch model.Channel, m map[string]any, stream bool, kind string) result {
	if stream {
		if _, ok := m["stream_options"]; !ok {
			m["stream_options"] = map[string]any{"include_usage": true}
		}
		m["stream"] = true
	}
	body, _ := json.Marshal(m)
	url := strings.TrimRight(ch.BaseURL, "/") + paths[kind]
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return result{status: http.StatusInternalServerError, err: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ch.Key)
	resp, err := httpClient.Do(req)
	if err != nil {
		return result{status: http.StatusBadGateway, err: fmt.Sprintf("上游连接失败：%v", err)}
	}
	defer resp.Body.Close()
	return forwardOpenAI(c, resp, stream)
}

// forwardOpenAI 处理 OpenAI 兼容上游响应：非流式返回 body+usage，流式边读边回写并累加 usage。
func forwardOpenAI(c *gin.Context, resp *http.Response, stream bool) result {
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}

	if !stream {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return result{status: resp.StatusCode, err: extractUpstreamMessage(body)}
		}
		u := usageFromBody(body)
		return result{promptT: u.PromptTokens, compT: u.CompletionTokens, status: http.StatusOK, body: body, ct: ct}
	}

	// ---- 流式 ----
	c.Header("Content-Type", ct)
	c.Status(resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result{status: resp.StatusCode, err: extractUpstreamMessage(body)}
	}
	flusher, _ := c.Writer.(http.Flusher)
	var (
		u       usageJSON
		started bool
	)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if u2, ok := usageFromData(line); ok {
			u = u2
		}
		if _, err := c.Writer.WriteString(line + "\n"); err != nil {
			break // 客户端已断开，停止转发
		}
		started = true
		if flusher != nil {
			flusher.Flush()
		}
	}
	return result{promptT: u.PromptTokens, compT: u.CompletionTokens, status: http.StatusOK, started: started}
}