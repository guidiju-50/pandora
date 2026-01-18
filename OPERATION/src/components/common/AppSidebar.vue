<script setup>
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'

const route = useRoute()

const navItems = [
  { 
    name: 'Dashboard', 
    path: '/', 
    icon: 'dashboard',
    routeName: 'dashboard'
  },
  { 
    name: 'Projects', 
    path: '/projects', 
    icon: 'folder',
    routeName: 'projects'
  },
  { 
    name: 'Experiments', 
    path: '/experiments', 
    icon: 'flask',
    routeName: 'experiments'
  },
  { 
    name: 'Jobs', 
    path: '/jobs', 
    icon: 'cpu',
    routeName: 'jobs'
  },
  { 
    name: 'Analysis', 
    path: '/analysis', 
    icon: 'chart',
    routeName: 'analysis'
  },
  { 
    name: 'Settings', 
    path: '/settings', 
    icon: 'settings',
    routeName: 'settings'
  }
]

function isActive(item) {
  return route.name === item.routeName || route.path.startsWith(item.path + '/')
}
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="logo">
        <div class="logo-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" fill="url(#gradient1)"/>
            <path d="M2 17L12 22L22 17" stroke="url(#gradient2)" stroke-width="2"/>
            <path d="M2 12L12 17L22 12" stroke="url(#gradient2)" stroke-width="2"/>
            <defs>
              <linearGradient id="gradient1" x1="2" y1="2" x2="22" y2="12">
                <stop stop-color="#06b6d4"/>
                <stop offset="1" stop-color="#8b5cf6"/>
              </linearGradient>
              <linearGradient id="gradient2" x1="2" y1="12" x2="22" y2="22">
                <stop stop-color="#06b6d4"/>
                <stop offset="1" stop-color="#8b5cf6"/>
              </linearGradient>
            </defs>
          </svg>
        </div>
        <span class="logo-text">Pandora</span>
      </div>
    </div>
    
    <nav class="sidebar-nav">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :class="{ active: isActive(item) }"
      >
        <span class="nav-icon">
          <!-- Dashboard -->
          <svg v-if="item.icon === 'dashboard'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7"></rect>
            <rect x="14" y="3" width="7" height="7"></rect>
            <rect x="14" y="14" width="7" height="7"></rect>
            <rect x="3" y="14" width="7" height="7"></rect>
          </svg>
          <!-- Folder -->
          <svg v-if="item.icon === 'folder'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
          </svg>
          <!-- Flask -->
          <svg v-if="item.icon === 'flask'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 3h6v5l4 9H5l4-9V3z"></path>
            <path d="M9 3h6"></path>
          </svg>
          <!-- CPU -->
          <svg v-if="item.icon === 'cpu'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect>
            <rect x="9" y="9" width="6" height="6"></rect>
            <line x1="9" y1="1" x2="9" y2="4"></line>
            <line x1="15" y1="1" x2="15" y2="4"></line>
            <line x1="9" y1="20" x2="9" y2="23"></line>
            <line x1="15" y1="20" x2="15" y2="23"></line>
          </svg>
          <!-- Chart -->
          <svg v-if="item.icon === 'chart'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="20" x2="18" y2="10"></line>
            <line x1="12" y1="20" x2="12" y2="4"></line>
            <line x1="6" y1="20" x2="6" y2="14"></line>
          </svg>
          <!-- Settings -->
          <svg v-if="item.icon === 'settings'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"></circle>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
          </svg>
        </span>
        <span class="nav-text">{{ item.name }}</span>
      </RouterLink>
    </nav>
    
    <div class="sidebar-footer">
      <div class="version">v1.0.0</div>
    </div>
  </aside>
</template>

<style lang="scss" scoped>
.sidebar {
  position: fixed;
  left: 0;
  top: 0;
  width: var(--sidebar-width);
  height: 100vh;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  z-index: 100;
}

.sidebar-header {
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.logo-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-text {
  font-size: 1.5rem;
  font-weight: 700;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.sidebar-nav {
  flex: 1;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  text-decoration: none;
  transition: all var(--transition-fast);
  
  &:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
  
  &.active {
    background: rgba(6, 182, 212, 0.1);
    color: var(--accent-primary);
    
    .nav-icon {
      color: var(--accent-primary);
    }
  }
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
}

.nav-text {
  font-size: 0.875rem;
  font-weight: 500;
}

.sidebar-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-color);
}

.version {
  font-size: 0.75rem;
  color: var(--text-muted);
}
</style>
