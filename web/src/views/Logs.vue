<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NCard, NDataTable, NTag, NSpace, NDatePicker, NButton, useMessage } from 'naive-ui'
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
const loading = ref(false)
const logs = ref<any[]>([])

const columns = [
  { title: '时间', key: 'created_at', width: 180 },
  { title: '用户', key: 'username', width: 100 },
  { title: '操作', key: 'action', width: 120, render: (row: any) => {
    const colors: Record<string, any> = { create: 'success', update: 'warning', delete: 'error', login: 'info' }
    return h(NTag, { type: colors[row.action] || 'default', size: 'small' }, () => row.action)
  }},
  { title: '对象', key: 'resource', width: 150 },
  { title: 'IP 地址', key: 'ip_address', width: 130 },
  { title: '结果', key: 'status', width: 80, render: (row: any) => h(NTag, { type: row.status === 'success' ? 'success' : 'error', size: 'small' }, () => row.status) },
  { title: '详情', key: 'details' }
]

async function loadLogs() {
  loading.value = true
  try {
    // TODO: 实现审计日志 API
    logs.value = []
    message.info('审计日志功能开发中')
  } catch (e) {
    message.error('加载日志失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadLogs)
</script>

<template>
  <div>
    <n-space justify="space-between" style="margin-bottom: 16px">
      <h2>审计日志</h2>
      <n-space>
        <n-button @click="loadLogs">刷新</n-button>
      </n-space>
    </n-space>
    
    <n-card>
      <n-data-table :columns="columns" :data="logs" :loading="loading" />
    </n-card>
  </div>
</template>