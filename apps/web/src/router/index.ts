import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const authDisabled = import.meta.env.VITE_NOVEL_GENERATER_AUTH_DISABLED !== '0'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login/index.vue'),
    meta: { public: true, title: '登录' }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/Register/index.vue'),
    meta: { public: true, title: '注册' }
  },
  {
    path: '/',
    name: 'NovelWorkspace',
    component: () => import('@/views/NovelWriter/index.vue'),
    meta: { title: '小说工作台' }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

let authBootstrapped = false

router.beforeEach(async (to) => {
  if (authDisabled) {
    if (to.path === '/login' || to.path === '/register') return '/'
    return true
  }

  const authStore = useAuthStore()
  const isPublic = Boolean(to.meta.public)

  if (!authBootstrapped) {
    authBootstrapped = true
    await authStore.fetchMe()
  }

  if (isPublic && authStore.isLoggedIn) {
    return '/'
  }

  if (!isPublic && !authStore.isLoggedIn) {
    return '/login'
  }

  return true
})

export default router
