<script setup>
import { ref, computed } from 'vue'
import { useJobsStore } from '@/store/jobs'

const jobsStore = useJobsStore()

const filter = ref('all')

const filteredJobs = computed(() => {
  if (filter.value === 'all') return jobsStore.jobs
  return jobsStore.jobs.filter(job => job.status === filter.value)
})

const statusCounts = computed(() => ({
  all: jobsStore.jobs.length,
  running: jobsStore.runningJobs.length,
  completed: jobsStore.completedJobs.length,
  failed: jobsStore.failedJobs.length
}))

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
            <div class="job-title">{{ job.type }}</div>
            <div class="job-id">{{ job.id }}</div>
          </div>
        </div>
        
        <div class="job-meta">
          <span class="badge" :class="getStatusClass(job.status)">{{ job.status }}</span>
          <span class="job-date">{{ formatDate(job.created_at) }}</span>
        </div>

        <div v-if="job.status === 'running'" class="job-progress">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: job.progress + '%' }"></div>
          </div>
          <span class="progress-text">{{ job.progress }}%</span>
        </div>
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
</style>
