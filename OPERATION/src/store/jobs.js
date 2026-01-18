import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/services/api'

export const useJobsStore = defineStore('jobs', () => {
  const jobs = ref([])
  const loading = ref(false)
  const error = ref(null)

  const pendingJobs = computed(() => 
    jobs.value.filter(j => j.status === 'pending' || j.status === 'queued')
  )
  
  const runningJobs = computed(() => 
    jobs.value.filter(j => j.status === 'running')
  )
  
  const completedJobs = computed(() => 
    jobs.value.filter(j => j.status === 'completed')
  )
  
  const failedJobs = computed(() => 
    jobs.value.filter(j => j.status === 'failed')
  )

  async function fetchJobs(projectId) {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.get('/jobs', {
        params: { project_id: projectId }
      })
      jobs.value = response.data.jobs || []
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to fetch jobs'
    } finally {
      loading.value = false
    }
  }

  async function fetchJob(id) {
    try {
      const response = await api.get(`/jobs/${id}`)
      const index = jobs.value.findIndex(j => j.id === id)
      if (index !== -1) {
        jobs.value[index] = response.data
      }
      return response.data
    } catch (err) {
      return null
    }
  }

  async function createJob(data) {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.post('/jobs', data)
      jobs.value.unshift(response.data)
      return { success: true, job: response.data }
    } catch (err) {
      error.value = err.response?.data?.error || 'Failed to create job'
      return { success: false, error: error.value }
    } finally {
      loading.value = false
    }
  }

  async function cancelJob(id) {
    try {
      await api.post(`/jobs/${id}/cancel`)
      const index = jobs.value.findIndex(j => j.id === id)
      if (index !== -1) {
        jobs.value[index].status = 'cancelled'
      }
      return { success: true }
    } catch (err) {
      return { success: false, error: err.response?.data?.error }
    }
  }

  function updateJobStatus(id, status, progress) {
    const index = jobs.value.findIndex(j => j.id === id)
    if (index !== -1) {
      jobs.value[index].status = status
      if (progress !== undefined) {
        jobs.value[index].progress = progress
      }
    }
  }

  return {
    jobs,
    loading,
    error,
    pendingJobs,
    runningJobs,
    completedJobs,
    failedJobs,
    fetchJobs,
    fetchJob,
    createJob,
    cancelJob,
    updateJobStatus
  }
})
