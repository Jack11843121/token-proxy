package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Jack11843121/token-proxy/internal/model"
)

// TestChannel 对渠道做一次连通性测试，返回上游可用模型列表。
func TestChannel(ctx context.Context, ch model.Channel) ([]string, error) {
	url := strings.TrimRight(ch.BaseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if ch.Type == model.ChannelTypeAnthropic {
		req.URL.Path = strings.TrimRight(req.URL.Path, "/") + "/v1/models"
		req.Header.Set("x-api-key", ch.Key)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+ch.Key)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接失败：%v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d：%s", resp.StatusCode, extractUpstreamMessage(body))
	}
	var obj struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, fmt.Errorf("模型列表解析失败：%v", err)
	}
	models := make([]string, 0, len(obj.Data))
	for _, d := range obj.Data {
		if d.ID != "" {
			models = append(models, d.ID)
		}
	}
	return models, nil
}