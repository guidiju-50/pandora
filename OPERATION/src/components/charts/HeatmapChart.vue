<script setup>
import { ref, onMounted, watch } from 'vue'
import * as d3 from 'd3'

const props = defineProps({
  data: {
    type: Object,
    default: () => ({ rows: [], columns: [], values: [] })
  }
})

const svgRef = ref(null)

const cellSize = 40
const margin = { top: 100, right: 20, bottom: 20, left: 100 }

function renderChart() {
  if (!svgRef.value || !props.data.values.length) return

  const svg = d3.select(svgRef.value)
  svg.selectAll('*').remove()

  const { rows, columns, values } = props.data
  const width = columns.length * cellSize + margin.left + margin.right
  const height = rows.length * cellSize + margin.top + margin.bottom

  svg.attr('width', width).attr('height', height)

  const g = svg
    .append('g')
    .attr('transform', `translate(${margin.left},${margin.top})`)

  // Color scale
  const allValues = values.flat()
  const colorScale = d3.scaleSequential()
    .domain([d3.min(allValues), d3.max(allValues)])
    .interpolator(d3.interpolateRdBu)

  // Draw cells
  rows.forEach((row, i) => {
    columns.forEach((col, j) => {
      g.append('rect')
        .attr('x', j * cellSize)
        .attr('y', i * cellSize)
        .attr('width', cellSize - 1)
        .attr('height', cellSize - 1)
        .attr('fill', colorScale(values[i][j]))
        .attr('rx', 2)
        .style('cursor', 'pointer')
        .on('mouseover', function(event) {
          d3.select(this).attr('stroke', '#fff').attr('stroke-width', 2)
        })
        .on('mouseout', function() {
          d3.select(this).attr('stroke', 'none')
        })
    })
  })

  // Row labels
  g.selectAll('.row-label')
    .data(rows)
    .enter()
    .append('text')
    .attr('class', 'row-label')
    .attr('x', -5)
    .attr('y', (d, i) => i * cellSize + cellSize / 2)
    .attr('text-anchor', 'end')
    .attr('alignment-baseline', 'middle')
    .attr('fill', '#94a3b8')
    .attr('font-size', '12px')
    .text(d => d.length > 10 ? d.substring(0, 10) + '...' : d)

  // Column labels
  g.selectAll('.col-label')
    .data(columns)
    .enter()
    .append('text')
    .attr('class', 'col-label')
    .attr('x', (d, i) => i * cellSize + cellSize / 2)
    .attr('y', -5)
    .attr('text-anchor', 'start')
    .attr('transform', (d, i) => `rotate(-45, ${i * cellSize + cellSize / 2}, -5)`)
    .attr('fill', '#94a3b8')
    .attr('font-size', '12px')
    .text(d => d.length > 10 ? d.substring(0, 10) + '...' : d)
}

onMounted(renderChart)
watch(() => props.data, renderChart, { deep: true })
</script>

<template>
  <div class="heatmap-chart">
    <svg ref="svgRef"></svg>
    <div class="color-legend">
      <span>Low</span>
      <div class="gradient"></div>
      <span>High</span>
    </div>
  </div>
</template>

<style scoped>
.heatmap-chart {
  display: flex;
  flex-direction: column;
  align-items: center;
  overflow-x: auto;
}

svg {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.color-legend {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.gradient {
  width: 100px;
  height: 12px;
  background: linear-gradient(to right, #2563eb, #f8fafc, #dc2626);
  border-radius: 2px;
}
</style>
