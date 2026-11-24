package relay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jack11843121/token-proxy/internal/model"
)

// claudeMessage / claudeRequest 对应 Anthropic Messages API 的请求体。
type claudeMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type claudeRequest struct {
	Model         string          `json:"model"`
	Messages      []claudeMessage `json:"messages"`
	System        string          `json:"system,omitempty"`
	MaxTokens     int             `json:"max_tokens"`
	Stream        bool            `json:"stream,omitempty"`
	Temperature   *float64        `json:"temperature,omitempty"`
	TopP          *float64        `json:"top_p,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeResponse struct {
	ID         string          `json:"id"`
	Model      string          `json:"model"`
	Content    []claudeContent `json:"content"`
	StopReason string          `json:"stop_reason"`
	Usage      claudeUsage     `json:"usage"`
}

// relayAnthropic 将 OpenAI 请求转发到 Claude（Anthropic Messages API），并把响应/流式事件回转成 OpenAI 格式。
func relayAnthropic(c *gin.Context, ch model.Channel, m map[string]any, stream bool, upstream, requestModel string) result {
	sys, msgs := convertMessages(m)
	maxTokens := 4096
	if v, ok := toInt(m["max_tokens"]); ok {
		maxTokens = v
	} else if v, ok := toInt(m["max_completion_tokens"]); ok {
		maxTokens = v
	}
	req := claudeRequest{Model: upstream, Messages: msgs, System: sys, MaxTokens: maxTokens, Stream: stream}
	if v, ok := m["temperature"].(float64); ok {
		req.Temperature = &v
	}
	if v, ok := m["top_p"].(float64); ok {
		req.TopP = &v
	}
	req.StopSequences = toStringSlice(m["stop"])

	body, _ := json.Marshal(req)
	url := strings.TrimRight(ch.BaseURL, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return result{status: http.StatusInternalServerError, err: err.Error()}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", ch.Key)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return result{status: http.StatusBadGateway, err: fmt.Sprintf("上游连接失败：%v", err)}
	}
	defer resp.Body.Close()
	return forwardAnthropic(c, resp, stream, requestModel)
}

// forwardAnthropic 处理 Claude 上游响应并转换为 OpenAI 格式。
func forwardAnthropic(c *gin.Context, resp *http.Response, stream bool, requestModel string) result {
	if !stream {
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return result{status: resp.StatusCode, err: extractUpstreamMessage(raw)}
		}
		var cr claudeResponse
		if err := json.Unmarshal(raw, &cr); err != nil {
			return result{status: http.StatusBadGateway, err: "Claude 响应解析失败：" + err.Error()}
		}
		text := claudeText(cr.Content)
		out := map[string]any{
			"id":      cr.ID,
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   requestModel,
			"choices": []any{map[string]any{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": text},
				"finish_reason": mapStopReason(cr.StopReason),
			}},
			"usage": map[string]any{
				"prompt_tokens":     cr.Usage.InputTokens,
				"completion_tokens": cr.Usage.OutputTokens,
				"total_tokens":      cr.Usage.InputTokens + cr.Usage.OutputTokens,
			},
		}
		b, _ := json.Marshal(out)
		return result{promptT: cr.Usage.InputTokens, compT: cr.Usage.OutputTokens, status: http.StatusOK, body: b, ct: "application/json"}
	}

	// ---- 流式：Claude SSE → OpenAI SSE ----
	c.Header("Content-Type", "text/event-stream")
	c.Status(resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return result{status: resp.StatusCode, err: extractUpstreamMessage(raw)}
	}
	flusher, _ := c.Writer.(http.Flusher)
	id := randID("chatcmpl")
	created := time.Now().Unix()
	var inputTokens, outputTokens int
	stopReason := ""
	started := false

	emit := func(chunk map[string]any) {
		b, _ := json.Marshal(chunk)
		if _, err := c.Writer.WriteString("data: " + string(b) + "\n\n"); err == nil {
			started = true
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
	roleChunk := func() map[string]any {
		return map[string]any{
			"id": id, "object": "chat.completion.chunk", "created": created, "model": requestModel,
			"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant"}, "finish_reason": nil}},
		}
	}
	roleSent := false

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		s := strings.TrimSpace(line)
		if !strings.HasPrefix(s, "data:") {
			continue
		}
		s = strings.TrimSpace(strings.TrimPrefix(s, "data:"))
		var ev struct {
			Type  string `json:"type"`
			Delta struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
			Message struct {
				ID string `json:"id"`
			} `json:"message"`
			Usage claudeUsage `json:"usage"`
		}
		if json.Unmarshal([]byte(s), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "message_start":
			inputTokens = ev.Usage.InputTokens
			if ev.Message.ID != "" {
				id = ev.Message.ID
			}
			if !roleSent {
				emit(roleChunk())
				roleSent = true
			}
		case "content_block_delta":
			if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
				if !roleSent {
					emit(roleChunk())
					roleSent = true
				}
				emit(map[string]any{
					"id": id, "object": "chat.completion.chunk", "created": created, "model": requestModel,
					"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": ev.Delta.Text}, "finish_reason": nil}},
				})
			}
		case "message_delta":
			if ev.Delta.StopReason != "" {
				stopReason = ev.Delta.StopReason
			}
			outputTokens = ev.Usage.OutputTokens
		case "message_stop":
			emit(map[string]any{
				"id": id, "object": "chat.completion.chunk", "created": created, "model": requestModel,
				"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": mapStopReason(stopReason)}},
				"usage": map[string]any{
					"prompt_tokens":     inputTokens,
					"completion_tokens": outputTokens,
					"total_tokens":      inputTokens + outputTokens,
				},
			})
			if _, err := c.Writer.WriteString("data: [DONE]\n\n"); err == nil && flusher != nil {
				flusher.Flush()
			}
		}
	}
	return result{promptT: inputTokens, compT: outputTokens, status: http.StatusOK, started: started}
}

// convertMessages 把 OpenAI messages 转为 Claude 结构的 (system, messages)。
func convertMessages(m map[string]any) (string, []claudeMessage) {
	var system strings.Builder
	out := make([]claudeMessage, 0)
	ms, _ := m["messages"].([]any)
	for _, mi := range ms {
		mm, ok := mi.(map[string]any)
		if !ok {
			continue
		}
		role, _ := mm["role"].(string)
		content := mm["content"]
		switch role {
		case "system", "developer":
			system.WriteString(contentText(content))
			system.WriteString("\n")
		case "user", "assistant":
			out = append(out, claudeMessage{Role: role, Content: normalizeContent(content)})
		default: // tool / function 等 → 作为 user 文本补充
			out = append(out, claudeMessage{Role: "user", Content: contentText(content)})
		}
	}
	return strings.TrimSpace(system.String()), out
}

// contentText 提取文本内容（string 或 OpenAI 多段内容中的 text）。
func contentText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, it := range v {
			if im, ok := it.(map[string]any); ok {
				if im["type"] == "text" {
					if t, ok := im["text"].(string); ok {
						b.WriteString(t)
					}
				}
			}
		}
		return b.String()
	}
	return ""
}

// normalizeContent 将 OpenAI 内容转为 Claude 可接受的内容（文本 + data URL 图片的 base64 转换）。
func normalizeContent(content any) any {
	arr, ok := content.([]any)
	if !ok {
		return content
	}
	blocks := make([]any, 0)
	for _, it := range arr {
		im, ok := it.(map[string]any)
		if !ok {
			continue
		}
		switch im["type"] {
		case "text":
			if t, ok := im["text"].(string); ok {
				blocks = append(blocks, map[string]any{"type": "text", "text": t})
			}
		case "image_url":
			if iu, ok := im["image_url"].(map[string]any); ok {
				if u, ok := iu["url"].(string); ok && strings.HasPrefix(u, "data:") {
					if mime, data, ok2 := parseDataURL(u); ok2 {
						blocks = append(blocks, map[string]any{
							"type":   "image",
							"source": map[string]any{"type": "base64", "media_type": mime, "data": data},
						})
					}
				}
			}
		}
	}
	return blocks
}

func parseDataURL(u string) (mime, data string, ok bool) {
	rest := strings.TrimPrefix(u, "data:")
	i := strings.IndexByte(rest, ',')
	if i < 0 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}

func claudeText(content []claudeContent) string {
	var b strings.Builder
	for _, c := range content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

func mapStopReason(s string) string {
	switch s {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return "stop"
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}