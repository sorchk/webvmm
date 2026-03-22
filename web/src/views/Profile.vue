<script setup lang="ts">
import { ref } from 'vue'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'

const message = useMessage()
const authStore = useAuthStore()

const loading = ref(false)
const formValue = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

async function handleChangePassword() {
  if (formValue.value.new_password !== formValue.value.confirm_password) {
    message.error('两次输入的密码不一致')
    return
  }
  if (formValue.value.new_password.length < 8) {
    message.error('密码长度至少为 8 位')
    return
  }
  
  loading.value = true
  try {
    await authStore.changePassword(formValue.value.old_password, formValue.value.new_password)
    message.success('密码修改成功')
    formValue.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (error: any) {
    message.error(error.response?.data?.error || '密码修改失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h2 style="margin-bottom: 24px">个人设置</h2>
    
    <n-card title="修改密码" style="max-width: 500px">
      <n-form :model="formValue">
        <n-form-item label="当前密码" path="old_password">
          <n-input
            v-model:value="formValue.old_password"
            type="password"
            placeholder="请输入当前密码"
            show-password-on="click"
          />
        </n-form-item>
        <n-form-item label="新密码" path="new_password">
          <n-input
            v-model:value="formValue.new_password"
            type="password"
            placeholder="请输入新密码"
            show-password-on="click"
          />
        </n-form-item>
        <n-form-item label="确认密码" path="confirm_password">
          <n-input
            v-model:value="formValue.confirm_password"
            type="password"
            placeholder="再次输入新密码"
            show-password-on="click"
          />
        </n-form-item>
        <n-space>
          <n-button type="primary" :loading="loading" @click="handleChangePassword">
            修改密码
          </n-button>
        </n-space>
      </n-form>
    </n-card>
  </div>
</template>