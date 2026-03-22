<script setup lang="ts">
import { h, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NLayout, NLayoutSider, NLayoutContent, NMenu, NAvatar, NDropdown, NSpace, NIcon, useMessage } from 'naive-ui'
import { 
  DesktopOutline, 
  ServerOutline, 
  CloudOutline, 
  PeopleOutline, 
  DocumentTextOutline,
  PersonOutline,
  LogOutOutline
} from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import type { Component } from 'vue'

const router = useRouter()
const message = useMessage()
const authStore = useAuthStore()

const collapsed = ref(false)

function renderIcon(icon: Component) {
  return () => h(icon)
}

const menuOptions = [
  {
    label: '仪表盘',
    key: '/',
    icon: renderIcon(DesktopOutline)
  },
  {
    label: '虚拟机',
    key: '/vms',
    icon: renderIcon(ServerOutline)
  },
  {
    label: '存储池',
    key: '/storage',
    icon: renderIcon(CloudOutline)
  },
  {
    label: '网络',
    key: '/networks',
    icon: renderIcon(CloudOutline)
  },
  {
    label: '用户管理',
    key: '/users',
    icon: renderIcon(PeopleOutline),
    show: authStore.isAdmin
  },
  {
    label: '审计日志',
    key: '/logs',
    icon: renderIcon(DocumentTextOutline)
  }
]

const userOptions = [
  { label: '个人设置', key: 'profile' },
  { label: '退出登录', key: 'logout' }
]

function handleMenuSelect(key: string) {
  router.push(key)
}

async function handleUserSelect(key: string) {
  if (key === 'logout') {
    await authStore.logout()
    message.success('已退出登录')
    router.push('/login')
  } else if (key === 'profile') {
    router.push('/profile')
  }
}
</script>

<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="200"
      :collapsed="collapsed"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="logo">
        <span v-if="!collapsed">WebVMM</span>
        <span v-else>W</span>
      </div>
      <n-menu
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions.filter(o => o.show !== false)"
        :default-value="$route.path"
        @update:value="handleMenuSelect"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-content>
        <div class="header">
          <n-dropdown :options="userOptions" @select="handleUserSelect">
            <n-space align="center" style="cursor: pointer">
              <n-avatar round size="small">
                <n-icon :component="PersonOutline" />
              </n-avatar>
              <span>{{ authStore.user?.username }}</span>
            </n-space>
          </n-dropdown>
        </div>
        <div class="content">
          <router-view />
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<style scoped>
.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: bold;
  color: #63e2b7;
  border-bottom: 1px solid #333;
}

.header {
  height: 64px;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  border-bottom: 1px solid #333;
}

.content {
  padding: 24px;
  min-height: calc(100vh - 64px);
}
</style>