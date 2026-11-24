<template>
  <div>
    <div class="toolbar">
      <el-input v-model="kw" placeholder="搜索用户名/昵称" clearable style="width: 240px" @keyup.enter="load" @clear="load" />
      <el-button type="primary" @click="openAdd">＋ 新增用户</el-button>
    </div>

    <el-table :data="list" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="username" label="用户名" min-width="110" />
      <el-table-column prop="display_name" label="昵称" min-width="110" />
      <el-table-column prop="role" label="角色" width="80" />
      <el-table-column label="状态" width="76">
        <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag></template>
      </el-table-column>
      <el-table-column label="剩余额度" width="120">
        <template #default="{ row }"><span class="quota">{{ Number(row.quota).toFixed(4) }}</span></template>
      </el-table-column>
      <el-table-column label="已用额度" width="120">
        <template #default="{ row }">{{ Number(row.used_quota).toFixed(4) }}</template>
      </el-table-column>
      <el-table-column prop="request_count" label="请求数" width="80" />
      <el-table-column label="创建时间" width="160">
        <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="210" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openCharge(row)">充值</el-button>
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="del(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination class="pager" background layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" @current-change="onPage" />

    <!-- 新增/编辑 -->
    <el-dialog v-model="dlg.show" :title="dlg.id ? '编辑用户' : '新增用户'" width="460px">
      <el-form label-width="90px">
        <el-form-item label="用户名">
          <el-input v-model="dlg.form.username" :disabled="!!dlg.id" />
        </el-form-item>
        <el-form-item label="昵称"><el-input v-model="dlg.form.display_name" /></el-form-item>
        <el-form-item :label="dlg.id ? '重置密码' : '密码'">
          <el-input v-model="dlg.form.password" :placeholder="dlg.id ? '留空则不修改' : '必填'" />
        </el-form-item>
        <el-form-item label="额度"><el-input-number v-model="dlg.form.quota" :min="0" :precision="4" style="width: 100%" /></el-form-item>
        <el-form-item label="状态"><el-switch v-model="dlg.form.status" :active-value="1" :inactive-value="0" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="dlg.form.note" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg.show = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <!-- 充值 -->
    <el-dialog v-model="charge.show" title="额度充值" width="380px">
      <p>用户：<b>{{ charge.name }}</b>，当前额度：<b>{{ charge.cur }}</b></p>
      <el-input-number v-model="charge.amount" :precision="4" style="width: 100%" />
      <p class="muted">正数=充值，负数=扣减（结果不低于 0）</p>
      <template #footer>
        <el-button @click="charge.show = false">取消</el-button>
        <el-button type="primary" @click="doCharge">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'
import { fmtTime } from '../util'

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const kw = ref('')
const loading = ref(false)

const dlg = reactive({ show: false, id: 0, form: { username: '', display_name: '', password: '', quota: 0, status: 1, note: '' } })
const charge = reactive({ show: false, id: 0, name: '', cur: 0, amount: 10 })

async function load() {
  loading.value = true
  try {
    const res = await http.get('/api/users', { params: { page: page.value, page_size: pageSize, keyword: kw.value } })
    list.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}
function onPage(p) { page.value = p; load() }

function openAdd() {
  Object.assign(dlg, { show: true, id: 0, form: { username: '', display_name: '', password: '', quota: 0, status: 1, note: '' } })
}
function openEdit(row) {
  Object.assign(dlg, { show: true, id: row.id, form: { username: row.username, display_name: row.display_name, password: '', quota: row.quota, status: row.status, note: row.note } })
}
async function save() {
  const f = dlg.form
  if (!f.username || (!dlg.id && !f.password)) { ElMessage.warning('请填写用户名和密码'); return }
  try {
    if (dlg.id) await http.put(`/api/users/${dlg.id}`, f)
    else await http.post('/api/users', f)
    ElMessage.success('已保存')
    dlg.show = false
    load()
  } catch (e) {}
}
function del(row) {
  ElMessageBox.confirm(`确认删除用户「${row.username}」及其所有令牌？`, '提示', { type: 'warning' }).then(async () => {
    await http.delete(`/api/users/${row.id}`)
    ElMessage.success('已删除')
    load()
  }).catch(() => {})
}
function openCharge(row) {
  Object.assign(charge, { show: true, id: row.id, name: row.display_name || row.username, cur: Number(row.quota).toFixed(4), amount: 10 })
}
async function doCharge() {
  try {
    const res = await http.post(`/api/users/${charge.id}/quota`, { amount: charge.amount })
    ElMessage.success(`充值成功，当前额度 ${Number(res.data.quota).toFixed(4)}`)
    charge.show = false
    load()
  } catch (e) {}
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 12px; margin-bottom: 14px; }
.pager { margin-top: 14px; justify-content: flex-end; }
.quota { color: #e6a23c; font-weight: 600; }
.muted { color: #8492a6; font-size: 12px; }
</style>