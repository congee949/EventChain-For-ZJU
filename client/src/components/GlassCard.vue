<script setup>
import { useGlassHighlight } from '../composables/useGlassHighlight.js';

const props = defineProps({
  variant: {
    type: String,
    default: 'regular',
    validator: (v) => ['dense', 'regular', 'subtle'].includes(v),
  },
  hoverable: {
    type: Boolean,
    default: true,
  },
  padding: {
    type: String,
    default: '24px',
  },
});

const { elementRef } = useGlassHighlight();

const classMap = {
  dense: 'glass-dense',
  regular: 'glass',
  subtle: 'glass-subtle',
};
</script>

<template>
  <div
    ref="elementRef"
    :class="[classMap[props.variant], { 'no-hover': !props.hoverable }]"
    :style="{ padding: props.padding }"
    class="glass-card"
  >
    <slot />
  </div>
</template>

<style scoped>
.glass-card {
  overflow: hidden;
}

.glass-card.no-hover:hover {
  transform: none;
  box-shadow:
    inset 0 0.5px 0 rgba(255, 255, 255, 0.5),
    var(--shadow-card);
}
</style>
