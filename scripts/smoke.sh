#!/usr/bin/env bash
# smoke.sh —— 端到端冒烟测试：mock 上游 + token-proxy 全链路。
# 用法：bash scripts/smoke.sh [base_url]
set -euo pipefail

cd "$(dirname "$0")/.."
BASE="${1:-http://127.0.0.1:3000}"
export TOKEN_PROXY_LISTEN=:3000 TOKEN_PROXY_DATA_DIR=/tmp/tp-smoke TOKEN_PROXY_ADMIN_PASS=123456

rm -rf /tmp/tp-smoke
python3 scripts/mock_upstream.py 9999 &
MOCK=$!
./bin/token-proxy &
PROXY=$!
trap 'kill $MOCK $PROXY 2>/dev/null || true' EXIT
sleep 2

echo "== health =="; curl -s "$BASE/health"; echo

JWT=$(curl -s -X POST "$BASE/api/auth/login" -H 'Content-Type: application/json' -d '{"username":"root","password":"123456"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["token"])')
AUTH="Authorization: Bearer $JWT"
echo "== 登录成功，JWT 长度 ${#JWT} =="

echo "== 创建用户 alice =="
curl -s -X POST "$BASE/api/users" -H "$AUTH" -H 'Content-Type: application/json' -d '{"username":"alice","password":"123456","display_name":"Alice","quota":100}'; echo

echo "== 创建渠道（指向 mock 上游）=="
curl -s -X POST "$BASE/api/channels" -H "$AUTH" -H 'Content-Type: application/json' -d '{"name":"mock","type":"openai","base_url":"http://127.0.0.1:9999/v1","key":"mock-key","models":"test-model,gpt-4o"}'; echo

echo "== 渠道连通性测试 =="
curl -s -X POST "$BASE/api/channels/1/test" -H "$AUTH"; echo

TK=$(curl -s -X POST "$BASE/api/tokens" -H "$AUTH" -H 'Content-Type: application/json' -d '{"user_id":2,"name":"alice-key"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["key"])')
echo "== 创建令牌 OK =="

echo "== GET /v1/models =="
curl -s "$BASE/v1/models" -H "Authorization: Bearer $TK"; echo

echo "== 非流式 chat/completions =="
curl -s "$BASE/v1/chat/completions" -H "Authorization: Bearer $TK" -H 'Content-Type: application/json' -d '{"model":"test-model","messages":[{"role":"user","content":"hi"}]}'; echo

echo "== 流式 chat/completions =="
curl -s -N "$BASE/v1/chat/completions" -H "Authorization: Bearer $TK" -H 'Content-Type: application/json' -d '{"model":"test-model","stream":true,"messages":[{"role":"user","content":"hi"}]}' | head -8

echo "== 无效 Key 应返回 401 =="
curl -s -o /dev/null -w "%{http_code}\n" "$BASE/v1/chat/completions" -H "Authorization: Bearer sk-bad" -H 'Content-Type: application/json' -d '{"model":"test-model","messages":[]}'

echo "== dashboard =="
curl -s "$BASE/api/dashboard" -H "$AUTH" | python3 -m json.tool

echo "== 最近日志 =="
curl -s "$BASE/api/logs?page_size=3" -H "$AUTH" | python3 -c 'import sys,json; d=json.load(sys.stdin)["data"]; [print(l["model"], l["status"], l["prompt_tokens"], l["completion_tokens"], l["quota"]) for l in d["list"]]'

echo "== 剩余额度（应已扣减）=="
curl -s "$BASE/api/users/2" -H "$AUTH" | python3 -c 'import sys,json; u=json.load(sys.stdin)["data"]; print("quota=", u["quota"], "used=", u["used_quota"], "requests=", u["request_count"])'

echo "✅ SMOKE DONE"