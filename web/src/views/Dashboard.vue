<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NGrid, NGi, NStatistic, NSpin, NProgress, NTag } from 'naive-ui'
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

const loading = ref(true)
const stats = ref({
  vms: { total: 0, running: 0, stopped: 0 },
  storage: { pools: 0, capacity: 0, used: 0 },
  networks: 0,
  host: { cpu: 0, memory: 0, disk: 0 }
})

onMounted(async () => {
  try {
    const [vmRes, poolRes, netRes] = await Promise.all([
      api.get('/vms').catch(() => ({ data: { vms: [] } })),
      api.get('/storage/pools').catch(() => ({ data: { pools: [] } })),
      api.get('/networks').catch(() => ({ data: { networks: [] } }))
    ])
    
    const vms = vmRes.data.vms || []
    stats.value.vms.total = vms.length
    stats.value.vms.running = vms.filter((v: any) => v.state === 'running').length
    stats.value.vms.stopped = vms.filter((v: any) => v.state !== 'running').length
    
    stats.value.storage.pools = poolRes.data.pools?.length || 0
    stats.value.networks = netRes.data.networks?.length || 0
  } catch (e) {
    console.error('Failed to load stats:', e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <h2 style="margin-bottom: 24px">仪表盘</h2>
    
    <n-spin :show="loading">
      <n-grid :cols="4" :x-gap="16" :y-gap="16">
        <n-gi>
          <n-card>
            <n-statistic label="虚拟机总数" :value="stats.vms.total">
              <template #suffix>
                <n-tag type="success" size="small" style="margin-left: 8px">
                  {{ stats.vms.running }} 运行中
                </n-tag>
              </template>
            </n-statistic>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card>
            <n-statistic label="存储池" :value="stats.storage.pools" />
          </n-card>
        </n-gi>
        <n-gi>
          <n-card>
            <n-statistic label="虚拟网络" :value="stats.networks" />
          </n-card>
        </n-gi>
        <n-gi>
          <n-card>
            <n-statistic label="系统状态">
              <n-tag type="success">正常</n-tag>
            </n-statistic>
          </n-card>
        </n-gi>
      </n-grid>

      <n-grid :cols="2" :x-gap="16" :y-gap="16" style="margin-top: 16px">
        <n-gi>
          <n-card title="虚拟机状态分布">
            <n-grid :cols="2" :x-gap="16">
              <n-gi>
                <n-statistic label="运行中" :value="stats.vms.running">
                  <template #prefix>
                    <span style="color: #63e2b7">●</span>
                  </template>
                </n-statistic>
              </n-gi>
              <n-gi>
                <n-statistic label="已停止" :value="stats.vms.stopped">
                  <template #prefix>
                    <span style="color: #909399">●</span>
                  </template>
                </n-statistic>
              </n-gi>
            </n-grid>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card title="快速操作">
            <p>使用左侧菜单管理虚拟机、存储池和网络。</p>
          </n-card>
        </n-gi>
      </n-grid>
    </n-spin>
  </div>
</template>