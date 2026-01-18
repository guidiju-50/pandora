import axios from 'axios'

// Processing API client (connects to PROCESSING module)
const processingApi = axios.create({
  baseURL: import.meta.env.VITE_PROCESSING_URL || 'http://localhost:8081/api/v1',
  timeout: 600000, // 10 minutes for large downloads
  headers: {
    'Content-Type': 'application/json'
  }
})

/**
 * Download SRR files from NCBI/ENA
 * @param {string[]} accessions - List of SRR accessions
 * @returns {Promise} Download results
 */
export async function downloadSRR(accessions) {
  const response = await processingApi.post('/jobs/download', {
    accessions,
    use_prefetch: false
  })
  return response.data
}

/**
 * Run full pipeline (download + trimming)
 * @param {Object} params - Pipeline parameters
 * @returns {Promise} Pipeline results
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
