<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { searchSRA } from '@/services/processing'
import { 
  startCompletePipeline, 
  getPipelineJob, 
  cancelPipelineJob,
  listOrganisms 
} from '@/services/analysis'

// Form state
const accessionInput = ref('')
const searchQuery = ref('')
const selectedOrganism = ref('helicoverpa_armigera')

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
const searchResults = ref([])
const selectedAccessions = ref([])
const organisms = ref([])

// Progress tracking
const activeJobs = ref([])
const pollingIntervals = ref([])

// Parsed accessions from input
const accessions = computed(() => {
  return accessionInput.value
    .split(/[\s,;]+/)
    .map(a => a.trim().toUpperCase())
    .filter(a => a.startsWith('SRR') || a.startsWith('ERR') || a.startsWith('DRR'))
})

// Load available organisms on mount
onMounted(async () => {
  try {
    const data = await listOrganisms()
    organisms.value = data.organisms || []
  } catch (e) {
    console.error('Failed to load organisms:', e)
  }
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

// Toggle accession selection
function toggleAccession(record) {
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

// Submit complete pipeline job
async function handleSubmit() {
  if (accessions.value.length === 0) {
    error.value = 'Please enter at least one valid SRR/ERR/DRR accession'
    return
  }
  
  loading.value = true
  error.value = ''
  activeJobs.value = []
  
  try {
    // Start complete pipeline for each accession
    for (const acc of accessions.value) {
      const jobInfo = await startCompletePipeline({
        accession: acc,
        organism: selectedOrganism.value,
        ...trimmingOptions.value
      })
      
      // Add job to active jobs for tracking
      activeJobs.value.push({
        id: jobInfo.job_id,
        accession: acc,
        organism: selectedOrganism.value,
        progress: 0,
        stage: 'Initializing',
        message: 'Pipeline started...',
        status: 'pending',
        output: null
      })
      
      // Start polling for progress
      startPolling(jobInfo.job_id)
    }
    
    // Clear input after submission
    accessionInput.value = ''
    selectedAccessions.value = []
  } catch (e) {
    error.value = e.response?.data?.error || 'Failed to start pipeline'
    loading.value = false
  }
}

// Start polling for job progress
function startPolling(jobId) {
  const interval = setInterval(async () => {
    try {
      const job = await getPipelineJob(jobId)
      
      // Update active job
      const jobIndex = activeJobs.value.findIndex(j => j.id === jobId)
      if (jobIndex !== -1) {
        activeJobs.value[jobIndex] = {
          ...activeJobs.value[jobIndex],
          progress: job.progress,
          stage: job.stage,
          message: job.message,
          status: job.status,
          output: job.output,
          error: job.error
        }
      }
      
      // Check if completed
      if (job.status === 'completed' || job.status === 'failed') {
        clearInterval(interval)
        
        // Check if all jobs are done
        const allDone = activeJobs.value.every(
          j => j.status === 'completed' || j.status === 'failed'
        )
        if (allDone) {
          loading.value = false
        }
      }
    } catch (e) {
      console.error('Polling error:', e)
    }
  }, 2000)
  
  pollingIntervals.value.push(interval)
}

// Cleanup on unmount
onUnmounted(() => {
  pollingIntervals.value.forEach(interval => clearInterval(interval))
})

// Get organism display name
function getOrganismName(key) {
  const org = organisms.value.find(o => o.name === key)
  return org?.scientific_name || key
}

// Cancel a job
async function handleCancelJob(jobId) {
  try {
    await cancelPipelineJob(jobId)
    
    // Update local job state
    const jobIndex = activeJobs.value.findIndex(j => j.id === jobId)
    if (jobIndex !== -1) {
      activeJobs.value[jobIndex].status = 'cancelled'
      activeJobs.value[jobIndex].stage = 'Cancelled'
      activeJobs.value[jobIndex].message = 'Job cancelled by user'
    }
    
    // Stop polling for this job
    // Check if all jobs are done
    const allDone = activeJobs.value.every(
      j => j.status === 'completed' || j.status === 'failed' || j.status === 'cancelled'
    )
    if (allDone) {
      loading.value = false
    }
  } catch (e) {
    console.error('Failed to cancel job:', e)
    error.value = e.response?.data?.error || 'Failed to cancel job'
  }
}

// Cancel all active jobs
async function handleCancelAll() {
  const runningJobs = activeJobs.value.filter(
    j => j.status === 'pending' || j.status === 'running'
  )
  
  for (const job of runningJobs) {
    await handleCancelJob(job.id)
  }
}
</script>

<template>
  <div class="sra-download-page">
    <div class="page-header">
      <div>
        <h2>Complete RNA-seq Pipeline</h2>
        <p>Download → Trim → Quantify → TPM Matrix</p>
      </div>
      <div class="pipeline-stages">
        <span class="stage">1. Download (NCBI/ENA)</span>
        <span class="arrow">→</span>
        <span class="stage">2. Trimmomatic</span>
        <span class="arrow">→</span>
        <span class="stage">3. Kallisto</span>
        <span class="arrow">→</span>
        <span class="stage">4. Matrix TPM</span>
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
            placeholder="e.g., Helicoverpa armigera RNA-seq"
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

      <!-- Pipeline Form -->
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
              placeholder="Enter SRR accessions (e.g., SRR974918)"
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
            <label class="form-label">Target Organism</label>
            <select v-model="selectedOrganism" class="form-input form-select">
              <option 
                v-for="org in organisms" 
                :key="org.name" 
                :value="org.name"
              >
                {{ org.scientific_name }} 
                <span v-if="org.available">(index ready)</span>
              </option>
            </select>
            <small class="form-hint">
              Index will be downloaded automatically if not available (~5-10 min first time)
            </small>
          </div>

          <div class="trimming-options">
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
            class="btn btn--primary btn--lg submit-btn" 
            :disabled="loading || accessions.length === 0"
          >
            <span v-if="loading" class="spinner"></span>
            <span v-else>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="5 3 19 12 5 21 5 3"></polygon>
              </svg>
              Start Complete Pipeline
            </span>
          </button>
        </form>
      </div>
    </div>

    <!-- Active Jobs with Progress -->
    <div v-if="activeJobs.length > 0" class="active-jobs-section">
      <div class="section-header">
        <h3>Pipeline Jobs</h3>
        <button 
          v-if="activeJobs.some(j => j.status === 'pending' || j.status === 'running')"
          class="btn btn--danger btn--sm"
          @click="handleCancelAll"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
            <line x1="9" y1="9" x2="15" y2="15"></line>
            <line x1="15" y1="9" x2="9" y2="15"></line>
          </svg>
          Cancel All
        </button>
      </div>
      <div class="jobs-list">
        <div v-for="job in activeJobs" :key="job.id" class="job-card" :class="'job-' + job.status">
          <div class="job-header">
            <div class="job-info">
              <span class="accession">{{ job.accession }}</span>
              <span class="organism">{{ getOrganismName(job.organism) }}</span>
            </div>
            <div class="job-actions">
              <span 
                class="badge" 
                :class="{
                  'badge--warning': job.status === 'pending' || job.status === 'running',
                  'badge--success': job.status === 'completed',
                  'badge--danger': job.status === 'failed',
                  'badge--muted': job.status === 'cancelled'
                }"
              >
                {{ job.status }}
              </span>
              <button 
                v-if="job.status === 'pending' || job.status === 'running'"
                class="btn btn--danger btn--sm btn--cancel"
                @click.stop="handleCancelJob(job.id)"
                title="Cancel this job"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="15" y1="9" x2="9" y2="15"></line>
                  <line x1="9" y1="9" x2="15" y2="15"></line>
                </svg>
                Cancel
              </button>
            </div>
          </div>
          
          <!-- Progress Bar -->
          <div class="progress-container">
            <div class="progress-bar">
              <div 
                class="progress-fill" 
                :style="{ width: `${job.progress}%` }"
                :class="{ 'progress-animated': job.status === 'running' }"
              ></div>
            </div>
            <span class="progress-text">{{ job.progress }}%</span>
          </div>
          
          <div class="job-stage">
            <strong>{{ job.stage }}</strong>
            <span class="job-message">{{ job.message }}</span>
          </div>
          
          <!-- Show output when completed -->
          <div v-if="job.status === 'completed' && job.output" class="job-output">
            <div class="output-grid">
              <div class="output-item">
                <span class="label">Matrix TPM</span>
                <span class="value file">{{ job.output.matrix_file?.split('/').pop() }}</span>
              </div>
              <div class="output-item">
                <span class="label">Total Reads</span>
                <span class="value">{{ job.output.total_reads?.toLocaleString() }}</span>
              </div>
              <div class="output-item">
                <span class="label">Mapped Reads</span>
                <span class="value">{{ job.output.mapped_reads?.toLocaleString() }}</span>
              </div>
              <div class="output-item">
                <span class="label">Mapping Rate</span>
                <span class="value success">{{ (job.output.mapping_rate * 100).toFixed(1) }}%</span>
              </div>
              <div class="output-item">
                <span class="label">Transcripts</span>
                <span class="value">{{ job.output.transcript_count?.toLocaleString() }}</span>
              </div>
            </div>
            <div v-if="job.output.matrix_file" class="matrix-path">
              <strong>Output file:</strong>
              <code>{{ job.output.matrix_file }}</code>
            </div>
          </div>
          
          <div v-if="job.status === 'failed'" class="job-error">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="15" y1="9" x2="9" y2="15"></line>
              <line x1="9" y1="9" x2="15" y2="15"></line>
            </svg>
            {{ job.error || 'Pipeline failed' }}
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
    background: linear-gradient(135deg, var(--accent-primary), #a78bfa);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }
  
  p {
    color: var(--text-muted);
    margin-bottom: 1rem;
  }
}

.pipeline-stages {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  
  .stage {
    padding: 0.375rem 0.75rem;
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-weight: 500;
  }
  
  .arrow {
    color: var(--accent-primary);
    font-weight: bold;
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

.form-select {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239ca3af' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.75rem center;
  padding-right: 2.5rem;
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

.submit-btn {
  width: 100%;
  margin-top: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  
  svg {
    flex-shrink: 0;
  }
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

// Active Jobs Section
.active-jobs-section {
  h3 {
    margin-bottom: 1rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    
    &::before {
      content: '';
      display: inline-block;
      width: 8px;
      height: 8px;
      background: var(--accent-primary);
      border-radius: 50%;
      animation: pulse 2s ease infinite;
    }
  }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.jobs-list {
  display: grid;
  gap: 1rem;
}

.job-card {
  padding: 1.5rem;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  transition: all 0.3s ease;
  
  &.job-completed {
    border-color: rgba(34, 197, 94, 0.3);
    background: linear-gradient(135deg, rgba(34, 197, 94, 0.05), transparent);
  }
  
  &.job-failed {
    border-color: rgba(239, 68, 68, 0.3);
    background: linear-gradient(135deg, rgba(239, 68, 68, 0.05), transparent);
  }
  
  &.job-running {
    border-color: rgba(6, 182, 212, 0.3);
  }
}

.job-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.job-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  
  .accession {
    font-family: var(--font-mono);
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text-primary);
  }
  
  .organism {
    font-size: 0.875rem;
    color: var(--text-muted);
    font-style: italic;
  }
}

.progress-container {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.progress-bar {
  flex: 1;
  height: 10px;
  background: var(--bg-secondary);
  border-radius: 5px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-primary), #a78bfa);
  border-radius: 5px;
  transition: width 0.5s ease;
}

.progress-animated {
  background: linear-gradient(
    90deg,
    var(--accent-primary),
    #a78bfa,
    var(--accent-primary)
  );
  background-size: 200% 100%;
  animation: progressShimmer 2s ease infinite;
}

@keyframes progressShimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.progress-text {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--accent-primary);
  min-width: 3.5rem;
  text-align: right;
}

.job-stage {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  
  strong {
    color: var(--text-primary);
    font-size: 0.875rem;
  }
  
  .job-message {
    font-size: 0.8rem;
    color: var(--text-muted);
  }
}

.job-output {
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
}

.output-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.output-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  
  .label {
    font-size: 0.75rem;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  
  .value {
    font-weight: 600;
    font-size: 1rem;
    color: var(--text-primary);
    
    &.file {
      font-family: var(--font-mono);
      font-size: 0.8rem;
      color: var(--accent-primary);
    }
    
    &.success {
      color: #22c55e;
    }
  }
}

.matrix-path {
  padding: 0.75rem;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  
  strong {
    margin-right: 0.5rem;
    color: var(--text-secondary);
  }
  
  code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--accent-primary);
    word-break: break-all;
  }
}

.job-error {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: var(--radius-md);
  color: var(--accent-danger);
  font-size: 0.875rem;
  
  svg {
    flex-shrink: 0;
  }
}

.badge--warning {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}

.badge--muted {
  background: rgba(107, 114, 128, 0.2);
  color: #9ca3af;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.job-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.btn--cancel {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  
  svg {
    flex-shrink: 0;
  }
}

.btn--danger {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
  
  &:hover {
    background: rgba(239, 68, 68, 0.25);
    border-color: rgba(239, 68, 68, 0.5);
  }
}

.btn--sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.job-cancelled {
  opacity: 0.7;
  
  .progress-fill {
    background: linear-gradient(90deg, #6b7280, #9ca3af);
  }
}
</style>
