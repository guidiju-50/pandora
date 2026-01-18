<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const route = useRoute()
const authStore = useAuthStore()

const pageTitle = computed(() => {
  const titles = {
    dashboard: 'Dashboard',
    projects: 'Projects',
    'project-detail': 'Project Details',
    experiments: 'Experiments',
    jobs: 'Jobs',
    analysis: 'Analysis',
    settings: 'Settings'
  }
  return titles[route.name] || 'Pandora'
})

const userInitials = computed(() => {
  const name = authStore.userName || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

function handleLogout() {
  authStore.logout()
  window.location.href = '/login'
}
</script>

<template>
  <header class="app-header">
    <div class="header-left">
      <h1 class="page-title">{{ pageTitle }}</h1>
    </div>
    
    <div class="header-right">
      <button class="notification-btn" title="Notifications">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path>
          <path d="M13.73 21a2 2 0 0 1-3.46 0"></path>
        </svg>
        <span class="notification-dot"></span>
      </button>
      
      <div class="user-menu">
        <div class="avatar">{{ userInitials }}</div>
        <div class="user-info">
          <span class="user-name">{{ authStore.userName }}</span>
          <span class="user-role">{{ authStore.userRole }}</span>
        </div>
        <button class="logout-btn" @click="handleLogout" title="Logout">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
            <polyline points="16 17 21 12 16 7"></polyline>
            <line x1="21" y1="12" x2="9" y2="12"></line>
          </svg>
        </button>
      </div>
    </div>
  </header>
</template>

<style lang="scss" scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--header-height);
  padding: 0 2rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.notification-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  
  &:hover {
    color: var(--accent-primary);
    border-color: var(--accent-primary);
  }
}

.notification-dot {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 8px;
  height: 8px;
  background: var(--accent-danger);
  border-radius: 50%;
}

.user-menu {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--gradient-primary);
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 600;
  color: white;
}

.user-info {
  display: flex;
  flex-direction: column;
}

.user-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-primary);
}

.user-role {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: capitalize;
}

.logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  transition: color var(--transition-fast);
  
  &:hover {
    color: var(--accent-danger);
  }
}
</style>
