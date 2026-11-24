<template>
  <div class="login-wrap">
    <div class="login-card">
      <h1>🌉 Token 网关</h1>
      <p class="sub">多模型大模型中转站 · 管理后台</p>
      <el-form @submit.prevent="submit">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" size="large" show-password @keyup.enter="submit" />
        </el-form-item>
        <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="submit">登 录</el-button>
      </el-form>
      <p class="hint">默认账号 root / 123456（生产部署后请及时修改）</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api'
import { auth } from '../store'

const router = useRouter()
const loading = ref(false)
const form = reactive({ username: '', password: '' })

async function submit() {
  if (!form.username || !form.password) return
  loading.value = true
  try {
    const res = await http.post('/api/auth/login', form)
    auth.setAuth(res.data.token, res.data.user)
    router.push('/')
  } catch (e) {
    /* 已在拦截器提示 */
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1e3a8a, #0f172a);
}
.login-card {
  width: 360px;
  padding: 40px 36px;
  background: #fff;
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}
h1 { margin: 0 0 4px; font-size: 26px; color: #1e3a8a; text-align: center; }
.sub { margin: 0 0 24px; text-align: center; color: #8492a6; font-size: 13px; }
.hint { margin: 16px 0 0; text-align: center; color: #c0c4cc; font-size: 12px; }
</style>