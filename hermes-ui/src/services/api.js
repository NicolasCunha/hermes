import axios from 'axios'

// In production (Docker), nginx proxies /hermes to backend
// In development, vite proxy handles it
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/hermes'

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = sessionStorage.getItem('hermes_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor to handle 401 errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Only redirect to login on 401 if we're not already on the login page
    // and if we're not in the middle of a login attempt
    if (error.response?.status === 401) {
      const isLoginRequest = error.config?.url?.includes('/users/login')
      if (!isLoginRequest) {
        sessionStorage.removeItem('hermes_token')
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

export const authService = {
  login: async (subject, password) => {
    const response = await api.post('/users/login', { subject, password })
    return response.data
  },
  
  logout: () => {
    sessionStorage.removeItem('hermes_token')
  },
  
  getToken: () => {
    return sessionStorage.getItem('hermes_token')
  },
  
  setToken: (token) => {
    sessionStorage.setItem('hermes_token', token)
  }
}

export const userService = {
  getAll: async () => {
    const response = await api.get('/users')
    return response.data
  },
  
  getById: async (id) => {
    const response = await api.get(`/users/${id}`)
    return response.data
  },
  
  create: async (userData) => {
    const response = await api.post('/users/register', userData)
    return response.data
  },
  
  update: async (id, userData) => {
    const response = await api.put(`/users/${id}`, userData)
    return response.data
  },
  
  delete: async (id) => {
    const response = await api.delete(`/users/${id}`)
    return response.data
  },
  
  changePassword: async (id, oldPassword, newPassword) => {
    const response = await api.put(`/users/${id}/password`, {
      old_password: oldPassword,
      new_password: newPassword
    })
    return response.data
  },
  
  updateRoles: async (id, roles) => {
    const response = await api.put(`/users/${id}`, { roles })
    return response.data
  }
}

export const serviceService = {
  getAll: async () => {
    const response = await api.get('/services')
    return response.data
  },
  
  register: async (serviceData) => {
    const response = await api.post('/services', serviceData)
    return response.data
  },
  
  delete: async (id) => {
    const response = await api.delete(`/services/${id}`)
    return response.data
  },

  getHealthLogs: async (id, limit = 20) => {
    const response = await api.get(`/services/${id}/health-logs?limit=${limit}`)
    return response.data
  }
}

export default api
