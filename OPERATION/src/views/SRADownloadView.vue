<script setup>
import { ref, computed } from 'vue'
import { downloadSRR, runFullPipeline, searchSRA } from '@/services/processing'

// Form state
const accessionInput = ref('')
const searchQuery = ref('')
const useFullPipeline = ref(true)

// Trimming options
const trimmingOptions = ref({
  leading: 3,
  trailing: 3,
  slidingWindow: '4:15',
  minLen: 36
})

// State
const loading = ref(false)
const searchLoading = ref(false)
const error = ref('')
const results = ref(null)
const searchResults = ref([])
const selectedAccessions = ref([])

// Parsed accessions from input
const accessions = computed(() => {
  return accessionInput.value
    .split(/[\s,;]+/)
    .map(a => a.trim().toUpperCase())
    .filter(a => a.startsWith('SRR') || a.startsWith('ERR') || a.startsWith('DRR'))
})

// Search SRA database
async function handleSearch() {
  if (!searchQuery.value.trim()) return
  
  searchLoading.value = true
  error.value = ''
  
  try {
    const data = await searchSRA(searchQuery.value, 20)
    searchResults.value = data.data || []
  } catch (e) {
    error.value = e.response?.data?.error || 'Search failed'
  } finally {
    searchLoading.value = false
  }
}

// Toggle accession selection - use run_accession for downloads
function toggleAccession(record) {
  // Use run_accession (SRR) for downloads, fallback to accession (SRX)
  const acc = record.run_accession || record.accession
  const idx = selectedAccessions.value.indexOf(acc)
  if (idx === -1) {
    selectedAccessions.value.push(acc)
  } else {
    selectedAccessions.value.splice(idx, 1)
  }
}

// Check if a record is selected
function isSelected(record) {
  const acc = record.run_accession || record.accession
  return selectedAccessions.value.includes(acc)
}

// Add selected to input
function addSelectedToInput() {
  const current = accessionInput.value.trim()
  const newAccessions = selectedAccessions.value.filter(
    a => !current.includes(a)
  )
  if (newAccessions.length > 0) {
    accessionInput.value = current 
      ? `${current}, ${newAccessions.join(', ')}`
      : newAccessions.join(', ')
  }
  selectedAccessions.value = []
}

// Submit download/pipeline job
async function handleSubmit() {
  if (accessions.value.length === 0) {
    error.value = 'Please enter at least one valid SRR/ERR/DRR accession'
    return
  }
  
  loading.value = true
  error.value = ''
  results.value = null
  
  try {
    if (useFullPipeline.value) {
      // Run full pipeline for each accession
      const allResults = []
      for (const acc of accessions.value) {
        const result = await runFullPipeline({
          accession: acc,
          ...trimmingOptions.value
        })
        allResults.push(result)
      }
      results.value = { pipeline: allResults }
    } else {
      // Download only
      const data = await downloadSRR(accessions.value)
      results.value = data
    }
  } catch (e) {
    error.value = e.response?.data?.error || 'Operation failed'
  } finally {
    loading.value = false
  }
}

// Format file size
function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Format duration
function formatDuration(ns) {
  if (!ns) return '0s'
  const seconds = ns / 1000000000
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  const minutes = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${minutes}m ${secs}s`
}
</script>

<template>
  <div class="sra-download-page">
    <div class="page-header">
      <div>
        <h2>SRA Download & Processing</h2>
        <p>Download RNA-seq data from NCBI/ENA and process with Trimmomatic</p>
      </div>
    </div>

    <div class="content-grid">
      <!-- Search Panel -->
      <div class="card search-panel">
        <h3>Search SRA Database</h3>
        <form @submit.prevent="handleSearch" class="search-form">
          <input
            v-model="searchQuery"
            type="text"
            class="form-input"
            placeholder="e.g., Homo sapiens RNA-seq cancer"
          />
          <button type="submit" class="btn btn--primary" :disabled="searchLoading">
            {{ searchLoading ? 'Searching...' : 'Search' }}
          </button>
        </form>
        
        <div v-if="searchResults.length > 0" class="search-results">
          <div class="results-header">
            <span>{{ searchResults.length }} results</span>
            <button 
              v-if="selectedAccessions.length > 0"
              class="btn btn--secondary btn--sm"
              @click="addSelectedToInput"
            >
              Add Selected ({{ selectedAccessions.length }})
            </button>
          </div>
          <div 
            v-for="record in searchResults" 
            :key="record.run_accession || record.accession"
            class="search-result-item"
            :class="{ selected: isSelected(record) }"
            @click="toggleAccession(record)"
          >
            <div class="result-main">
              <span class="accession">{{ record.run_accession || record.accession }}</span>
              <span v-if="record.run_accession" class="experiment-id">({{ record.accession }})</span>
              <span class="title">{{ record.title || 'No title' }}</span>
            </div>
            <div class="result-meta">
              <span v-if="record.organism">{{ record.organism }}</span>
              <span v-if="record.total_reads">{{ record.total_reads.toLocaleString() }} reads</span>
              <span v-if="record.library_strategy">{{ record.library_strategy }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Download Form -->
      <div class="card download-panel">
        <h3>Download & Process</h3>
        
        <div v-if="error" class="error-message">{{ error }}</div>
        
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label class="form-label">SRR Accession(s)</label>
            <textarea
              v-model="accessionInput"
              class="form-input"
              rows="3"
              placeholder="Enter SRR accessions (e.g., SRR12345678, SRR87654321)"
            ></textarea>
            <small class="form-hint">
              Separate multiple accessions with commas, spaces, or newlines
            </small>
          </div>

          <div v-if="accessions.length > 0" class="accession-tags">
            <span v-for="acc in accessions" :key="acc" class="tag">
              {{ acc }}
            </span>
          </div>

          <div class="form-group">
            <label class="toggle-label">
              <input type="checkbox" v-model="useFullPipeline" class="toggle" />
              <span>Run full pipeline (download + trimming)</span>
            </label>
          </div>

          <div v-if="useFullPipeline" class="trimming-options">
            <h4>Trimmomatic Options</h4>
            <div class="options-grid">
              <div class="form-group">
                <label class="form-label">Leading</label>
                <input
                  v-model.number="trimmingOptions.leading"
                  type="number"
                  class="form-input"
                  min="0"
                  max="40"
                />
              </div>
              <div class="form-group">
                <label class="form-label">Trailing</label>
                <input
                  v-model.number="trimmingOptions.trailing"
                  type="number"
                  class="form-input"
                  min="0"
                  max="40"
                />
              </div>
              <div class="form-group">
                <label class="form-label">Sliding Window</label>
                <input
                  v-model="trimmingOptions.slidingWindow"
                  type="text"
                  class="form-input"
                  placeholder="4:15"
                />
              </div>
              <div class="form-group">
                <label class="form-label">Min Length</label>
                <input
                  v-model.number="trimmingOptions.minLen"
                  type="number"
                  class="form-input"
                  min="1"
                />
              </div>
            </div>
          </div>

          <button 
            type="submit" 
            class="btn btn--primary btn--lg" 
            :disabled="loading || accessions.length === 0"
            style="width: 100%"
          >
            <span v-if="loading" class="spinner"></span>
            <span v-else>
              {{ useFullPipeline ? 'Download & Process' : 'Download Only' }}
            </span>
          </button>
        </form>
      </div>
    </div>

    <!-- Results -->
    <div v-if="results" class="results-section">
      <h3>Results</h3>
      
      <div v-if="results.results" class="results-list">
        <div v-for="result in results.results" :key="result.accession" class="result-card">
          <div class="result-header">
            <span class="accession">{{ result.accession }}</span>
            <span class="badge" :class="result.status === 'completed' ? 'badge--success' : 'badge--danger'">
              {{ result.status }}
            </span>
          </div>
          <div v-if="result.error" class="result-error">{{ result.error }}</div>
          <div v-else class="result-details">
            <div class="detail-item">
              <span class="label">Files:</span>
              <span class="value">{{ result.files?.length || 0 }}</span>
            </div>
            <div class="detail-item">
              <span class="label">Duration:</span>
              <span class="value">{{ formatDuration(result.duration) }}</span>
            </div>
            <div v-if="result.files" class="files-list">
              <div v-for="file in result.files" :key="file" class="file-item">
                {{ file.split('/').pop() }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="results.pipeline" class="results-list">
        <div v-for="result in results.pipeline" :key="result.download?.accession" class="result-card">
          <div class="result-header">
            <span class="accession">{{ result.download?.accession }}</span>
            <span class="badge" :class="result.status === 'completed' ? 'badge--success' : 'badge--danger'">
              {{ result.status }}
            </span>
          </div>
          <div v-if="result.error" class="result-error">{{ result.error }}</div>
          <div v-else class="result-details">
            <h4>Download</h4>
            <div class="detail-item">
              <span class="label">Files:</span>
              <span class="value">{{ result.download?.files?.length || 0 }}</span>
            </div>
            <h4>Trimming</h4>
            <div class="detail-item">
              <span class="label">Input Reads:</span>
              <span class="value">{{ result.trimming?.input_reads?.toLocaleString() }}</span>
            </div>
            <div class="detail-item">
              <span class="label">Output Reads:</span>
              <span class="value">{{ result.trimming?.output_reads?.toLocaleString() }}</span>
            </div>
            <div v-if="result.quality_comparison" class="quality-section">
              <h4>Quality Improvement</h4>
              <div class="detail-item">
                <span class="label">Mean Quality:</span>
                <span class="value">
                  {{ result.quality_comparison.before?.mean_quality?.toFixed(1) }} →
                  {{ result.quality_comparison.after?.mean_quality?.toFixed(1) }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.sra-download-page {
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

.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-bottom: 2rem;
  
  @media (max-width: 1024px) {
    grid-template-columns: 1fr;
  }
}

.search-panel, .download-panel {
  h3 {
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--border-color);
  }
}

.search-form {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1rem;
  
  .form-input {
    flex: 1;
  }
}

.search-results {
  max-height: 400px;
  overflow-y: auto;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
  font-size: 0.875rem;
  color: var(--text-muted);
}

.search-result-item {
  padding: 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  margin-bottom: 0.5rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  
  &:hover {
    border-color: var(--accent-primary);
  }
  
  &.selected {
    border-color: var(--accent-primary);
    background: rgba(6, 182, 212, 0.1);
  }
}

.result-main {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  
  .accession {
    font-family: var(--font-mono);
    font-weight: 600;
    color: var(--accent-primary);
  }
  
  .experiment-id {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    color: var(--text-muted);
    margin-left: 0.5rem;
  }
  
  .title {
    font-size: 0.875rem;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.result-meta {
  display: flex;
  gap: 1rem;
  margin-top: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.accession-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.tag {
  padding: 0.25rem 0.75rem;
  background: rgba(6, 182, 212, 0.1);
  border: 1px solid var(--accent-primary);
  border-radius: 9999px;
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--accent-primary);
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  
  span {
    font-size: 0.875rem;
  }
}

.trimming-options {
  margin-top: 1rem;
  padding: 1rem;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  
  h4 {
    margin-bottom: 1rem;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }
}

.options-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.form-hint {
  display: block;
  margin-top: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-muted);
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

.spinner {
  display: inline-block;
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.results-section {
  h3 {
    margin-bottom: 1rem;
  }
}

.results-list {
  display: grid;
  gap: 1rem;
}

.result-card {
  padding: 1.25rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
  
  .accession {
    font-family: var(--font-mono);
    font-size: 1.125rem;
    font-weight: 600;
  }
}

.result-error {
  color: var(--accent-danger);
  font-size: 0.875rem;
}

.result-details {
  h4 {
    margin-top: 1rem;
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    color: var(--text-secondary);
    
    &:first-child {
      margin-top: 0;
    }
  }
}

.detail-item {
  display: flex;
  gap: 0.5rem;
  font-size: 0.875rem;
  margin-bottom: 0.25rem;
  
  .label {
    color: var(--text-muted);
  }
  
  .value {
    color: var(--text-primary);
  }
}

.files-list {
  margin-top: 0.5rem;
  padding: 0.75rem;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.file-item {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-secondary);
  padding: 0.25rem 0;
}

.quality-section {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
}
</style>
