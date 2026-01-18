<script setup>
// Experiments View - placeholder
</script>

<template>
  <div class="experiments-page">
    <div class="page-header">
      <div>
        <h2>Experiments</h2>
        <p>Manage your RNA-seq experiments</p>
      </div>
      <button class="btn btn--primary">New Experiment</button>
    </div>

    <div class="empty-state">
      <div class="empty-icon">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M9 3h6v5l4 9H5l4-9V3z"></path>
          <path d="M9 3h6"></path>
        </svg>
      </div>
      <h3>No experiments yet</h3>
      <p>Create a project first, then add experiments to it</p>
      <router-link to="/projects" class="btn btn--primary">Go to Projects</router-link>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.experiments-page {
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
</style>
