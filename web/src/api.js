import axios from 'axios'
import { ElMessage } from 'element-plus'
import { auth } from './store'
import router from './router'

const http = axios.create({ baseURL: '', timeout: 60000 })

http.interceptors.request.use((cfg) => {
  if (auth.token) cfg.headers.Authorization = 'Bearer ' + auth.token
  return cfg
})

http.interceptors.response.use(
  (r) => r.data,
  (err) => {
    const status = err.response?.status
    let msg = err.response?.data?.error
    if (typeof msg === 'object') msg = msg.message || JSON.stringify(msg)
    if (!msg) msg = err.message || '请求失败'
    if (status === 401) {
      auth.logout()
      if (router.currentRoute.value.path !== '/login') router.push('/login')
    }
    ElMessage.error(String(msg))
    return Promise.reject(new Error(String(msg)))
  }
)

export default http