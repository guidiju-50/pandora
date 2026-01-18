<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useProjectsStore } from '@/store/projects'

const route = useRoute()
const router = useRouter()
const projectsStore = useProjectsStore()

const project = ref(null)

onMounted(async () => {
  const data = await projectsStore.fetchProject(route.params.id)
  if (data) {
    project.value = data
  } else {
    router.push('/projects')
  }
})

function formatDate(date) {
  return new Date(date).toLocaleDateString('en-US', {
    month: 'long',
    day: 'numeric',
    year: 'numeric'
  })
}
</script>

<template>
  <div class="project-detail">
    <div v-if="projectsStore.loading" class="loading">
      <div class="spinner"></div>
    </div>

    <template v-else-if="project">
      <div class="page-header">
        <div>
          <router-link to="/projects" class="back-link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
            Back to Projects
          </router-link>
          <h2>{{ project.name }}</h2>
        </div>
        <div class="header-actions">
          <button class="btn btn--secondary">Edit</button>
          <button class="btn btn--primary">New Experiment</button>
        </div>
      </div>

      <div class="project-info card">
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">Status</span>
            <span class="badge" :class="project.status === 'active' ? 'badge--success' : 'badge--info'">
              {{ project.status }}
            </span>
          </div>
          <div class="info-item">
            <span class="info-label">Created</span>
            <span class="info-value">{{ formatDate(project.created_at) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Last Updated</span>
            <span class="info-value">{{ formatDate(project.updated_at) }}</span>
          </div>
        </div>
        <div v-if="project.description" class="project-description">
          <span class="info-label">Description</span>
          <p>{{ project.description }}</p>
        </div>
      </div>

      <div class="section">
        <h3>Experiments</h3>
        <div class="empty-state">
          <p>No experiments yet</p>
          <button class="btn btn--primary btn--sm">Create Experiment</button>
        </div>
      </div>

      <div class="section">
        <h3>Recent Jobs</h3>
        <div class="empty-state">
          <p>No jobs for this project</p>
        </div>
      </div>
    </template>
  </div>
</template>

<style lang="scss" scoped>
.project-detail {
  animation: fadeIn 0.3s ease-out;
}

.loading {
  display: flex;
  justify-content: center;
  padding: 4rem;
  
  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border-color);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.875rem;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
  
  &:hover {
    color: var(--accent-primary);
  }
}

.header-actions {
  display: flex;
  gap: 0.75rem;
}

.project-info {
  margin-bottom: 2rem;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1.5rem;
  margin-bottom: 1.5rem;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.info-label {
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
}

.info-value {
  color: var(--text-primary);
}

.project-description {
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-color);
  
  p {
    margin-top: 0.5rem;
  }
}

.section {
  margin-bottom: 2rem;
  
  h3 {
    margin-bottom: 1rem;
  }
}

.empty-state {
  text-align: center;
  padding: 2rem;
  background: var(--bg-card);
  border: 1px dashed var(--border-color);
  border-radius: var(--radius-lg);
  
  p {
    margin-bottom: 1rem;
    color: var(--text-muted);
  }
}
</style>
