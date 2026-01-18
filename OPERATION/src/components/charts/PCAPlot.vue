<script setup>
import { ref, onMounted, watch } from 'vue'
import * as d3 from 'd3'

const props = defineProps({
  data: {
    type: Array,
    default: () => []
  },
  varianceExplained: {
    type: Array,
    default: () => [0, 0]
  }
})

const emit = defineEmits(['pointClick'])
const svgRef = ref(null)

const width = 500
const height = 400
const margin = { top: 20, right: 100, bottom: 50, left: 60 }

function renderChart() {
  if (!svgRef.value || props.data.length === 0) return

  const svg = d3.select(svgRef.value)
  svg.selectAll('*').remove()

  const innerWidth = width - margin.left - margin.right
  const innerHeight = height - margin.top - margin.bottom

  const g = svg
    .append('g')
    .attr('transform', `translate(${margin.left},${margin.top})`)

  // Get unique groups for coloring
  const groups = [...new Set(props.data.map(d => d.group))]
  const colorScale = d3.scaleOrdinal()
    .domain(groups)
    .range(['#06b6d4', '#8b5cf6', '#10b981', '#f59e0b', '#ef4444'])

  // Scales
  const xExtent = d3.extent(props.data, d => d.pc1)
  const yExtent = d3.extent(props.data, d => d.pc2)

  const xScale = d3.scaleLinear()
    .domain([xExtent[0] * 1.1, xExtent[1] * 1.1])
    .range([0, innerWidth])

  const yScale = d3.scaleLinear()
    .domain([yExtent[0] * 1.1, yExtent[1] * 1.1])
    .range([innerHeight, 0])

  // Axes
  g.append('g')
    .attr('transform', `translate(0,${innerHeight})`)
    .call(d3.axisBottom(xScale))
    .selectAll('text, line, path')
    .attr('stroke', '#64748b')
    .attr('fill', '#94a3b8')

  g.append('g')
    .call(d3.axisLeft(yScale))
    .selectAll('text, line, path')
    .attr('stroke', '#64748b')
    .attr('fill', '#94a3b8')

  // Axis labels
  const pc1Var = props.varianceExplained[0]?.toFixed(1) || '0'
  const pc2Var = props.varianceExplained[1]?.toFixed(1) || '0'

  g.append('text')
    .attr('x', innerWidth / 2)
    .attr('y', innerHeight + 40)
    .attr('text-anchor', 'middle')
    .attr('fill', '#94a3b8')
    .text(`PC1 (${pc1Var}%)`)

  g.append('text')
    .attr('transform', 'rotate(-90)')
    .attr('x', -innerHeight / 2)
    .attr('y', -45)
    .attr('text-anchor', 'middle')
    .attr('fill', '#94a3b8')
    .text(`PC2 (${pc2Var}%)`)

  // Zero lines
  g.append('line')
    .attr('x1', 0)
    .attr('x2', innerWidth)
    .attr('y1', yScale(0))
    .attr('y2', yScale(0))
    .attr('stroke', '#334155')
    .attr('stroke-dasharray', '4,4')

  g.append('line')
    .attr('x1', xScale(0))
    .attr('x2', xScale(0))
    .attr('y1', 0)
    .attr('y2', innerHeight)
    .attr('stroke', '#334155')
    .attr('stroke-dasharray', '4,4')

  // Points
  g.selectAll('circle')
    .data(props.data)
    .enter()
    .append('circle')
    .attr('cx', d => xScale(d.pc1))
    .attr('cy', d => yScale(d.pc2))
    .attr('r', 8)
    .attr('fill', d => colorScale(d.group))
    .attr('stroke', '#fff')
    .attr('stroke-width', 1)
    .style('cursor', 'pointer')
    .on('click', (event, d) => emit('pointClick', d))
    .on('mouseover', function() {
      d3.select(this).attr('r', 10)
    })
    .on('mouseout', function() {
      d3.select(this).attr('r', 8)
    })

  // Labels
  g.selectAll('.label')
    .data(props.data)
    .enter()
    .append('text')
    .attr('class', 'label')
    .attr('x', d => xScale(d.pc1) + 12)
    .attr('y', d => yScale(d.pc2) + 4)
    .attr('fill', '#94a3b8')
    .attr('font-size', '11px')
    .text(d => d.name)

  // Legend
  const legend = svg
    .append('g')
    .attr('transform', `translate(${width - margin.right + 10}, ${margin.top})`)

  groups.forEach((group, i) => {
    const item = legend.append('g')
      .attr('transform', `translate(0, ${i * 25})`)

    item.append('circle')
      .attr('r', 6)
      .attr('fill', colorScale(group))

    item.append('text')
      .attr('x', 12)
      .attr('y', 4)
      .attr('fill', '#94a3b8')
      .attr('font-size', '12px')
      .text(group)
  })
}

onMounted(renderChart)
watch(() => props.data, renderChart, { deep: true })
</script>

<template>
  <div class="pca-plot">
    <svg ref="svgRef" :width="width" :height="height"></svg>
  </div>
</template>

<style scoped>
.pca-plot {
  display: flex;
  justify-content: center;
}

svg {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}
</style>
