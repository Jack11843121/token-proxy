<template>
  <div>
    <div class="toolbar">
      <el-input v-model="kw" placeholder="搜索模型/令牌/渠道" clearable style="width: 200px" @keyup.enter="load" @clear="load" />
      <el-input v-model="modelFilter" placeholder="按模型精确过滤" clearable style="width: 200px" @keyup.enter="load" @clear="load" />
      <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px" @change="load">
        <el-option label="成功" :value="1" />
        <el-option label="失败" :value="0" />
      </el-select>
      <el-date-picker v-model="range" type="datetimerange" start-placeholder="开始" end-placeholder="结束" style="width: 360px" @change="load" />
      <el-button @click="load">查询</el-button>
    </div>

    <el-table :data="list" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="时间" width="160">
        <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column prop="token_name" label="令牌" min-width="100" show-overflow-tooltip />
      <el-table-column prop="channel_name" label="渠道" min-width="100" show-overflow-tooltip />
      <el-table-column prop="model" label="模型" min-width="130" show-overflow-tooltip />
      <el-table-column label="Tokens" width="150">
        <template #default="{ row }">{{ row.prompt_tokens }} / {{ row.completion_tokens }} ({{ row.total_tokens }})</template>
      </el-table-column>
      <el-table-column label="消耗" width="100">
        <template #default="{ row }"><span class="quota">{{ Number(row.quota).toFixed(4) }}</span></template>
      </el-table-column>
      <el-table-column label="状态" width="76">
        <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '成功' : '失败' }}</el-tag></template>
      </el-table-column>
      <el-table-column label="延迟" width="80">
        <template #default="{ row }">{{ row.latency_ms }}ms</template>
      </el-table-column>
      <el-table-column label="错误信息" min-width="160" show-overflow-tooltip>
        <template #default="{ row }"><span class="err">{{ row.error || '-' }}</span></template>
      </el-table-column>
    </el-table>
    <el-pagination class="pager" background layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" @current-change="onPage" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import http from '../api'
import { fmtTime } from '../util'

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const kw = ref('')
const modelFilter = ref('')
const statusFilter = ref('')
const range = ref(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize }
    if (kw.value) params.keyword = kw.value
    if (modelFilter.value) params.model = modelFilter.value
    if (statusFilter.value !== '' && statusFilter.value != null) params.status = statusFilter.value
    if (range.value && range.value.length === 2) {
      params.start = Math.floor(range.value[0].getTime() / 1000)
      params.end = Math.floor(range.value[1].getTime() / 1000)
    }
    const res = await http.get('/api/logs', { params })
    list.value = res.data.list
    total.value = res.data.total
  } finally { loading.value = false }
}
function onPage(p) { page.value = p; load() }

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 14px; flex-wrap: wrap; }
.pager { margin-top: 14px; justify-content: flex-end; }
.quota { color: #e6a23c; font-weight: 600; }
.err { color: #f56c6c; }
</style>