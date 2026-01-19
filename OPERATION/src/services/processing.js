import axios from 'axios'

// Processing API base URL
const PROCESSING_URL = import.meta.env.VITE_PROCESSING_URL || 'http://localhost:8081/api/v1'

// Processing API client (connects to PROCESSING module)
const processingApi = axios.create({
  baseURL: PROCESSING_URL,
  timeout: 600000, // 10 minutes for large downloads
  headers: {
    'Content-Type': 'application/json'
  }
})

/**
 * Download SRR files from NCBI/ENA (async with job tracking)
 * @param {string[]} accessions - List of SRR accessions
 * @returns {Promise} Job info with job_id
 */
export async function downloadSRR(accessions) {
  const response = await processingApi.post('/jobs/download', {
    accessions,
    use_prefetch: false
  })
  return response.data
}

/**
 * Run full pipeline (download + trimming) - async with job tracking
 * @param {Object} params - Pipeline parameters
 * @returns {Promise} Job info with job_id
 */
export async function runFullPipeline(params) {
  const response = await processingApi.post('/jobs/full-pipeline', {
    accession: params.accession,
    use_prefetch: false,
    leading: params.leading || 3,
    trailing: params.trailing || 3,
    sliding_window: params.slidingWindow || '4:15',
    min_len: params.minLen || 36
  })
  return response.data
}

/**
 * Get job status
 * @param {string} jobId - Job ID
 * @returns {Promise} Job details
 */
export async function getJob(jobId) {
  const response = await processingApi.get(`/jobs/${jobId}`)
  return response.data
}

/**
 * Get all jobs
 * @returns {Promise} List of jobs
 */
export async function getAllJobs() {
  const response = await processingApi.get('/jobs')
  return response.data
}

/**
 * Subscribe to job progress updates via SSE
 * @param {string} jobId - Job ID
 * @param {Function} onProgress - Callback for progress updates
 * @param {Function} onComplete - Callback when job completes
 * @param {Function} onError - Callback on error
 * @returns {EventSource} Event source for cleanup
 */
export function subscribeToJobProgress(jobId, onProgress, onComplete, onError) {
  const eventSource = new EventSource(`${PROCESSING_URL}/jobs/${jobId}/progress`)
  
  eventSource.addEventListener('progress', (event) => {
    const data = JSON.parse(event.data)
    if (onProgress) onProgress(data)
    
    if (data.status === 'completed' || data.status === 'failed') {
      eventSource.close()
      if (onComplete) onComplete(data)
    }
  })
  
  eventSource.onerror = (error) => {
    eventSource.close()
    if (onError) onError(error)
  }
  
  return eventSource
}

/**
 * Poll job status until complete
 * @param {string} jobId - Job ID
 * @param {Function} onProgress - Progress callback
 * @param {number} interval - Poll interval in ms
 * @returns {Promise} Final job result
 */
export async function pollJobUntilComplete(jobId, onProgress, interval = 2000) {
  return new Promise((resolve, reject) => {
    const poll = async () => {
      try {
        const job = await getJob(jobId)
        if (onProgress) onProgress(job)
        
        if (job.status === 'completed') {
          resolve(job)
        } else if (job.status === 'failed') {
          reject(new Error(job.error || 'Job failed'))
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
 * Process FASTQ files with Trimmomatic
 * @param {Object} params - Processing parameters
 * @returns {Promise} Processing results
 */
export async function processFiles(params) {
  const response = await processingApi.post('/jobs/process', {
    input_file_1: params.inputFile1,
    input_file_2: params.inputFile2,
    output_dir: params.outputDir,
    leading: params.leading || 3,
    trailing: params.trailing || 3,
    sliding_window: params.slidingWindow || '4:15',
    min_len: params.minLen || 36
  })
  return response.data
}

/**
 * Search SRA database
 * @param {string} query - Search query
 * @param {number} maxResults - Maximum results
 * @returns {Promise} Search results
 */
export async function searchSRA(query, maxResults = 100) {
  const response = await processingApi.post('/jobs/scrape', {
    query,
    max_results: maxResults
  })
  return response.data
}

/**
 * Check quality of FASTQ file
 * @param {string} filePath - Path to FASTQ file
 * @returns {Promise} Quality metrics
 */
export async function checkQuality(filePath) {
  const response = await processingApi.post('/quality', {
    file_path: filePath
  })
  return response.data
}

/**
 * Get processing service health
 * @returns {Promise} Health status
 */
export async function getHealth() {
  const response = await processingApi.get('/health')
  return response.data
}

export default {
  downloadSRR,
  runFullPipeline,
  processFiles,
  searchSRA,
  checkQuality,
  getHealth
}
