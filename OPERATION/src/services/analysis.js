import axios from 'axios'

// Analysis API base URL
const ANALYSIS_URL = import.meta.env.VITE_ANALYSIS_URL || 'http://localhost:8082/api/v1'

// Analysis API client (connects to ANALYSIS module)
const analysisApi = axios.create({
  baseURL: ANALYSIS_URL,
  timeout: 1800000, // 30 minutes for complete pipeline
  headers: {
    'Content-Type': 'application/json'
  }
})

/**
 * Start the complete pipeline: Download → Trim → Quantify → Matrix TPM
 * @param {Object} params - Pipeline parameters
 * @param {string} params.accession - SRR accession number
 * @param {string} params.organism - Organism name (e.g., "helicoverpa_armigera")
 * @param {number} params.leading - Trimmomatic LEADING option
 * @param {number} params.trailing - Trimmomatic TRAILING option
 * @param {string} params.slidingWindow - Trimmomatic SLIDINGWINDOW option
 * @param {number} params.minLen - Trimmomatic MINLEN option
 * @returns {Promise} Job info with job_id
 */
export async function startCompletePipeline(params) {
  const response = await analysisApi.post('/pipeline/start', {
    accession: params.accession,
    organism: params.organism || 'helicoverpa_armigera',
    leading: params.leading || 3,
    trailing: params.trailing || 3,
    sliding_window: params.slidingWindow || '4:15',
    min_len: params.minLen || 36
  })
  return response.data
}

/**
 * Get pipeline job status
 * @param {string} jobId - Pipeline job ID
 * @returns {Promise} Job details
 */
export async function getPipelineJob(jobId) {
  const response = await analysisApi.get(`/pipeline/jobs/${jobId}`)
  return response.data
}

/**
 * Get all pipeline jobs
 * @returns {Promise} List of pipeline jobs
 */
export async function getAllPipelineJobs() {
  const response = await analysisApi.get('/pipeline/jobs')
  return response.data
}

/**
 * Cancel a pipeline job
 * @param {string} jobId - Pipeline job ID
 * @returns {Promise} Cancellation result
 */
export async function cancelPipelineJob(jobId) {
  const response = await analysisApi.post(`/pipeline/jobs/${jobId}/cancel`)
  return response.data
}

/**
 * Poll pipeline job until complete
 * @param {string} jobId - Job ID
 * @param {Function} onProgress - Progress callback
 * @param {number} interval - Poll interval in ms
 * @returns {Promise} Final job result
 */
export async function pollPipelineUntilComplete(jobId, onProgress, interval = 2000) {
  return new Promise((resolve, reject) => {
    const poll = async () => {
      try {
        const job = await getPipelineJob(jobId)
        if (onProgress) onProgress(job)
        
        if (job.status === 'completed') {
          resolve(job)
        } else if (job.status === 'failed') {
          reject(new Error(job.error || 'Pipeline failed'))
        } else {
          setTimeout(poll, interval)
        }
      } catch (err) {
        reject(err)
      }
    }
    poll()
  })
}

/**
 * List supported organisms with their Kallisto index status
 * @returns {Promise} List of organisms
 */
export async function listOrganisms() {
  const response = await analysisApi.get('/references')
  return response.data
}

/**
 * Ensure Kallisto index is available for an organism
 * Downloads and builds if necessary
 * @param {string} organism - Organism name
 * @returns {Promise} Index status
 */
export async function ensureIndex(organism) {
  const response = await analysisApi.post('/references/ensure', { organism })
  return response.data
}

/**
 * Run Kallisto quantification
 * @param {Object} params - Quantification parameters
 * @returns {Promise} Quantification results
 */
export async function runKallisto(params) {
  const response = await analysisApi.post('/quantify/kallisto', {
    sample_id: params.sampleId,
    reads1: params.reads1,
    reads2: params.reads2,
    index: params.index,
    output_dir: params.outputDir,
    bootstrap: params.bootstrap || 100
  })
  return response.data
}

/**
 * Generate TPM matrix from Kallisto output
 * @param {Object} params - Matrix generation parameters
 * @returns {Promise} Matrix file path
 */
export async function generateMatrix(params) {
  const response = await analysisApi.post('/quantify/matrix', {
    sample_id: params.sampleId,
    abundance_dir: params.abundanceDir,
    output_file: params.outputFile
  })
  return response.data
}

/**
 * Run differential expression analysis
 * @param {Object} params - Analysis parameters
 * @returns {Promise} Differential expression results
 */
export async function runDifferentialExpression(params) {
  const response = await analysisApi.post('/analysis/differential', params)
  return response.data
}

/**
 * Run PCA analysis
 * @param {Object} params - PCA parameters
 * @returns {Promise} PCA results
 */
export async function runPCA(params) {
  const response = await analysisApi.post('/analysis/pca', params)
  return response.data
}

/**
 * Run clustering analysis
 * @param {Object} params - Clustering parameters
 * @returns {Promise} Clustering results
 */
export async function runClustering(params) {
  const response = await analysisApi.post('/analysis/clustering', params)
  return response.data
}

/**
 * Get analysis service health
 * @returns {Promise} Health status
 */
export async function getHealth() {
  const response = await analysisApi.get('/health')
  return response.data
}

export default {
  startCompletePipeline,
  getPipelineJob,
  getAllPipelineJobs,
  cancelPipelineJob,
  pollPipelineUntilComplete,
  listOrganisms,
  ensureIndex,
  runKallisto,
  generateMatrix,
  runDifferentialExpression,
  runPCA,
  runClustering,
  getHealth
}
