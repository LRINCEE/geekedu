import axios from 'axios'
import { getToken, logout } from '../utils/token'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

request.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      return Promise.reject(new Error(res.msg || 'Request failed'))
    }
    return res
  },
  (error) => {
    if (error.response?.status === 401) {
      logout()
      window.location.href = '/login'
    }
    const msg = error.response?.data?.msg || error.message
    return Promise.reject(new Error(msg))
  }
)

export default request
