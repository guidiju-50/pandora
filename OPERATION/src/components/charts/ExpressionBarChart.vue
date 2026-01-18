<script setup>
import { ref, onMounted, watch } from 'vue'
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js'

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend)

const props = defineProps({
  data: {
    type: Object,
    default: () => ({ labels: [], datasets: [] })
  },
  title: {
    type: String,
    default: 'Gene Expression'
  }
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: true,
  plugins: {
    legend: {
      position: 'top',
      labels: {
        color: '#94a3b8',
        font: {
          family: 'Space Grotesk'
        }
      }
    },
    title: {
      display: true,
      text: props.title,
      color: '#f1f5f9',
      font: {
        family: 'Space Grotesk',
        size: 16,
        weight: '600'
      }
    },
    tooltip: {
      backgroundColor: '#1e293b',
      titleColor: '#f1f5f9',
      bodyColor: '#94a3b8',
      borderColor: '#334155',
      borderWidth: 1
    }
  },
  scales: {
    x: {
      ticks: {
        color: '#94a3b8',
        font: {
          family: 'Space Grotesk'
        }
      },
      grid: {
        color: '#1e293b'
      }
    },
    y: {
      ticks: {
        color: '#94a3b8',
        font: {
          family: 'Space Grotesk'
        }
      },
      grid: {
        color: '#1e293b'
      },
      title: {
        display: true,
        text: 'Expression Level',
        color: '#94a3b8'
      }
    }
  }
}

const chartData = ref({
  labels: [],
  datasets: []
})

function updateChartData() {
  chartData.value = {
    labels: props.data.labels || [],
    datasets: (props.data.datasets || []).map((dataset, i) => ({
      ...dataset,
      backgroundColor: ['#06b6d4', '#8b5cf6', '#10b981', '#f59e0b'][i % 4],
      borderColor: ['#06b6d4', '#8b5cf6', '#10b981', '#f59e0b'][i % 4],
      borderWidth: 1,
      borderRadius: 4
    }))
  }
}

onMounted(updateChartData)
watch(() => props.data, updateChartData, { deep: true })
</script>

<template>
  <div class="expression-chart">
    <Bar :data="chartData" :options="chartOptions" />
  </div>
</template>

<style scoped>
.expression-chart {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  padding: 1rem;
}
</style>
