import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/services/api'

export const useProjectsStore = defineStore('projects', () => {
  const projects = ref([])
  const currentProject = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const projectCount = computed(() => projects.value.length)

  async function fetchProjects() {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.get('/projects')
      projects.value = response.data.projects || []
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to fetch projects'
    } finally {
      loading.value = false
    }
  }

  async function fetchProject(id) {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.get(`/projects/${id}`)
      currentProject.value = response.data
      return response.data
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to fetch project'
      return null
    } finally {
      loading.value = false
    }
  }

  async function createProject(data) {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.post('/projects', data)
      projects.value.unshift(response.data)
      return { success: true, project: response.data }
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to create project'
      return { success: false, error: error.value }
    } finally {
      loading.value = false
    }
  }

  async function updateProject(id, data) {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.put(`/projects/${id}`, data)
      const index = projects.value.findIndex(p => p.id === id)
      if (index !== -1) {
        projects.value[index] = response.data
      }
      if (currentProject.value?.id === id) {
        currentProject.value = response.data
      }
      return { success: true, project: response.data }
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to update project'
      return { success: false, error: error.value }
    } finally {
      loading.value = false
    }
  }

  async function deleteProject(id) {
    loading.value = true
    error.value = null
    
    try {
      await api.delete(`/projects/${id}`)
      projects.value = projects.value.filter(p => p.id !== id)
      if (currentProject.value?.id === id) {
        currentProject.value = null
      }
      return { success: true }
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to delete project'
      return { success: false, error: error.value }
    } finally {
      loading.value = false
    }
  }

  return {
    projects,
    currentProject,
    loading,
    error,
    projectCount,
    fetchProjects,
    fetchProject,
    createProject,
    updateProject,
    deleteProject
  }
})
