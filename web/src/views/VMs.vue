<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NDataTable, NButton, NSpace, NTag, NPopconfirm, useMessage, NInput } from 'naive-ui'
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

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const syncing = ref(false)
const vms = ref<any[]>([])

const columns = [
  { title: '名称', key: 'name' },
  { title: '状态', key: 'status', render: (row: any) => {
    const color = row.status === 'running' ? 'success' : 'default'
    return h(NTag, { type: color, size: 'small' }, () => row.status)
  }},
  { title: 'CPU', key: 'vcpu' },
  { title: '内存', key: 'memory' },
  { title: '操作', key: 'actions', render: (row: any) => h(NSpace, {}, () => [
    h(NButton, { size: 'small', onClick: () => router.push(`/vms/${row.id}`) }, () => '详情'),
    row.status !== 'running' ? h(NButton, { size: 'small', type: 'primary', onClick: () => startVM(row.id) }, () => '启动') : null,
    row.status === 'running' ? h(NButton, { size: 'small', type: 'warning', onClick: () => stopVM(row.id) }, () => '停止') : null,
    h(NPopconfirm, { onPositiveClick: () => deleteVM(row.id) }, {
      trigger: () => h(NButton, { size: 'small', type: 'error' }, () => '删除'),
      default: () => '确定要删除此虚拟机吗？'
    })
  ])}
]

async function loadVMs() {
  loading.value = true
  try {
    const res = await api.get('/vms')
    vms.value = res.data.vms || []
  } catch (e) {
    message.error('加载虚拟机列表失败')
  } finally {
    loading.value = false
  }
}

async function syncVMs() {
  syncing.value = true
  try {
    const res = await api.get('/vms/sync')
    message.success(`${res.data.message}, 新增: ${res.data.added}, 更新: ${res.data.updated}`)
    loadVMs()
  } catch (e: any) {
    message.error(e.response?.data?.error || '同步失败')
  } finally {
    syncing.value = false
  }
}

async function startVM(id: string) {
  try {
    await api.post(`/vms/${id}/start`)
    message.success('虚拟机已启动')
    loadVMs()
  } catch (e: any) {
    message.error(e.response?.data?.error || '启动失败')
  }
}

async function stopVM(id: string) {
  try {
    await api.post(`/vms/${id}/stop`)
    message.success('虚拟机已停止')
    loadVMs()
  } catch (e: any) {
    message.error(e.response?.data?.error || '停止失败')
  }
}

async function deleteVM(id: string) {
  try {
    await api.delete(`/vms/${id}`)
    message.success('虚拟机已删除')
    loadVMs()
  } catch (e: any) {
    message.error(e.response?.data?.error || '删除失败')
  }
}

onMounted(loadVMs)
</script>

<template>
  <div>
    <n-space justify="space-between" style="margin-bottom: 16px">
      <h2>虚拟机管理</h2>
      <n-space>
        <n-button @click="syncVMs" :loading="syncing">
          同步KVM
        </n-button>
        <n-button type="primary" @click="message.info('创建功能开发中')">
          创建虚拟机
        </n-button>
      </n-space>
    </n-space>
    
    <n-card>
      <n-data-table :columns="columns" :data="vms" :loading="loading" />
    </n-card>
  </div>
</template>