<template>
  <div>
    <div class="toolbar">
      <el-input v-model="kw" placeholder="搜索模型名/标签" clearable style="width: 220px" @keyup.enter="load" @clear="load" />
      <el-button type="primary" @click="openAdd">＋ 新增模型价格</el-button>
      <span class="muted">价格单位 = 额度 / 100 万 tokens；未定价模型按兜底价 2.0 / 8.0 计费</span>
    </div>

    <el-table :data="list" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="模型名" min-width="160" />
      <el-table-column label="输入价/1M" width="130">
        <template #default="{ row }">{{ Number(row.price_input).toFixed(4) }}</template>
      </el-table-column>
      <el-table-column label="输出价/1M" width="130">
        <template #default="{ row }">{{ Number(row.price_output).toFixed(4) }}</template>
      </el-table-column>
      <el-table-column prop="tags" label="标签" min-width="110" />
      <el-table-column label="状态" width="76">
        <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag></template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="del(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination class="pager" background layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" @current-change="onPage" />

    <el-dialog v-model="dlg.show" :title="dlg.id ? '编辑模型价格' : '新增模型价格'" width="440px">
      <el-form label-width="110px">
        <el-form-item label="模型名" required><el-input v-model="dlg.form.name" /></el-form-item>
        <el-form-item label="输入价/1M"><el-input-number v-model="dlg.form.price_input" :min="0" :precision="4" style="width: 100%" /></el-form-item>
        <el-form-item label="输出价/1M"><el-input-number v-model="dlg.form.price_output" :min="0" :precision="4" style="width: 100%" /></el-form-item>
        <el-form-item label="标签"><el-input v-model="dlg.form.tags" placeholder="如 OpenAI / DeepSeek" /></el-form-item>
        <el-form-item label="状态"><el-switch v-model="dlg.form.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg.show = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const kw = ref('')
const loading = ref(false)
const dlg = reactive({ show: false, id: 0, form: { name: '', price_input: 1, price_output: 4, status: 1, tags: '' } })

async function load() {
  loading.value = true
  try {
    const res = await http.get('/api/models', { params: { page: page.value, page_size: pageSize, keyword: kw.value } })
    list.value = res.data.list
    total.value = res.data.total
  } finally { loading.value = false }
}
function onPage(p) { page.value = p; load() }

function openAdd() {
  Object.assign(dlg, { show: true, id: 0, form: { name: '', price_input: 1, price_output: 4, status: 1, tags: '' } })
}
function openEdit(row) {
  Object.assign(dlg, { show: true, id: row.id, form: { name: row.name, price_input: row.price_input, price_output: row.price_output, status: row.status, tags: row.tags } })
}
async function save() {
  const f = dlg.form
  if (!f.name) { ElMessage.warning('请填写模型名'); return }
  try {
    if (dlg.id) { await http.put(`/api/models/${dlg.id}`, f); ElMessage.success('已保存') }
    else { await http.post('/api/models', f); ElMessage.success('已创建') }
    dlg.show = false
    load()
  } catch (e) {}
}
function del(row) {
  ElMessageBox.confirm(`确认删除模型「${row.name}」的定价？`, '提示', { type: 'warning' }).then(async () => {
    await http.delete(`/api/models/${row.id}`)
    ElMessage.success('已删除')
    load()
  }).catch(() => {})
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 12px; margin-bottom: 14px; align-items: center; }
.pager { margin-top: 14px; justify-content: flex-end; }
.muted { color: #8492a6; font-size: 12px; }
</style>