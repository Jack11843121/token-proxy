# Token 网关 · 客户使用 API 说明

Token 网关对外提供 **OpenAI 兼容**接口。你只需在现有 OpenAI SDK/客户端中修改两处配置，即可调用网关背后接入的各家大模型。

## 1. 基本信息

| 项 | 值 |
| --- | --- |
| 接口地址（Base URL） | `http://<网关地址>:3000/v1` |
| 鉴权方式 | 请求头 `Authorization: Bearer <你的 API Key>` |
| API Key | 形如 `sk-xxxxxxxx`，由管理员在后台「令牌管理」分配 |

## 2. 可用接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/v1/chat/completions` | 对话补全（支持流式） |
| `POST` | `/v1/completions` | 文本补全 |
| `POST` | `/v1/embeddings` | 向量嵌入 |
| `GET` | `/v1/models` | 查询可用模型列表 |

## 3. 快速示例：对话补全

### curl

```bash
curl http://<网关地址>:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-你的APIKey" \
  -d '{
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

### Python（openai 库）

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://<网关地址>:3000/v1",
    api_key="sk-你的APIKey",
)

resp = client.chat.completions.create(
    model="deepseek-chat",
    messages=[{"role": "user", "content": "你好"}],
)
print(resp.choices[0].message.content)
```

### Node.js / TypeScript

```js
import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: 'http://<网关地址>:3000/v1',
  apiKey: 'sk-你的APIKey',
})

const res = await client.chat.completions.create({
  model: 'deepseek-chat',
  messages: [{ role: 'user', content: '你好' }],
})
console.log(res.choices[0].message.content)
```

## 4. 流式输出（stream）

设置 `stream: true`，网关会以 SSE 形式逐块返回，并在结束后返回用量 `usage`。

```python
stream = client.chat.completions.create(
    model="deepseek-chat",
    stream=True,
    messages=[{"role": "user", "content": "写一首关于秋天的诗"}],
)
for chunk in stream:
    delta = chunk.choices[0].delta.content or ""
    print(delta, end="", flush=True)
```

curl 流式调用：

```bash
curl -N http://<网关地址>:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-你的APIKey" \
  -d '{"model":"deepseek-chat","stream":true,"messages":[{"role":"user","content":"写一首诗"}]}'
```

## 5. 响应格式

非流式响应与 OpenAI 完全一致：

```json
{
  "id": "chatcmpl-xxxx",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "deepseek-chat",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "……"},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 30,
    "total_tokens": 42
  }
}
```

## 6. 错误处理

出现错误时，网关返回 OpenAI 风格错误体，且 HTTP 状态码与原因对应：

| 状态码 | 含义 |
| --- | --- |
| `401` | API Key 缺失或无效 |
| `403` | API Key / 账号被禁用、过期，或无权访问该模型 |
| `404` | 模型未开通（无可用渠道） |
| `429` | 额度不足，请联系管理员充值 |
| `502` | 上游渠道连接失败 |

```json
{
  "error": {
    "message": "额度已用尽，请联系管理员充值",
    "type": "invalid_request_error",
    "code": 429
  }
}
```

## 7. 说明

- **模型名**：可通过 `GET /v1/models` 查询网关当前开放的全部模型；网关会自动路由到已配置的厂商渠道，并透明处理 OpenAI ↔ Claude 等协议差异。
- **计费**：按实际消耗的 tokens（输入/输出分开计价）从你的账户额度中扣减，额度不足会被拒绝。
- **Key 安全**：请妥善保管 API Key，勿提交到公开仓库、勿直接暴露在前端页面。