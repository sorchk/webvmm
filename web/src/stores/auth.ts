import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

// 直接连接后端，绕过 Vite 代理
const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  timeout: 10000
})

// 请求拦截器添加 token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export interface User {
  id: number
  username: string
  email: string
  role: string
  totp_enabled: boolean
  created_at: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(localStorage.getItem('token'))
  const isSetupComplete = ref<boolean | null>(null)

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isAuditor = computed(() => user.value?.role === 'auditor')

  async function checkSetup(): Promise<boolean> {
    if (isSetupComplete.value !== null) return isSetupComplete.value
    try {
      const res = await api.get('/setup')
      isSetupComplete.value = res.data.initialized
      return res.data.initialized
    } catch {
      return false
    }
  }

  async function setup(data: { username: string; email: string; password: string }): Promise<void> {
    const res = await api.post('/setup', data)
    const accessToken = res.data.access_token || res.data.token
    token.value = accessToken
    localStorage.setItem('token', accessToken)
    user.value = res.data.user
    isSetupComplete.value = true
  }

  async function login(username: string, password: string, totpCode?: string): Promise<void> {
    const res = await api.post('/auth/login', { username, password, totp_code: totpCode })
    const accessToken = res.data.access_token || res.data.token
    token.value = accessToken
    localStorage.setItem('token', accessToken)
    if (res.data.user) {
      user.value = res.data.user
    }
  }

  async function logout(): Promise<void> {
    try {
      await api.post('/auth/logout')
    } catch {}
    token.value = null
    user.value = null
    localStorage.removeItem('token')
  }

  async function fetchProfile(): Promise<void> {
    if (!token.value) return
    try {
      const res = await api.get('/profile')
      user.value = res.data
    } catch (error: any) {
      // 只在认证错误时清除 token，其他错误保持登录状态
      if (error.response?.status === 401 || error.response?.status === 403) {
        token.value = null
        localStorage.removeItem('token')
      }
    }
  }

  async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
    await api.put('/profile/password', { old_password: oldPassword, new_password: newPassword })
  }

  return {
    user,
    token,
    isSetupComplete,
    isAuthenticated,
    isAdmin,
    isAuditor,
    checkSetup,
    setup,
    login,
    logout,
    fetchProfile,
    changePassword
  }
})