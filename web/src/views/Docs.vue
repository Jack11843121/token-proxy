<template>
  <div class="docs">
    <el-card>
      <h2>📖 客户使用 API 说明</h2>
      <p>Token 网关对外提供 <b>OpenAI 兼容</b>接口，你只需在现有的 OpenAI SDK/客户端里改两个地方：<code>base_url</code> 和 <code>api_key</code>，即可调用各家大模型。</p>

      <div class="box">
        <div class="kv"><span>中转接口地址</span><code>{{ origin }}/v1</code></div>
        <div class="kv"><span>鉴权方式</span><code>Authorization: Bearer sk-你的APIKey</code></div>
        <div class="kv"><span>获取 Key</span>由管理员在后台「令牌管理」中分配</div>
      </div>

      <h3>1. curl 调用</h3>
      <pre class="code"><code>curl {{ origin }}/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -d '{
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "你好"}]
  }'</code></pre>

      <h3>2. Python (openai 库)</h3>
      <pre class="code"><code>from openai import OpenAI

client = OpenAI(
    base_url="{{ origin }}/v1",
    api_key="sk-xxxxxxxx",
)
resp = client.chat.completions.create(
    model="deepseek-chat",
    messages=[{"role": "user", "content": "你好"}],
)
print(resp.choices[0].message.content)</code></pre>

      <h3>3. Node.js</h3>
      <pre class="code"><code>import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: '{{ origin }}/v1',
  apiKey: 'sk-xxxxxxxx',
})
const res = await client.chat.completions.create({
  model: 'deepseek-chat',
  messages: [{ role: 'user', content: '你好' }],
})
console.log(res.choices[0].message.content)</code></pre>

      <h3>4. 流式输出 (stream)</h3>
      <pre class="code"><code>curl {{ origin }}/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -d '{"model":"deepseek-chat","stream":true,"messages":[{"role":"user","content":"写一首诗"}]}'</code></pre>

      <h3>支持的接口</h3>
      <el-table :data="endpoints" border size="small">
        <el-table-column prop="method" label="方法" width="90" />
        <el-table-column prop="path" label="路径" width="220" />
        <el-table-column prop="desc" label="说明" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const origin = computed(() => window.location.origin)

const endpoints = [
  { method: 'POST', path: '/v1/chat/completions', desc: '对话补全（支持流式，OpenAI / Claude / DeepSeek / 通义 / Kimi / GLM / 豆包 等）' },
  { method: 'POST', path: '/v1/completions', desc: '文本补全（OpenAI 兼容渠道）' },
  { method: 'POST', path: '/v1/embeddings', desc: '向量嵌入（OpenAI 兼容渠道）' },
  { method: 'GET', path: '/v1/models', desc: '查询可用模型列表' },
]
</script>

<style scoped>
.docs { max-width: 900px; }
h2 { margin: 0 0 8px; }
h3 { margin: 22px 0 10px; color: #1e3a8a; }
.box { background: #f8fafc; border: 1px solid #e5e7eb; border-radius: 8px; padding: 14px 16px; margin: 14px 0; }
.kv { margin: 4px 0; }
.kv span { display: inline-block; width: 90px; color: #64748b; }
code { background: #eef2ff; padding: 1px 6px; border-radius: 4px; color: #4338ca; }
pre.code { background: #0f172a; color: #e2e8f0; padding: 16px; border-radius: 10px; overflow-x: auto; font-size: 13px; line-height: 1.6; }
pre.code code { background: transparent; color: #7dd3fc; padding: 0; }
</style>