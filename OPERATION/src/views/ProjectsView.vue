<script setup>
import { ref, onMounted } from 'vue'
import { useProjectsStore } from '@/store/projects'

const projectsStore = useProjectsStore()

const showModal = ref(false)
const newProject = ref({ name: '', description: '' })
const formError = ref('')

onMounted(() => {
  projectsStore.fetchProjects()
})

async function createProject() {
  if (!newProject.value.name) {
    formError.value = 'Project name is required'
    return
  }
  
  const result = await projectsStore.createProject(newProject.value)
  
  if (result.success) {
    showModal.value = false
    newProject.value = { name: '', description: '' }
    formError.value = ''
  } else {
    formError.value = result.error
  }
}

function formatDate(date) {
  return new Date(date).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric'
  })
}
</script>

<template>
  <div class="projects-page">
    <div class="page-header">
      <div>
        <h2>Projects</h2>
        <p>Manage your research projects</p>
      </div>
      <button class="btn btn--primary" @click="showModal = true">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        New Project
      </button>
    </div>

    <div v-if="projectsStore.loading" class="loading">
      <div class="spinner"></div>
      <p>Loading projects...</p>
    </div>

    <div v-else-if="projectsStore.projects.length === 0" class="empty-state">
      <div class="empty-icon">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
      </div>
      <h3>No projects yet</h3>
      <p>Create your first project to get started</p>
      <button class="btn btn--primary" @click="showModal = true">Create Project</button>
    </div>

    <div v-else class="projects-grid">
      <router-link
        v-for="project in projectsStore.projects"
        :key="project.id"
        :to="`/projects/${project.id}`"
        class="project-card"
      >
        <div class="project-header">
          <h3>{{ project.name }}</h3>
          <span class="badge" :class="project.status === 'active' ? 'badge--success' : 'badge--info'">
            {{ project.status }}
          </span>
        </div>
        <p class="project-description">{{ project.description || 'No description' }}</p>
        <div class="project-footer">
          <span class="project-date">Created {{ formatDate(project.created_at) }}</span>
        </div>
      </router-link>
    </div>

    <!-- Create Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>Create New Project</h3>
          <button class="modal-close" @click="showModal = false">&times;</button>
        </div>
        <form @submit.prevent="createProject" class="modal-body">
          <div v-if="formError" class="error-message">{{ formError }}</div>
          
          <div class="form-group">
            <label class="form-label">Project Name</label>
            <input
              v-model="newProject.name"
              type="text"
              class="form-input"
              placeholder="Enter project name"
            />
          </div>
          
          <div class="form-group">
            <label class="form-label">Description</label>
            <textarea
              v-model="newProject.description"
              class="form-input"
              rows="3"
              placeholder="Enter project description"
            ></textarea>
          </div>
          
          <div class="modal-actions">
            <button type="button" class="btn btn--secondary" @click="showModal = false">Cancel</button>
            <button type="submit" class="btn btn--primary" :disabled="projectsStore.loading">
              Create Project
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.projects-page {
  animation: fadeIn 0.3s ease-out;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
  
  h2 {
    margin-bottom: 0.25rem;
  }
  
  p {
    color: var(--text-muted);
  }
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem;
  
  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border-color);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 1rem;
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
    margin-bottom: 1.5rem;
  }
}

.projects-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.5rem;
}

.project-card {
  display: flex;
  flex-direction: column;
  padding: 1.5rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  text-decoration: none;
  transition: all var(--transition-normal);
  
  &:hover {
    border-color: var(--accent-primary);
    box-shadow: var(--shadow-glow);
    transform: translateY(-2px);
  }
}

.project-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.75rem;
  
  h3 {
    font-size: 1.125rem;
    color: var(--text-primary);
  }
}

.project-description {
  flex: 1;
  font-size: 0.875rem;
  color: var(--text-muted);
  margin-bottom: 1rem;
}

.project-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
}

.project-date {
  font-size: 0.75rem;
  color: var(--text-muted);
}

// Modal styles
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.2s ease-out;
}

.modal {
  width: 100%;
  max-width: 480px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
  
  h3 {
    font-size: 1.25rem;
  }
}

.modal-close {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  font-size: 1.5rem;
  color: var(--text-muted);
  cursor: pointer;
  
  &:hover {
    color: var(--text-primary);
  }
}

.modal-body {
  padding: 1.5rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  margin-top: 1.5rem;
}

.error-message {
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: var(--radius-md);
  color: var(--accent-danger);
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

textarea.form-input {
  resize: vertical;
  min-height: 80px;
}
</style>
