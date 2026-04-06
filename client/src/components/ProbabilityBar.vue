<script setup>
import { computed } from 'vue';

const props = defineProps({
  probA: { type: Number, required: true },  // 0-1
  labelA: { type: String, default: 'A' },
  labelB: { type: String, default: 'B' },
  colorA: { type: String, default: '#6366f1' },
  colorB: { type: String, default: '#ec4899' },
  height: { type: String, default: '32px' },
});

const pctA = computed(() => Math.round(props.probA * 100));
const pctB = computed(() => 100 - pctA.value);
</script>

<template>
  <div class="probability-bar" :style="{ height: props.height }">
    <div
      class="bar-segment bar-a"
      :style="{
        width: pctA + '%',
        background: `linear-gradient(90deg, ${props.colorA}, ${props.colorA}dd)`,
      }"
    >
      <span v-if="pctA >= 20" class="bar-label">{{ props.labelA }} {{ pctA }}%</span>
    </div>
    <div
      class="bar-segment bar-b"
      :style="{
        width: pctB + '%',
        background: `linear-gradient(90deg, ${props.colorB}dd, ${props.colorB})`,
      }"
    >
      <span v-if="pctB >= 20" class="bar-label">{{ props.labelB }} {{ pctB }}%</span>
    </div>
  </div>
</template>

<style scoped>
.probability-bar {
  display: flex;
  border-radius: 999px;
  overflow: hidden;
  width: 100%;
}

.bar-segment {
  display: flex;
  align-items: center;
  justify-content: center;
  transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
  min-width: 4px;
}

.bar-label {
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}
</style>
