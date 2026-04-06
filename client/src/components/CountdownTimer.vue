<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';

const props = defineProps({
  targetTime: { type: String, required: true },  // ISO string
});

const now = ref(Date.now());
let timer = null;

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now(); }, 1000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});

const remaining = computed(() => {
  const diff = new Date(props.targetTime).getTime() - now.value;
  if (diff <= 0) return { days: 0, hours: 0, mins: 0, secs: 0, expired: true };

  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const mins = Math.floor((diff % 3600000) / 60000);
  const secs = Math.floor((diff % 60000) / 1000);

  return { days, hours, mins, secs, expired: false };
});

function pad(n) {
  return String(n).padStart(2, '0');
}
</script>

<template>
  <div class="countdown" :class="{ 'countdown--expired': remaining.expired }">
    <template v-if="remaining.expired">
      <span class="countdown-label">已截止</span>
    </template>
    <template v-else>
      <div v-if="remaining.days > 0" class="countdown-unit">
        <span class="countdown-value">{{ remaining.days }}</span>
        <span class="countdown-label">天</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.hours) }}</span>
        <span class="countdown-label">时</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.mins) }}</span>
        <span class="countdown-label">分</span>
      </div>
      <div class="countdown-unit">
        <span class="countdown-value">{{ pad(remaining.secs) }}</span>
        <span class="countdown-label">秒</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.countdown {
  display: flex;
  align-items: center;
  gap: 4px;
}

.countdown-unit {
  display: flex;
  align-items: baseline;
  gap: 1px;
}

.countdown-value {
  font-size: 16px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
}

.countdown-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.countdown--expired .countdown-label {
  color: var(--color-danger);
  font-weight: 600;
  font-size: 14px;
}
</style>
