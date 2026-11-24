# 🌉 Token 网关 token-proxy

> 开源多模型 **Token 网关 / 大模型中转站**（参考 new-api 设计）。用一个 OpenAI 兼容接口，把 **OpenAI / Claude / Gemini / DeepSeek / 通义千问 / Kimi / 智谱 GLM / 豆包** 等各家大模型聚合成一条中继出口，自带用户/额度/计费/日志/统计管理。

**单二进制部署 · 内置 SQLite · 内置 Web 管理后台 · 无需 Docker/MySQL/Node**，Linux 一键安装，开箱即用。

── 网关大师系列的**第四个网关**：WA 网关（WhatsApp）· IP 网关（代理）· NG 网关（Web/API）· 🆕 **Token 网关（大模型）**

---

## ✨ 产品特性

| 模块 | 能力 |
| --- | --- |
| 🚀 **OpenAI 兼容中转** | 统一 `/v1` 接口，Python `openai` / Node `openai` / 各类客户端只改 `base_url` + `api_key` 即可调用**多家**模型 |
| 🔌 **多渠道聚合** | 每厂商一个「渠道」，支持优先级 + 权重调度、**故障自动切换**下一渠道、模型名映射 |
| 🔁 **协议自动转换** | OpenAI ↔ Claude（Anthropic Messages API）双向转换，**流式（SSE）**同样兼容 |
| 👤 **用户 & 令牌** | 多用户、`sk-` 令牌；可设独立额度/不限额度/过期时间/模型白名单/启停 |
| 💰 **计费系统** | 模型按「输入/输出每 100 万 tokens」定价（美元等价额度），实时扣费、充值/扣减、用量留痕 |
| 📊 **用量统计** | 今日/昨日/累计请求与 Token、近 7 日趋势、用户/令牌/渠道概览 |
| 📝 **调用日志** | 每次请求的成功/失败、模型、上游、Tokens、消耗额度、延迟、错误信息，多条件检索 |
| 🛡️ **安全** | 后台 JWT 登录、令牌鉴权、额度限制、`Authorization: Bearer` 校验 |
| ⚡ **部署极简** | 单二进制（前端已嵌入）+ SQLite（免运维），`install.sh` / Docker 两种一键安装 |

## 🏗️ 架构

```
客户端 (openai SDK)
      │  POST /v1/chat/completions  (Authorization: Bearer sk-xxx)
      ▼
┌──────────────────────────── token-proxy (Go 单二进制) ────────────────────────────┐
│  ① 鉴权：校验 sk- 令牌 → 用户状态 / 额度 / 模型白名单                              │
│  ② 路由：按模型选择渠道（优先级/权重），失败自动切换                                │
│  ③ 适配：OpenAI 透传｜OpenAI↔Claude 协议转换（含流式）                              │
│  ④ 计费：按模型价格扣减额度、写入日志、更新统计                                     │
│  ├─ Web 管理后台（内嵌 SPA）：用户/令牌/渠道/模型价格/日志/统计                      │
│  └─ SQLite（内嵌，零依赖）                                                         │
└──────────────────────────────────────────────────────────────────────────────────┘
      │                         │                          │
   OpenAI                    Claude                     DeepSeek/通义/Kimi/GLM/豆包…
```

## 🚀 快速开始

### 方式一：一键安装（Linux，推荐）

```bash
curl -fsSL https://www.dashi8.com/releases/token-proxy-install.sh -o install.sh
sudo bash install.sh
```

脚本自动下载预构建静态二进制、生成配置、注册并启动 systemd 服务，随后打印管理后台地址。

### 方式二：Docker

```bash
git clone https://github.com/Jack11843121/token-proxy.git
cd token-proxy
docker compose up -d --build
# 或使用预构建镜像
```

### 方式三：源码构建

```bash
git clone https://github.com/Jack11843121/token-proxy.git
cd token-proxy
make build          # 前端 + 后端，产物在 bin/token-proxy
./bin/token-proxy   # 默认监听 :3000
```

### 安装后三步接入

1. 打开 `http://<服务器IP>:3000`（默认 `root / 123456`，**请立即改密码**）
2. 「渠道管理」→ 新增各厂商渠道（填各自的 API Key 与 Base URL）
3. 「用户管理」新增用户并充值额度 → 「令牌管理」生成 `sk-` Key 发给使用者

## 🔑 客户如何使用（发给使用方）

客户只需把标准的 OpenAI 客户端指向中转站：

```python
from openai import OpenAI
client = OpenAI(base_url="http://<你的域名或IP>:3000/v1", api_key="sk-xxxxxxxx")
resp = client.chat.completions.create(model="deepseek-chat",
    messages=[{"role":"user","content":"你好"}])
print(resp.choices[0].message.content)
```

完整说明见 [docs/API.md](docs/API.md)，或部署后访问管理后台内的「API 文档」页。

## 🧩 支持渠道（Base URL 一览）

| 厂商 | 类型 | Base URL |
| --- | --- | --- |
| OpenAI | OpenAI 兼容 | `https://api.openai.com/v1` |
| DeepSeek | OpenAI 兼容 | `https://api.deepseek.com/v1` |
| 通义千问 | OpenAI 兼容 | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| Kimi | OpenAI 兼容 | `https://api.moonshot.cn/v1` |
| 智谱 GLM | OpenAI 兼容 | `https://open.bigmodel.cn/api/paas/v4` |
| 豆包 | OpenAI 兼容 | `https://ark.cn-beijing.volces.com/api/v3` |
| Gemini | OpenAI 兼容 | `https://generativelanguage.googleapis.com/v1beta/openai` |
| Claude | Claude(Messages) | `https://api.anthropic.com` |
| 本地 Ollama / vLLM | OpenAI 兼容 | `http://127.0.0.1:11434/v1` |

## ⚙️ 配置（环境变量）

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `TOKEN_PROXY_LISTEN` | `:3000` | 监听地址 |
| `TOKEN_PROXY_DATA_DIR` | `./data` | 数据目录（SQLite 与日志） |
| `TOKEN_PROXY_DB` | `<data>/token-proxy.db` | 数据库文件路径 |
| `TOKEN_PROXY_ADMIN_USER` | `root` | 初始管理员（首次启动 seed） |
| `TOKEN_PROXY_ADMIN_PASS` | `123456` | 初始管理员密码 |
| `TOKEN_PROXY_JWT_SECRET` | — | 后台登录签名密钥（生产务必修改） |
| `TOKEN_PROXY_SESSION_SECONDS` | `604800` | 后台会话有效期（秒） |

## 📁 目录结构

```
token-proxy/
├── main.go               # 入口：注册路由、托管内嵌前端
├── internal/
│   ├── api/              # 后台管理 REST API（用户/令牌/渠道/模型/日志/统计）
│   ├── relay/            # 中继核心：鉴权/路由/OpenAI↔Claude 适配/流式
│   ├── billing/          # 模型定价与额度计算
│   ├── auth/             # JWT
│   ├── middleware/       # CORS / 后台鉴权
│   ├── store/            # SQLite 迁移与初始数据
│   ├── model/            # 数据模型
│   └── config/           # 配置加载
├── web/                  # Vue3 + Element Plus 管理后台（源码）
├── ui/dist/              # 嵌入到二进制里的前端产物
├── docs/API.md           # 客户使用 API 文档
├── scripts/              # 冒烟测试与 mock 上游
├── install.sh            # 一键安装（下载二进制 + systemd）
├── Dockerfile / docker-compose.yml
└── Makefile              # build / release（交叉编译）
```

## 📄 License

[MIT](LICENSE) · 开源免费使用，定制开发/私有化部署欢迎联系：微信 jack11843121 · 邮箱 11843121@qq.com