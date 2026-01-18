<script setup>
import { ref } from 'vue'

const analysisTypes = [
  {
    id: 'differential',
    name: 'Differential Expression',
    description: 'Compare gene expression between conditions using DESeq2',
    icon: 'chart'
  },
  {
    id: 'pca',
    name: 'PCA Analysis',
    description: 'Reduce dimensionality and visualize sample clustering',
    icon: 'scatter'
  },
  {
    id: 'clustering',
    name: 'Hierarchical Clustering',
    description: 'Group samples or genes based on expression patterns',
    icon: 'tree'
  },
  {
    id: 'enrichment',
    name: 'Pathway Enrichment',
    description: 'Identify enriched GO terms and KEGG pathways',
    icon: 'network'
  }
]

const selectedAnalysis = ref(null)

function selectAnalysis(type) {
  selectedAnalysis.value = type
}
</script>

<template>
  <div class="analysis-page">
    <div class="page-header">
      <div>
        <h2>Analysis</h2>
        <p>Run statistical analyses on your data</p>
      </div>
    </div>

    <div class="analysis-grid">
      <div
        v-for="analysis in analysisTypes"
        :key="analysis.id"
        class="analysis-card"
        :class="{ selected: selectedAnalysis === analysis.id }"
        @click="selectAnalysis(analysis.id)"
      >
        <div class="analysis-icon">
          <svg v-if="analysis.icon === 'chart'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="20" x2="18" y2="10"></line>
            <line x1="12" y1="20" x2="12" y2="4"></line>
            <line x1="6" y1="20" x2="6" y2="14"></line>
          </svg>
          <svg v-else-if="analysis.icon === 'scatter'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="7.5" cy="7.5" r="1.5"></circle>
            <circle cx="18.5" cy="5.5" r="1.5"></circle>
            <circle cx="11.5" cy="11.5" r="1.5"></circle>
            <circle cx="7.5" cy="16.5" r="1.5"></circle>
            <circle cx="17.5" cy="14.5" r="1.5"></circle>
          </svg>
          <svg v-else-if="analysis.icon === 'tree'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 3v12"></path>
            <path d="M6 15v6"></path>
            <path d="M18 15v6"></path>
            <path d="M6 15h12"></path>
          </svg>
          <svg v-else width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"></circle>
            <circle cx="19" cy="5" r="2"></circle>
            <circle cx="5" cy="19" r="2"></circle>
            <line x1="14.5" y1="9.5" x2="17.5" y2="6.5"></line>
            <line x1="9.5" y1="14.5" x2="6.5" y2="17.5"></line>
          </svg>
        </div>
        <div class="analysis-content">
          <h3>{{ analysis.name }}</h3>
          <p>{{ analysis.description }}</p>
        </div>
        <div class="analysis-arrow">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </div>
      </div>
    </div>

    <div v-if="selectedAnalysis" class="analysis-config card">
      <h3>Configure {{ analysisTypes.find(a => a.id === selectedAnalysis)?.name }}</h3>
      <p class="text-muted">Select an experiment and configure analysis parameters</p>
      
      <div class="form-group mt-4">
        <label class="form-label">Select Experiment</label>
        <select class="form-input">
          <option value="">Choose an experiment...</option>
        </select>
      </div>

      <div class="form-actions mt-4">
        <button class="btn btn--secondary" @click="selectedAnalysis = null">Cancel</button>
        <button class="btn btn--primary">Run Analysis</button>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.analysis-page {
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

.analysis-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.analysis-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-normal);
  
  &:hover {
    border-color: var(--accent-primary);
    
    .analysis-icon {
      background: rgba(6, 182, 212, 0.1);
      color: var(--accent-primary);
    }
    
    .analysis-arrow {
      opacity: 1;
      transform: translateX(0);
    }
  }
  
  &.selected {
    border-color: var(--accent-primary);
    box-shadow: var(--shadow-glow);
  }
}

.analysis-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  transition: all var(--transition-fast);
}

.analysis-content {
  flex: 1;
  
  h3 {
    font-size: 1rem;
    margin-bottom: 0.25rem;
  }
  
  p {
    font-size: 0.875rem;
    color: var(--text-muted);
  }
}

.analysis-arrow {
  color: var(--accent-primary);
  opacity: 0;
  transform: translateX(-10px);
  transition: all var(--transition-fast);
}

.analysis-config {
  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
  }
}

.text-muted {
  color: var(--text-muted);
}

.mt-4 {
  margin-top: 1rem;
}

select.form-input {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%2394a3b8' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 1rem center;
  padding-right: 2.5rem;
}
</style>
