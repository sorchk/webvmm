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
const pools = ref<any[]>([])

const columns = [
  { title: '名称', key: 'name' },
  { title: '类型', key: 'type' },
  { title: '状态', key: 'state', render: (row: any) => {
    const color = row.state === 'running' ? 'success' : 'default'
    return h(NTag, { type: color, size: 'small' }, () => row.state)
  }},
  { title: '容量', key: 'capacity' },
  { title: '可用', key: 'available' },
  { title: '操作', key: 'actions', render: (row: any) => h(NSpace, {}, () => [
    h(NButton, { size: 'small', onClick: () => message.info('查看卷功能开发中') }, () => '查看卷')
  ])}
]

async function loadPools() {
  loading.value = true
  try {
    const res = await api.get('/storage/pools')
    pools.value = res.data.pools || []
  } catch (e) {
    message.error('加载存储池列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadPools)
</script>

<template>
  <div>
    <n-space justify="space-between" style="margin-bottom: 16px">
      <h2>存储池管理</h2>
      <n-button type="primary" @click="message.info('创建功能开发中')">
        创建存储池
      </n-button>
    </n-space>
    
    <n-card>
      <n-data-table :columns="columns" :data="pools" :loading="loading" />
    </n-card>
  </div>
</template>