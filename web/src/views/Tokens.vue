<template>
  <div>
    <div class="toolbar">
      <el-input v-model="kw" placeholder="搜索令牌名称" clearable style="width: 220px" @keyup.enter="load" @clear="load" />
      <el-select v-model="uidFilter" placeholder="按用户筛选" clearable style="width: 180px" @change="load">
        <el-option v-for="u in users" :key="u.id" :label="`${u.display_name || u.username} (#${u.id})`" :value="u.id" />
      </el-select>
      <el-button type="primary" @click="openAdd">＋ 新增令牌</el-button>
    </div>

    <el-table :data="list" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="名称" min-width="110" />
      <el-table-column label="API Key" min-width="280">
        <template #default="{ row }">
          <code class="key">{{ row.key }}</code>
          <el-button link type="primary" @click="copyKey(row.key)">复制</el-button>
        </template>
      </el-table-column>
      <el-table-column label="用户" width="110">
        <template #default="{ row }">{{ userName(row.user_id) }}</template>
      </el-table-column>
      <el-table-column label="额度" width="120">
        <template #default="{ row }">{{ row.unlimited_quota === 1 ? '不限' : (row.quota < 0 ? '跟随用户' : Number(row.quota).toFixed(2)) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="76">
        <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="request_count" label="请求数" width="80" />
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="del(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination class="pager" background layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" @current-change="onPage" />

    <el-dialog v-model="dlg.show" :title="dlg.id ? '编辑令牌' : '新增令牌'" width="520px">
      <el-form label-width="110px">
        <el-form-item label="所属用户" required>
          <el-select v-model="dlg.form.user_id" filterable style="width: 100%">
            <el-option v-for="u in users" :key="u.id" :label="`${u.display_name || u.username} (#${u.id})`" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称"><el-input v-model="dlg.form.name" placeholder="如：生产环境" /></el-form-item>
        <el-form-item label="额度方式">
          <el-radio-group v-model="dlg.quotaMode">
            <el-radio value="follow">跟随用户</el-radio>
            <el-radio value="fixed">独立额度</el-radio>
            <el-radio value="unlimited">不限额度</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="dlg.quotaMode === 'fixed'" label="独立额度">
          <el-input-number v-model="dlg.form.quota" :min="0" :precision="4" style="width: 100%" />
        </el-form-item>
        <el-form-item label="受限模型">
          <el-input v-model="dlg.form.model_scope" placeholder="逗号分隔，留空=全部模型" />
        </el-form-item>
        <el-form-item label="有效期">
          <el-switch v-model="dlg.forever" active-text="永不过期" style="margin-right: 12px" />
          <el-date-picker v-if="!dlg.forever" v-model="dlg.expiry" type="datetime" placeholder="选择过期时间" />
        </el-form-item>
        <el-form-item label="状态"><el-switch v-model="dlg.form.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg.show = false">取消</el-button>
        <el-button type="primary" @click="save">{{ dlg.id ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'
import { copy } from '../util'

const list = ref([])
const users = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const kw = ref('')
const uidFilter = ref('')
const loading = ref(false)

const dlg = reactive({ show: false, id: 0, quotaMode: 'follow', forever: true, expiry: null, form: { user_id: null, name: '', quota: -1, unlimited_quota: 0, model_scope: '', status: 1 } })

function userName(id) {
  const u = users.value.find((x) => x.id === id)
  return u ? (u.display_name || u.username) : `#${id}`
}

async function loadUsers() {
  try {
    const res = await http.get('/api/users', { params: { page: 1, page_size: 500 } })
    users.value = res.data.list
  } catch (e) {}
}
async function load() {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize, keyword: kw.value }
    if (uidFilter.value) params.user_id = uidFilter.value
    const res = await http.get('/api/tokens', { params })
    list.value = res.data.list
    total.value = res.data.total
  } finally { loading.value = false }
}
function onPage(p) { page.value = p; load() }

function openAdd() {
  Object.assign(dlg, { show: true, id: 0, quotaMode: 'follow', forever: true, expiry: null, form: { user_id: users.value[0]?.id ?? null, name: '', quota: -1, unlimited_quota: 0, model_scope: '', status: 1 } })
}
function openEdit(row) {
  let quotaMode = 'follow'
  if (row.unlimited_quota === 1) quotaMode = 'unlimited'
  else if (row.quota >= 0) quotaMode = 'fixed'
  Object.assign(dlg, {
    show: true, id: row.id, quotaMode, forever: !row.expired_time,
    expiry: row.expired_time ? new Date(row.expired_time * 1000) : null,
    form: { user_id: row.user_id, name: row.name, quota: row.quota, unlimited_quota: row.unlimited_quota, model_scope: row.model_scope, status: row.status },
  })
}
async function save() {
  const f = { ...dlg.form }
  if (!f.user_id) { ElMessage.warning('请选择所属用户'); return }
  if (dlg.quotaMode === 'unlimited') { f.unlimited_quota = 1; f.quota = -1 }
  else if (dlg.quotaMode === 'fixed') { f.unlimited_quota = 0 }
  else { f.unlimited_quota = 0; f.quota = -1 }
  f.expired_time = dlg.forever ? 0 : Math.floor((dlg.expiry ? dlg.expiry.getTime() : 0) / 1000)
  try {
    if (dlg.id) { await http.put(`/api/tokens/${dlg.id}`, f); ElMessage.success('已保存') }
    else {
      const res = await http.post('/api/tokens', f)
      ElMessage.success('创建成功，请复制密钥')
      await copy(res.data.key)
      ElMessageBox.alert(`API Key（已复制到剪贴板）：\n\n${res.data.key}`, '令牌创建成功', { confirmButtonText: '知道了' })
    }
    dlg.show = false
    load()
  } catch (e) {}
}
function del(row) {
  ElMessageBox.confirm(`确认删除令牌「${row.name || row.key.slice(0, 12)}」？`, '提示', { type: 'warning' }).then(async () => {
    await http.delete(`/api/tokens/${row.id}`)
    ElMessage.success('已删除')
    load()
  }).catch(() => {})
}
async function copyKey(key) {
  if (await copy(key)) ElMessage.success('已复制')
  else ElMessage.warning('复制失败，请手动复制')
}

onMounted(() => { loadUsers(); load() })
</script>

<style scoped>
.toolbar { display: flex; gap: 12px; margin-bottom: 14px; }
.pager { margin-top: 14px; justify-content: flex-end; }
.key { font-family: monospace; font-size: 12px; color: #334155; word-break: break-all; }
</style>