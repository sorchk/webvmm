<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NTag, useMessage } from 'naive-ui'
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
const networks = ref<any[]>([])

const columns = [
  { title: '名称', key: 'name' },
  { title: '桥接', key: 'bridge' },
  { title: '模式', key: 'mode' },
  { title: '状态', key: 'state', render: (row: any) => {
    const color = row.state === 'active' ? 'success' : 'default'
    return h(NTag, { type: color, size: 'small' }, () => row.state)
  }},
  { title: '操作', key: 'actions', render: (row: any) => h(NSpace, {}, () => [
    h(NButton, { size: 'small', onClick: () => message.info('编辑功能开发中') }, () => '编辑')
  ])}
]

async function loadNetworks() {
  loading.value = true
  try {
    const res = await api.get('/networks')
    networks.value = res.data.networks || []
  } catch (e) {
    message.error('加载网络列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadNetworks)
</script>

<template>
  <div>
    <n-space justify="space-between" style="margin-bottom: 16px">
      <h2>网络管理</h2>
      <n-button type="primary" @click="message.info('创建功能开发中')">
        创建网络
      </n-button>
    </n-space>
    
    <n-card>
      <n-data-table :columns="columns" :data="networks" :loading="loading" />
    </n-card>
  </div>
</template>