<script setup>
import { ref, onMounted } from 'vue'
import { useProjectsStore } from '@/store/projects'
import { useJobsStore } from '@/store/jobs'

const projectsStore = useProjectsStore()
const jobsStore = useJobsStore()

const stats = ref([
  { label: 'Projects', value: 0, icon: 'folder', color: 'cyan' },
  { label: 'Running Jobs', value: 0, icon: 'cpu', color: 'purple' },
  { label: 'Completed', value: 0, icon: 'check', color: 'green' },
  { label: 'Total Samples', value: 0, icon: 'dna', color: 'amber' }
])

const recentProjects = ref([])
const recentJobs = ref([])

onMounted(async () => {
  await projectsStore.fetchProjects()
  
  stats.value[0].value = projectsStore.projectCount
  recentProjects.value = projectsStore.projects.slice(0, 5)
})

function formatDate(date) {
  return new Date(date).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric'
  })
}

function getStatusClass(status) {
  const classes = {
    pending: 'badge--info',
    queued: 'badge--info',
    running: 'badge--warning',
    completed: 'badge--success',
    failed: 'badge--danger'
  }
  return classes[status] || 'badge--info'
}
</script>

<template>
  <div class="dashboard">
    <!-- Stats Grid -->
    <div class="stats-grid">
      <div v-for="stat in stats" :key="stat.label" class="stat-card" :class="`stat-card--${stat.color}`">
        <div class="stat-icon">
          <svg v-if="stat.icon === 'folder'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
          </svg>
          <svg v-if="stat.icon === 'cpu'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect>
            <rect x="9" y="9" width="6" height="6"></rect>
          </svg>
          <svg v-if="stat.icon === 'check'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          <svg v-if="stat.icon === 'dna'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M2 15c6.667-6 13.333 0 20-6"></path>
            <path d="M9 22c1.798-1.998 2.518-3.995 2.807-5.993"></path>
            <path d="M15 2c-1.798 1.998-2.518 3.995-2.807 5.993"></path>
            <path d="M17 6l-2.5-2.5"></path>
            <path d="M14 8l-3-3"></path>
            <path d="M7 18l2.5 2.5"></path>
            <path d="M3.5 14.5l.5.5"></path>
            <path d="M20 9l.5.5"></path>
            <path d="M6.5 12.5l1 1"></path>
            <path d="M16.5 10.5l1 1"></path>
            <path d="M10 16l3 3"></path>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-label">{{ stat.label }}</div>
        </div>
      </div>
    </div>

    <!-- Content Grid -->
    <div class="content-grid">
      <!-- Recent Projects -->
      <div class="card">
        <div class="card__header">
          <h3 class="card__title">Recent Projects</h3>
          <router-link to="/projects" class="btn btn--ghost btn--sm">View All</router-link>
        </div>
        <div class="card__content">
          <div v-if="recentProjects.length === 0" class="empty-state">
            <p>No projects yet</p>
            <router-link to="/projects" class="btn btn--primary btn--sm">Create Project</router-link>
          </div>
          <div v-else class="project-list">
            <div v-for="project in recentProjects" :key="project.id" class="project-item">
              <div class="project-info">
                <router-link :to="`/projects/${project.id}`" class="project-name">
                  {{ project.name }}
                </router-link>
                <span class="project-date">{{ formatDate(project.created_at) }}</span>
              </div>
              <span class="badge" :class="project.status === 'active' ? 'badge--success' : 'badge--info'">
                {{ project.status }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Jobs -->
      <div class="card">
        <div class="card__header">
          <h3 class="card__title">Recent Jobs</h3>
          <router-link to="/jobs" class="btn btn--ghost btn--sm">View All</router-link>
        </div>
        <div class="card__content">
          <div v-if="recentJobs.length === 0" class="empty-state">
            <p>No jobs running</p>
          </div>
          <div v-else class="job-list">
            <div v-for="job in recentJobs" :key="job.id" class="job-item">
              <div class="job-info">
                <span class="job-type">{{ job.type }}</span>
                <span class="job-id">{{ job.id.slice(0, 8) }}</span>
              </div>
              <span class="badge" :class="getStatusClass(job.status)">
                {{ job.status }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div class="quick-actions">
      <h3>Quick Actions</h3>
      <div class="actions-grid">
        <router-link to="/projects" class="action-card">
          <div class="action-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
          </div>
          <span>New Project</span>
        </router-link>
        <router-link to="/analysis" class="action-card">
          <div class="action-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
            </svg>
          </div>
          <span>Run Analysis</span>
        </router-link>
        <router-link to="/jobs" class="action-card">
          <div class="action-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
          </div>
          <span>View Jobs</span>
        </router-link>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.dashboard {
  animation: fadeIn 0.3s ease-out;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  transition: all var(--transition-normal);
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-lg);
  }
  
  &--cyan .stat-icon { color: var(--accent-primary); background: rgba(6, 182, 212, 0.1); }
  &--purple .stat-icon { color: var(--accent-secondary); background: rgba(139, 92, 246, 0.1); }
  &--green .stat-icon { color: var(--accent-tertiary); background: rgba(16, 185, 129, 0.1); }
  &--amber .stat-icon { color: var(--accent-warning); background: rgba(245, 158, 11, 0.1); }
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  line-height: 1;
  color: var(--text-primary);
}

.stat-label {
  font-size: 0.875rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.empty-state {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  
  p {
    margin-bottom: 1rem;
  }
}

.project-list, .job-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.project-item, .job-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.project-info, .job-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.project-name {
  font-weight: 500;
  color: var(--text-primary);
  
  &:hover {
    color: var(--accent-primary);
  }
}

.project-date, .job-id {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.job-type {
  font-weight: 500;
  color: var(--text-primary);
  text-transform: capitalize;
}

.quick-actions {
  h3 {
    margin-bottom: 1rem;
  }
}

.actions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.action-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 1.5rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  color: var(--text-secondary);
  text-decoration: none;
  transition: all var(--transition-normal);
  
  &:hover {
    border-color: var(--accent-primary);
    color: var(--accent-primary);
    
    .action-icon {
      background: rgba(6, 182, 212, 0.1);
    }
  }
}

.action-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  transition: background var(--transition-fast);
}
</style>
