import { reactive } from 'vue'

export const auth = reactive({
  token: localStorage.getItem('tp_token') || '',
  user: JSON.parse(localStorage.getItem('tp_user') || 'null'),
  setAuth(token, user) {
    this.token = token
    this.user = user
    localStorage.setItem('tp_token', token)
    localStorage.setItem('tp_user', JSON.stringify(user))
  },
  logout() {
    this.token = ''
    this.user = null
    localStorage.removeItem('tp_token')
    localStorage.removeItem('tp_user')
  },
})