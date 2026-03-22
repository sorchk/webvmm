<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NDescriptions, NDescriptionsItem, NButton, NSpace, NTag, useMessage } from 'naive-ui'
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

const route = useRoute()
const router = useRouter()
const message = useMessage()
const vm = ref<any>(null)
const loading = ref(true)

async function loadVM() {
  try {
    const res = await api.get(`/vms/${route.params.id}`)
    vm.value = res.data
  } catch (e) {
    message.error('加载虚拟机详情失败')
  } finally {
    loading.value = false
  }
}

async function startVM() {
  try {
    await api.post(`/vms/${route.params.id}/start`)
    message.success('虚拟机已启动')
    loadVM()
  } catch (e: any) {
    message.error(e.response?.data?.error || '启动失败')
  }
}

async function stopVM() {
  try {
    await api.post(`/vms/${route.params.id}/stop`)
    message.success('虚拟机已停止')
    loadVM()
  } catch (e: any) {
    message.error(e.response?.data?.error || '停止失败')
  }
}

onMounted(loadVM)
</script>

<template>
  <div>
    <n-space justify="space-between" style="margin-bottom: 16px">
      <h2>{{ vm?.name || '虚拟机详情' }}</h2>
      <n-space>
        <n-button @click="router.push('/vms')">返回列表</n-button>
        <n-button v-if="vm?.state !== 'running'" type="primary" @click="startVM">启动</n-button>
        <n-button v-if="vm?.state === 'running'" type="warning" @click="stopVM">停止</n-button>
      </n-space>
    </n-space>
    
    <n-card :loading="loading">
      <n-descriptions label-placement="left" :column="2" bordered>
        <n-descriptions-item label="名称">{{ vm?.name }}</n-descriptions-item>
        <n-descriptions-item label="状态">
          <n-tag :type="vm?.state === 'running' ? 'success' : 'default'">{{ vm?.state }}</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="CPU">{{ vm?.vcpu }} 核</n-descriptions-item>
        <n-descriptions-item label="内存">{{ vm?.memory }}</n-descriptions-item>
        <n-descriptions-item label="磁盘">{{ vm?.disk }}</n-descriptions-item>
        <n-descriptions-item label="网络">{{ vm?.network }}</n-descriptions-item>
      </n-descriptions>
    </n-card>
  </div>
</template>