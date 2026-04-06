<script setup>
import { computed } from 'vue';

const props = defineProps({
  data: { type: Array, required: true },   // array of numbers
  width: { type: Number, default: 80 },
  height: { type: Number, default: 24 },
  color: { type: String, default: '#6366f1' },
  filled: { type: Boolean, default: true },
});

const pathD = computed(() => {
  const pts = props.data;
  if (!pts.length) return '';

  const max = Math.max(...pts);
  const min = Math.min(...pts);
  const range = max - min || 1;
  const stepX = props.width / (pts.length - 1 || 1);
  const pad = 2;
  const usableH = props.height - pad * 2;

  const points = pts.map((v, i) => {
    const x = i * stepX;
    const y = pad + usableH - ((v - min) / range) * usableH;
    return `${x},${y}`;
  });

  return 'M' + points.join(' L');
});

const fillD = computed(() => {
  if (!props.filled || !props.data.length) return '';
  return `${pathD.value} L${props.width},${props.height} L0,${props.height} Z`;
});
</script>

<template>
  <svg
    :viewBox="`0 0 ${props.width} ${props.height}`"
    :width="props.width"
    :height="props.height"
    class="sparkline"
  >
    <path
      v-if="props.filled"
      :d="fillD"
      :fill="`${props.color}20`"
    />
    <path
      :d="pathD"
      fill="none"
      :stroke="props.color"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>

<style scoped>
.sparkline {
  display: inline-block;
  vertical-align: middle;
}
</style>
