import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/setup',
      name: 'Setup',
      component: () => import('@/views/Setup.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/views/Dashboard.vue')
        },
        {
          path: 'vms',
          name: 'VMs',
          component: () => import('@/views/VMs.vue')
        },
        {
          path: 'vms/:id',
          name: 'VMDetail',
          component: () => import('@/views/VMDetail.vue')
        },
        {
          path: 'storage',
          name: 'Storage',
          component: () => import('@/views/Storage.vue')
        },
        {
          path: 'networks',
          name: 'Networks',
          component: () => import('@/views/Networks.vue')
        },
        {
          path: 'users',
          name: 'Users',
          component: () => import('@/views/Users.vue')
        },
        {
          path: 'profile',
          name: 'Profile',
          component: () => import('@/views/Profile.vue')
        },
        {
          path: 'logs',
          name: 'Logs',
          component: () => import('@/views/Logs.vue')
        }
      ]
    }
  ]
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // 公开页面直接放行
  if (to.meta.requiresAuth === false) {
    // 如果已认证且访问登录页，重定向到首页
    if (to.path === '/login' && authStore.isAuthenticated) {
      next('/')
      return
    }
    next()
    return
  }

  // 检查认证状态（基于 token）
  if (!authStore.isAuthenticated) {
    next('/login')
    return
  }

  // 已认证用户，直接放行
  next()
})

export default router