<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="brand">🌉 Token 网关</div>
      <el-menu :default-active="$route.path" router class="menu">
        <el-menu-item v-for="m in menus" :key="m.path" :index="m.path">
          <span>{{ m.icon }} {{ m.title }}</span>
        </el-menu-item>
      </el-menu>
      <div class="aside-foot">v1.0 · 开源免费</div>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="crumb">{{ $route.meta.title }}</div>
        <div class="right">
          <span class="who">{{ auth.user?.display_name || auth.user?.username }}</span>
          <el-button link type="primary" @click="logout">退出</el-button>
        </div>
      </el-header>
      <el-main class="main"><router-view /></el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { auth } from '../store'

const router = useRouter()
const menus = router.options.routes.find((r) => r.path === '/').children
  .filter((r) => r.path !== '')
  .map((r) => ({ path: '/' + r.path, title: r.meta.title, icon: r.meta.icon }))

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.layout { height: 100vh; }
.aside { background: #0f172a; color: #fff; display: flex; flex-direction: column; }
.brand { height: 56px; line-height: 56px; text-align: center; font-size: 18px; font-weight: 700; color: #fff; }
.menu { border-right: none; background: transparent; --el-menu-text-color: #cbd5e1; --el-menu-hover-bg-color: #1e293b; --el-menu-active-color: #38bdf8; }
.menu :deep(.el-menu-item.is-active) { background: #1e293b; }
.aside-foot { margin-top: auto; padding: 14px; text-align: center; color: #64748b; font-size: 12px; }
.header { background: #fff; border-bottom: 1px solid #e5e7eb; display: flex; align-items: center; justify-content: space-between; }
.crumb { font-size: 16px; font-weight: 600; color: #1f2937; }
.right { display: flex; align-items: center; gap: 10px; }
.who { color: #374151; font-size: 14px; }
.main { background: #f5f7fa; }
</style>