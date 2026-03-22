<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NTag, useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  timeout: 10000
})
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

const message = useMessage()
const authStore = useAuthStore()
const loading = ref(false)
const users = ref<any[]>([])

const columns = [
  { title: '用户名', key: 'username' },
  { title: '邮箱', key: 'email' },
  { title: '角色', key: 'role', render: (row: any) => {
    const colors: Record<string, any> = { admin: 'error', auditor: 'warning', user: 'info' }
    return h(NTag, { type: colors[row.role] || 'default', size: 'small' }, () => row.role)
  }},
  { title: '2FA', key: 'totp_enabled', render: (row: any) => h(NTag, { type: row.totp_enabled ? 'success' : 'default', size: 'small' }, () => row.totp_enabled ? '已启用' : '未启用') },
  { title: '创建时间', key: 'created_at' },
  { title: '操作', key: 'actions', render: (row: any) => h(NSpace, {}, () => [
    h(NButton, { size: 'small', onClick: () => message.info('编辑功能开发中') }, () => '编辑'),
    h(NButton, { size: 'small', type: 'warning', onClick: () => message.info('重置密码功能开发中') }, () => '重置密码')
  ])}
]

async function loadUsers() {
  if (!authStore.isAdmin) {
    message.error('无权访问')
    return
  }
  loading.value = true
  try {
    const res = await api.get('/users')
    users.value = res.data.users || []
  } catch (e) {
    message.error('加载用户列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadUsers)
</script>

<template>
  <div>
    <n-space justify="space-between" style="margin-bottom: 16px">
      <h2>用户管理</h2>
      <n-button type="primary" @click="message.info('创建功能开发中')">
        创建用户
      </n-button>
    </n-space>
    
    <n-card>
      <n-data-table :columns="columns" :data="users" :loading="loading" />
    </n-card>
  </div>
</template>