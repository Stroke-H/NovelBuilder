import axios from 'axios'
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

const retryDelayMs = 800

function sleep(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

function isRetryableRequest(error: any) {
  const method = String(error?.config?.method || 'get').toLowerCase()
  if (!['get', 'head', 'options'].includes(method)) return false

  const status = error?.response?.status
  const message = String(error?.message || '').toLowerCase()
  const code = error?.code

  return code === 'ECONNABORTED'
    || message.includes('timeout')
    || message.includes('network error')
    || [500, 502, 503, 504].includes(status)
}

function notifyRetrying() {
  ElMessage({
    message: '网络波动，正在重新尝试...',
    type: 'warning',
    duration: 1800,
    showClose: true
  })
}

const service: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 15000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json'
  }
})

service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => config,
  (error) => Promise.reject(error)
)

service.interceptors.response.use(
  (response: AxiosResponse) => response.data,
  async (error) => {
    const config = error.config as any
    if (config && isRetryableRequest(error) && !config.__retried) {
      config.__retried = true
      notifyRetrying()
      await sleep(retryDelayMs)
      return service.request(config)
    }

    const { response, code, message } = error
    let errorMsg = '网络或服务器异常'

    if (response) {
      switch (response.status) {
        case 401:
          errorMsg = '未授权，请重新登录'
          break
        case 403:
          errorMsg = '拒绝访问：权限不足'
          break
        case 404:
          errorMsg = '请求地址错误 (404)'
          break
        case 500:
          errorMsg = response.data?.error || response.data?.message || '后端服务错误 (500)'
          break
        case 502:
        case 503:
          errorMsg = '服务暂时不可用 (502/503)'
          break
        default:
          errorMsg = response.data?.error || response.data?.message || `请求失败 (${response.status})`
      }
    } else if (code === 'ECONNABORTED' || String(message).includes('timeout')) {
      errorMsg = '服务器响应超时，请确认独立后端服务是否已启动'
    } else if (String(message).includes('Network Error')) {
      errorMsg = '无法连接到独立小说服务'
    } else {
      errorMsg = message || '发生了未知网络错误'
    }

    error.customMessage = errorMsg
    return Promise.reject(error)
  }
)

export default service
