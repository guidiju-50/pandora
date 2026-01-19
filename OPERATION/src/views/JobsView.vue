<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useJobsStore } from '@/store/jobs'
import { getAllJobs } from '@/services/processing'

const jobsStore = useJobsStore()

const filter = ref('all')
const processingJobs = ref([])
const loadingProcessingJobs = ref(false)
let pollInterval = null

// Combine jobs from both sources
const allJobs = computed(() => {
  // Map processing jobs to consistent format
  const mappedProcessingJobs = processingJobs.value.map(job => ({
    ...job,
    source: 'processing',
    type: job.type || 'download',
    created_at: job.created_at,
    started_at: job.started_at,
    completed_at: job.completed_at
  }))
  
  // Add source to control jobs
  const controlJobs = jobsStore.jobs.map(job => ({
    ...job,
    source: 'control'
  }))
  
  return [...mappedProcessingJobs, ...controlJobs].sort((a, b) => 
    new Date(b.created_at) - new Date(a.created_at)
  )
})

const filteredJobs = computed(() => {
  if (filter.value === 'all') return allJobs.value
  return allJobs.value.filter(job => job.status === filter.value)
})

const statusCounts = computed(() => ({
  all: allJobs.value.length,
  running: allJobs.value.filter(j => j.status === 'running').length,
  completed: allJobs.value.filter(j => j.status === 'completed').length,
  failed: allJobs.value.filter(j => j.status === 'failed').length
}))

// Fetch processing jobs
async function fetchProcessingJobs() {
  try {
    const data = await getAllJobs()
    processingJobs.value = data.jobs || []
  } catch (e) {
    console.error('Failed to fetch processing jobs:', e)
  }
}

onMounted(() => {
  fetchProcessingJobs()
  // Poll every 3 seconds
  pollInterval = setInterval(fetchProcessingJobs, 3000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})

function getStatusClass(status) {
  const classes = {
    pending: 'badge--info',
    queued: 'badge--info',
    running: 'badge--warning',
    completed: 'badge--success',
    failed: 'badge--danger',
    cancelled: 'badge--info'
  }
  return classes[status] || 'badge--info'
}

function formatDate(date) {
  return new Date(date).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<template>
  <div class="jobs-page">
    <div class="page-header">
      <div>
        <h2>Jobs</h2>
        <p>Monitor your processing and analysis jobs</p>
      </div>
    </div>

    <div class="filter-tabs">
      <button
        v-for="(count, key) in statusCounts"
        :key="key"
        class="filter-tab"
        :class="{ active: filter === key }"
        @click="filter = key"
      >
        {{ key.charAt(0).toUpperCase() + key.slice(1) }}
        <span class="count">{{ count }}</span>
      </button>
    </div>

    <div v-if="filteredJobs.length === 0" class="empty-state">
      <div class="empty-icon">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect>
          <rect x="9" y="9" width="6" height="6"></rect>
        </svg>
      </div>
      <h3>No jobs found</h3>
      <p>Jobs will appear here when you run analyses</p>
    </div>

    <div v-else class="jobs-list">
      <div v-for="job in filteredJobs" :key="job.id" class="job-card">
        <div class="job-main">
          <div class="job-type">
            <svg v-if="job.type === 'scrape'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="2" y1="12" x2="22" y2="12"></line>
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>
            </svg>
            <svg v-else-if="job.type === 'download' || job.type === 'full-pipeline'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
              <polyline points="7 10 12 15 17 10"></polyline>
              <line x1="12" y1="15" x2="12" y2="3"></line>
            </svg>
            <svg v-else-if="job.type === 'process'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6"></polyline>
              <polyline points="8 6 2 12 8 18"></polyline>
            </svg>
            <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="20" x2="18" y2="10"></line>
              <line x1="12" y1="20" x2="12" y2="4"></line>
              <line x1="6" y1="20" x2="6" y2="14"></line>
            </svg>
          </div>
          <div class="job-info">
            <div class="job-title">
              {{ job.type === 'full-pipeline' ? 'Download & Process' : job.type }}
              <span v-if="job.input?.accession" class="job-accession">{{ job.input.accession }}</span>
            </div>
            <div class="job-id">{{ job.id?.substring(0, 8) }}...</div>
          </div>
        </div>
        
        <div class="job-meta">
          <span class="badge" :class="getStatusClass(job.status)">{{ job.status }}</span>
          <span v-if="job.source === 'processing'" class="source-badge">PROCESSING</span>
          <span class="job-date">{{ formatDate(job.created_at) }}</span>
        </div>

        <div v-if="job.status === 'running' || job.status === 'pending'" class="job-progress">
          <div class="progress-bar">
            <div 
              class="progress-fill" 
              :class="{ 'progress-animated': job.status === 'running' }"
              :style="{ width: (job.progress || 0) + '%' }"
            ></div>
          </div>
          <span class="progress-text">{{ job.progress || 0 }}%</span>
        </div>
        
        <div v-if="job.message" class="job-message">{{ job.message }}</div>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.jobs-page {
  animation: fadeIn 0.3s ease-out;
}

.page-header {
  margin-bottom: 2rem;
  
  h2 {
    margin-bottom: 0.25rem;
  }
  
  p {
    color: var(--text-muted);
  }
}

.filter-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.filter-tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: none;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: 0.875rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  
  &:hover {
    background: var(--bg-tertiary);
  }
  
  &.active {
    background: rgba(6, 182, 212, 0.1);
    border-color: var(--accent-primary);
    color: var(--accent-primary);
  }
  
  .count {
    padding: 0.125rem 0.5rem;
    background: var(--bg-tertiary);
    border-radius: 9999px;
    font-size: 0.75rem;
  }
  
  &.active .count {
    background: var(--accent-primary);
    color: white;
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem;
  text-align: center;
  
  .empty-icon {
    color: var(--text-muted);
    margin-bottom: 1.5rem;
  }
  
  h3 {
    margin-bottom: 0.5rem;
  }
  
  p {
    color: var(--text-muted);
  }
}

.jobs-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.job-card {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  transition: all var(--transition-fast);
  
  &:hover {
    border-color: var(--border-glow);
  }
}

.job-main {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex: 1;
  min-width: 200px;
}

.job-type {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  color: var(--accent-primary);
}

.job-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.job-title {
  font-weight: 500;
  text-transform: capitalize;
  color: var(--text-primary);
}

.job-id {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-muted);
}

.job-meta {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.job-date {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.job-progress {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-color);
}

.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--gradient-primary);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--accent-primary);
  min-width: 40px;
}

.progress-animated {
  background: linear-gradient(
    90deg,
    var(--accent-primary),
    #22d3ee,
    var(--accent-primary)
  );
  background-size: 200% 100%;
  animation: progressShimmer 1.5s ease infinite;
}

@keyframes progressShimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.job-accession {
  font-family: var(--font-mono);
  font-size: 0.875rem;
  color: var(--accent-primary);
  margin-left: 0.5rem;
}

.source-badge {
  font-size: 0.625rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  background: rgba(6, 182, 212, 0.15);
  color: var(--accent-primary);
  border-radius: 4px;
  text-transform: uppercase;
}

.job-message {
  width: 100%;
  font-size: 0.75rem;
  color: var(--text-secondary);
  padding-top: 0.5rem;
}

.badge--warning {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}
</style>
