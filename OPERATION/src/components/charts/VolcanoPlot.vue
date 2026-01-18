<script setup>
import { ref, onMounted, watch } from 'vue'
import * as d3 from 'd3'

const props = defineProps({
  data: {
    type: Array,
    default: () => []
  },
  logFcThreshold: {
    type: Number,
    default: 1
  },
  pValueThreshold: {
    type: Number,
    default: 0.05
  }
})

const emit = defineEmits(['pointClick'])
const svgRef = ref(null)

const width = 600
const height = 400
const margin = { top: 20, right: 30, bottom: 50, left: 60 }

function renderChart() {
  if (!svgRef.value || props.data.length === 0) return

  const svg = d3.select(svgRef.value)
  svg.selectAll('*').remove()

  const innerWidth = width - margin.left - margin.right
  const innerHeight = height - margin.top - margin.bottom

  const g = svg
    .append('g')
    .attr('transform', `translate(${margin.left},${margin.top})`)

  // Scales
  const xExtent = d3.extent(props.data, d => d.log2FoldChange)
  const xScale = d3.scaleLinear()
    .domain([Math.min(xExtent[0], -5), Math.max(xExtent[1], 5)])
    .range([0, innerWidth])

  const yExtent = d3.extent(props.data, d => -Math.log10(d.pvalue))
  const yScale = d3.scaleLinear()
    .domain([0, Math.max(yExtent[1], 10)])
    .range([innerHeight, 0])

  // Axes
  g.append('g')
    .attr('transform', `translate(0,${innerHeight})`)
    .call(d3.axisBottom(xScale))
    .selectAll('text')
    .attr('fill', '#94a3b8')

  g.append('g')
    .call(d3.axisLeft(yScale))
    .selectAll('text')
    .attr('fill', '#94a3b8')

  // Axis labels
  g.append('text')
    .attr('x', innerWidth / 2)
    .attr('y', innerHeight + 40)
    .attr('text-anchor', 'middle')
    .attr('fill', '#94a3b8')
    .text('log₂(Fold Change)')

  g.append('text')
    .attr('transform', 'rotate(-90)')
    .attr('x', -innerHeight / 2)
    .attr('y', -45)
    .attr('text-anchor', 'middle')
    .attr('fill', '#94a3b8')
    .text('-log₁₀(p-value)')

  // Threshold lines
  const negLogPThreshold = -Math.log10(props.pValueThreshold)

  // Horizontal line
  g.append('line')
    .attr('x1', 0)
    .attr('x2', innerWidth)
    .attr('y1', yScale(negLogPThreshold))
    .attr('y2', yScale(negLogPThreshold))
    .attr('stroke', '#64748b')
    .attr('stroke-dasharray', '4,4')

  // Vertical lines
  g.append('line')
    .attr('x1', xScale(-props.logFcThreshold))
    .attr('x2', xScale(-props.logFcThreshold))
    .attr('y1', 0)
    .attr('y2', innerHeight)
    .attr('stroke', '#64748b')
    .attr('stroke-dasharray', '4,4')

  g.append('line')
    .attr('x1', xScale(props.logFcThreshold))
    .attr('x2', xScale(props.logFcThreshold))
    .attr('y1', 0)
    .attr('y2', innerHeight)
    .attr('stroke', '#64748b')
    .attr('stroke-dasharray', '4,4')

  // Points
  g.selectAll('circle')
    .data(props.data)
    .enter()
    .append('circle')
    .attr('cx', d => xScale(d.log2FoldChange))
    .attr('cy', d => yScale(-Math.log10(d.pvalue)))
    .attr('r', 4)
    .attr('fill', d => {
      const isSignificant = d.pvalue < props.pValueThreshold
      const isUp = d.log2FoldChange > props.logFcThreshold
      const isDown = d.log2FoldChange < -props.logFcThreshold

      if (isSignificant && isUp) return '#10b981'
      if (isSignificant && isDown) return '#ef4444'
      return '#64748b'
    })
    .attr('opacity', 0.7)
    .style('cursor', 'pointer')
    .on('click', (event, d) => {
      emit('pointClick', d)
    })
    .on('mouseover', function() {
      d3.select(this).attr('r', 6).attr('opacity', 1)
    })
    .on('mouseout', function() {
      d3.select(this).attr('r', 4).attr('opacity', 0.7)
    })
}

onMounted(renderChart)
watch(() => props.data, renderChart)
</script>

<template>
  <div class="volcano-plot">
    <svg ref="svgRef" :width="width" :height="height"></svg>
    <div class="legend">
      <span class="legend-item up">● Upregulated</span>
      <span class="legend-item down">● Downregulated</span>
      <span class="legend-item ns">● Not significant</span>
    </div>
  </div>
</template>

<style scoped>
.volcano-plot {
  display: flex;
  flex-direction: column;
  align-items: center;
}

svg {
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.legend {
  display: flex;
  gap: 1.5rem;
  margin-top: 1rem;
  font-size: 0.875rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.legend-item.up { color: #10b981; }
.legend-item.down { color: #ef4444; }
.legend-item.ns { color: #64748b; }
</style>
