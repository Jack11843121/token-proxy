<template>
  <div>
    <div class="toolbar">
      <el-input v-model="kw" placeholder="搜索名称/地址" clearable style="width: 220px" @keyup.enter="load" @clear="load" />
      <el-button type="primary" @click="openAdd">＋ 新增渠道</el-button>
      <span class="muted">优先级越小越先被调度，同优先级按权重从大到小；失败自动切换下一渠道</span>
    </div>

    <el-table :data="list" v-loading="loading" border stripe>
      <el-table-column prop="id" label="ID" width="64" />
      <el-table-column prop="name" label="名称" min-width="110" />
      <el-table-column label="类型" width="96">
        <template #default="{ row }"><el-tag :type="row.type === 'anthropic' ? 'warning' : 'primary'" size="small">{{ row.type === 'anthropic' ? 'Claude' : 'OpenAI兼容' }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="base_url" label="Base URL" min-width="220" show-overflow-tooltip />
      <el-table-column label="模型" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.models || '全部(*) ' }}</template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="76" />
      <el-table-column prop="weight" label="权重" width="70" />
      <el-table-column label="状态" width="76">
        <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag></template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="test(row)">测试</el-button>
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="del(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dlg.show" :title="dlg.id ? '编辑渠道' : '新增渠道'" width="560px">
      <el-form label-width="100px">
        <el-form-item label="名称"><el-input v-model="dlg.form.name" placeholder="如：DeepSeek 主渠道" /></el-form-item>
        <el-form-item label="渠道模板">
          <el-select v-model="preset" placeholder="选择厂商自动填充" clearable style="width: 100%" @change="applyPreset">
            <el-option v-for="p in presets" :key="p.name" :label="p.name" :value="p.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="dlg.form.type">
            <el-radio value="openai">OpenAI 兼容</el-radio>
            <el-radio value="anthropic">Claude(Messages)</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Base URL" required>
          <el-input v-model="dlg.form.base_url" placeholder="OpenAI兼容需含 /v1，Claude填 https://api.anthropic.com" />
        </el-form-item>
        <el-form-item label="API Key"><el-input v-model="dlg.form.key" type="password" show-password /></el-form-item>
        <el-form-item label="支持模型">
          <el-input v-model="dlg.form.models" placeholder="逗号分隔，如 gpt-4o,deepseek-chat；留空或 * = 全部" />
        </el-form-item>
        <el-form-item label="模型映射">
          <el-input v-model="dlg.form.model_mapping" type="textarea" :rows="2" placeholder='JSON，如 {"gpt-4o":"gpt-4o-2024-11-20"}' />
        </el-form-item>
        <el-form-item label="优先级"><el-input-number v-model="dlg.form.priority" :min="0" style="width: 140px" /></el-form-item>
        <el-form-item label="权重"><el-input-number v-model="dlg.form.weight" :min="1" style="width: 140px" /></el-form-item>
        <el-form-item label="状态"><el-switch v-model="dlg.form.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg.show = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="testDlg.show" title="渠道测试" width="520px">
      <p v-loading="testDlg.loading">
        <span v-if="testDlg.loading">正在测试连通性…</span>
        <span v-else-if="testDlg.success">✅ 测试通过，上游可用模型 <el-tag v-for="m in testDlg.models" :key="m" size="small" style="margin:2px">{{ m }}</el-tag></span>
        <el-alert v-else type="error" :title="testDlg.error" :closable="false" />
      </p>
      <template #footer><el-button @click="testDlg.show = false">关闭</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const presets = [
  { name: 'OpenAI', type: 'openai', base_url: 'https://api.openai.com/v1' },
  { name: 'DeepSeek', type: 'openai', base_url: 'https://api.deepseek.com/v1' },
  { name: '通义千问', type: 'openai', base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1' },
  { name: 'Kimi(月之暗面)', type: 'openai', base_url: 'https://api.moonshot.cn/v1' },
  { name: '智谱 GLM', type: 'openai', base_url: 'https://open.bigmodel.cn/api/paas/v4' },
  { name: '豆包(火山方舟)', type: 'openai', base_url: 'https://ark.cn-beijing.volces.com/api/v3' },
  { name: 'Gemini(OpenAI兼容)', type: 'openai', base_url: 'https://generativelanguage.googleapis.com/v1beta/openai' },
  { name: 'Claude(Anthropic)', type: 'anthropic', base_url: 'https://api.anthropic.com' },
  { name: '本地 Ollama/vLLM', type: 'openai', base_url: 'http://127.0.0.1:11434/v1' },
]

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const kw = ref('')
const loading = ref(false)
const preset = ref('')

const dlg = reactive({ show: false, id: 0, form: { name: '', type: 'openai', base_url: '', key: '', models: '', model_mapping: '', priority: 0, weight: 1, status: 1, groups: '' } })
const testDlg = reactive({ show: false, loading: false, success: false, models: [], error: '' })

function applyPreset(name) {
  const p = presets.find((x) => x.name === name)
  if (!p) return
  dlg.form.type = p.type
  dlg.form.base_url = p.base_url
}

async function load() {
  loading.value = true
  try {
    const res = await http.get('/api/channels', { params: { page: page.value, page_size: pageSize, keyword: kw.value } })
    list.value = res.data.list
    total.value = res.data.total
  } finally { loading.value = false }
}

function openAdd() {
  Object.assign(dlg, { show: true, id: 0, form: { name: '', type: 'openai', base_url: '', key: '', models: '', model_mapping: '', priority: 0, weight: 1, status: 1, groups: '' } })
  preset.value = ''
}
function openEdit(row) {
  Object.assign(dlg, { show: true, id: row.id, form: { name: row.name, type: row.type, base_url: row.base_url, key: row.key, models: row.models, model_mapping: row.model_mapping, priority: row.priority, weight: row.weight, status: row.status, groups: row.groups } })
  preset.value = ''
}
async function save() {
  const f = { ...dlg.form }
  if (!f.base_url) { ElMessage.warning('请填写 Base URL'); return }
  if (f.model_mapping) {
    try { JSON.parse(f.model_mapping) } catch (e) { ElMessage.warning('模型映射不是合法 JSON'); return }
  }
  try {
    if (dlg.id) { await http.put(`/api/channels/${dlg.id}`, f); ElMessage.success('已保存') }
    else { await http.post('/api/channels', f); ElMessage.success('已创建') }
    dlg.show = false
    load()
  } catch (e) {}
}
function del(row) {
  ElMessageBox.confirm(`确认删除渠道「${row.name || row.base_url}」？`, '提示', { type: 'warning' }).then(async () => {
    await http.delete(`/api/channels/${row.id}`)
    ElMessage.success('已删除')
    load()
  }).catch(() => {})
}
async function test(row) {
  Object.assign(testDlg, { show: true, loading: true, success: false, models: [], error: '' })
  try {
    const res = await http.post(`/api/channels/${row.id}/test`)
    testDlg.loading = false
    testDlg.success = res.data.success
    testDlg.models = res.data.models || []
    if (!res.data.success) testDlg.error = res.data.error || '测试失败'
  } catch (e) {
    testDlg.loading = false
    testDlg.success = false
    testDlg.error = e.message
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 12px; margin-bottom: 14px; align-items: center; }
.muted { color: #8492a6; font-size: 12px; }
</style>