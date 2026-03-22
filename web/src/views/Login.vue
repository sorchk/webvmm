<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const message = useMessage()
const authStore = useAuthStore()

const formRef = ref()
const loading = ref(false)
const formValue = ref({
  username: '',
  password: '',
  totp_code: ''
})
const needTotp = ref(false)

async function handleSubmit() {
  loading.value = true
  try {
    await authStore.login(formValue.value.username, formValue.value.password, formValue.value.totp_code || undefined)
    message.success('登录成功')
    router.push('/')
  } catch (error: any) {
    loading.value = false
    if (error.response?.data?.error?.includes('TOTP')) {
      needTotp.value = true
      message.warning('请输入双因素认证码')
    } else {
      message.error(error.response?.data?.error || '登录失败')
    }
  }
}
</script>

<template>
  <div class="login-container">
    <n-card title="WebVMM 登录" class="login-card">
      <n-form ref="formRef" :model="formValue">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="formValue.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="formValue.password"
            type="password"
            placeholder="请输入密码"
            show-password-on="click"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>
        <n-form-item v-if="needTotp" label="验证码" path="totp_code">
          <n-input
            v-model:value="formValue.totp_code"
            placeholder="请输入 6 位验证码"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>
        <n-space vertical>
          <n-button type="primary" block :loading="loading" @click="handleSubmit">
            登 录
          </n-button>
        </n-space>
      </n-form>
    </n-card>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}
.login-card {
  width: 400px;
}
</style>