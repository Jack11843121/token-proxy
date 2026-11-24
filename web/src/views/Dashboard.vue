<template>
  <div>
    <div class="cards">
      <div class="card"><div class="k">今日请求</div><div class="v">{{ d.today?.requests ?? 0 }}</div></div>
      <div class="card"><div class="k">今日 Token</div><div class="v">{{ fmt(d.today?.tokens) }}</div></div>
      <div class="card"><div class="k">今日消耗(额度)</div><div class="v money">{{ d.today?.quota?.toFixed(4) ?? 0 }}</div></div>
      <div class="card"><div class="k">用户 / 令牌 / 渠道</div><div class="v small">{{ d.counts?.users ?? 0 }} / {{ d.counts?.tokens ?? 0 }} / {{ d.counts?.channels ?? 0 }}</div></div>
    </div>

    <div class="cards" style="margin-top:14px">
      <div class="card"><div class="k">昨日请求</div><div class="v">{{ d.yesterday?.requests ?? 0 }}</div></div>
      <div class="card"><div class="k">累计请求</div><div class="v">{{ d.total?.requests ?? 0 }}</div></div>
      <div class="card"><div class="k">累计 Token</div><div class="v">{{ fmt(d.total?.tokens) }}</div></div>
      <div class="card"><div class="k">累计消耗(额度)</div><div class="v money">{{ d.total?.quota?.toFixed(4) ?? 0 }}</div></div>
    </div>

    <el-card style="margin-top:14px">
      <template #header>近 7 日调用趋势</template>
      <div class="bars">
        <div v-for="t in d.trend" :key="t.date" class="bar-col" :title="`${t.date}: ${t.requests} 次 / ${t.quota.toFixed(4)} 额度`">
          <div class="bar" :style="{ height: barHeight(t.requests) + 'px' }"></div>
          <div class="bar-label">{{ t.date.slice(5) }}</div>
          <div class="bar-num">{{ t.requests }}</div>
        </div>
        <el-empty v-if="!d.trend || !d.trend.length" description="暂无数据" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import http from '../api'

const d = ref({})

function fmt(n) {
  if (n == null) return 0
  if (n >= 1e9) return (n / 1e9).toFixed(2) + 'B'
  if (n >= 1e6) return (n / 1e6).toFixed(2) + 'M'
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K'
  return n
}

function barHeight(n) {
  return Math.max(4, Math.min(120, n))
}

onMounted(async () => {
  try {
    d.value = (await http.get('/api/dashboard')).data
  } catch (e) {}
})
</script>

<style scoped>
.cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
.card { background: #fff; border-radius: 10px; padding: 18px; box-shadow: 0 1px 3px rgba(0,0,0,.06); }
.k { color: #8492a6; font-size: 13px; }
.v { margin-top: 8px; font-size: 28px; font-weight: 700; color: #1f2937; }
.v.money { color: #e6a23c; }
.v.small { font-size: 20px; }
.bars { display: flex; align-items: flex-end; gap: 12px; height: 200px; padding: 10px 4px; overflow-x: auto; }
.bar-col { display: flex; flex-direction: column; align-items: center; justify-content: flex-end; gap: 4px; }
.bar { width: 30px; background: linear-gradient(180deg, #38bdf8, #1e3a8a); border-radius: 4px 4px 0 0; }
.bar-label { font-size: 11px; color: #8492a6; }
.bar-num { font-size: 12px; color: #1f2937; }
</style>