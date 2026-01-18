import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/services/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const token = ref(localStorage.getItem('token'))
  const refreshToken = ref(localStorage.getItem('refreshToken'))

  const isAuthenticated = computed(() => !!token.value)
  const userName = computed(() => user.value?.name || 'User')
  const userRole = computed(() => user.value?.role || 'viewer')

  async function login(email, password) {
    try {
      const response = await api.post('/auth/login', { email, password })
      const { user: userData, token: tokenData } = response.data
      
      user.value = userData
      token.value = tokenData.access_token
      refreshToken.value = tokenData.refresh_token
      
      localStorage.setItem('token', tokenData.access_token)
      localStorage.setItem('refreshToken', tokenData.refresh_token)
      
      api.defaults.headers.common['Authorization'] = `Bearer ${tokenData.access_token}`
      
      return { success: true }
    } catch (error) {
      return { 
        success: false, 
        error: error.response?.data?.error || 'Login failed' 
      }
    }
  }

  async function register(name, email, password) {
    try {
      const response = await api.post('/auth/register', { name, email, password })
      const { user: userData, token: tokenData } = response.data
      
      user.value = userData
      token.value = tokenData.access_token
      refreshToken.value = tokenData.refresh_token
      
      localStorage.setItem('token', tokenData.access_token)
      localStorage.setItem('refreshToken', tokenData.refresh_token)
      
      api.defaults.headers.common['Authorization'] = `Bearer ${tokenData.access_token}`
      
      return { success: true }
    } catch (error) {
      return { 
        success: false, 
        error: error.response?.data?.error || 'Registration failed' 
      }
    }
  }

  async function fetchUser() {
    if (!token.value) return
    
    try {
      api.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
      const response = await api.get('/auth/me')
      user.value = response.data
    } catch (error) {
      logout()
    }
  }

  async function refreshAccessToken() {
    if (!refreshToken.value) return false
    
    try {
      const response = await api.post('/auth/refresh', {
        refresh_token: refreshToken.value
      })
      
      token.value = response.data.access_token
      refreshToken.value = response.data.refresh_token
      
      localStorage.setItem('token', response.data.access_token)
      localStorage.setItem('refreshToken', response.data.refresh_token)
      
      api.defaults.headers.common['Authorization'] = `Bearer ${response.data.access_token}`
      
      return true
    } catch {
      logout()
      return false
    }
  }

  function logout() {
    user.value = null
    token.value = null
    refreshToken.value = null
    
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    
    delete api.defaults.headers.common['Authorization']
  }

  // Initialize
  if (token.value) {
    fetchUser()
  }

  return {
    user,
    token,
    isAuthenticated,
    userName,
    userRole,
    login,
    register,
    logout,
    fetchUser,
    refreshAccessToken
  }
})
