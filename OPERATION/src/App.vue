<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/common/AppHeader.vue'
import AppSidebar from '@/components/common/AppSidebar.vue'
import { useAuthStore } from '@/store/auth'

const route = useRoute()
const authStore = useAuthStore()

const isAuthPage = computed(() => {
  return route.name === 'login' || route.name === 'register'
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
</script>

<template>
  <div class="app" :class="{ 'auth-layout': isAuthPage }">
    <template v-if="!isAuthPage && isAuthenticated">
      <AppSidebar />
      <div class="main-content">
        <AppHeader />
        <main class="page-content">
          <router-view />
        </main>
      </div>
    </template>
    <template v-else>
      <router-view />
    </template>
  </div>
</template>

<style lang="scss">
.app {
  display: flex;
  min-height: 100vh;
  background: var(--bg-primary);

  &.auth-layout {
    justify-content: center;
    align-items: center;
    background: linear-gradient(135deg, var(--bg-primary) 0%, var(--bg-secondary) 100%);
  }
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-left: var(--sidebar-width);
  min-height: 100vh;
}

.page-content {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
}
</style>
