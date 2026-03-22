<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const message = useMessage()
const authStore = useAuthStore()

const loading = ref(false)
const formValue = ref({
  username: 'admin',
  email: '',
  password: '',
  password2: ''
})

async function handleSubmit() {
  if (formValue.value.password !== formValue.value.password2) {
    message.error('两次输入的密码不一致')
    return
  }
  if (formValue.value.password.length < 8) {
    message.error('密码长度至少为 8 位')
    return
  }
  
  loading.value = true
  try {
    await authStore.setup({
      username: formValue.value.username,
      email: formValue.value.email,
      password: formValue.value.password
    })
    message.success('安装完成，欢迎使用 WebVMM！')
    router.push('/')
  } catch (error: any) {
    message.error(error.response?.data?.error || '安装失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="setup-container">
    <n-card title="WebVMM 安装向导" class="setup-card">
      <p class="welcome">欢迎使用 WebVMM！请创建管理员账户完成初始化。</p>
      <n-form :model="formValue">
        <n-form-item label="管理员用户名" path="username">
          <n-input v-model:value="formValue.username" placeholder="admin" />
        </n-form-item>
        <n-form-item label="邮箱地址" path="email">
          <n-input v-model:value="formValue.email" placeholder="admin@example.com" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="formValue.password"
            type="password"
            placeholder="至少 8 位，包含大小写字母和数字"
            show-password-on="click"
          />
        </n-form-item>
        <n-form-item label="确认密码" path="password2">
          <n-input
            v-model:value="formValue.password2"
            type="password"
            placeholder="再次输入密码"
            show-password-on="click"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>
        <n-space vertical>
          <n-button type="primary" block :loading="loading" @click="handleSubmit">
            完成安装
          </n-button>
        </n-space>
      </n-form>
    </n-card>
  </div>
</template>

<style scoped>
.setup-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}
.setup-card {
  width: 450px;
}
.welcome {
  margin-bottom: 20px;
  color: #666;
}
</style>