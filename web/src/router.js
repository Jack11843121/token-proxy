import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/login', component: () => import('./views/Login.vue'), meta: { title: '登录' } },
  {
    path: '/',
    component: () => import('./views/Layout.vue'),
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', component: () => import('./views/Dashboard.vue'), meta: { title: '概览', icon: '📊' } },
      { path: 'users', component: () => import('./views/Users.vue'), meta: { title: '用户管理', icon: '👤' } },
      { path: 'tokens', component: () => import('./views/Tokens.vue'), meta: { title: '令牌管理', icon: '🔑' } },
      { path: 'channels', component: () => import('./views/Channels.vue'), meta: { title: '渠道管理', icon: '🔌' } },
      { path: 'models', component: () => import('./views/Models.vue'), meta: { title: '模型价格', icon: '💰' } },
      { path: 'logs', component: () => import('./views/Logs.vue'), meta: { title: '调用日志', icon: '📝' } },
      { path: 'docs', component: () => import('./views/Docs.vue'), meta: { title: 'API 文档', icon: '📖' } },
    ],
  },
]

const router = createRouter({ history: createWebHashHistory(), routes })

router.beforeEach((to) => {
  const token = localStorage.getItem('tp_token')
  if (to.path !== '/login' && !token) return '/login'
  if (to.path === '/login' && token) return '/'
  return true
})

export default router