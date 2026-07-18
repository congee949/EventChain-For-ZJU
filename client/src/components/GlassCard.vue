<script setup>

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

const classMap = { dense:'ec-card--dense', regular:'ec-card--regular', subtle:'ec-card--subtle' };
</script>

<template>
  <div
    :class="[classMap[props.variant], { 'no-hover': !props.hoverable }]"
    :style="{ padding: props.padding }"
    class="glass-card"
  >
    <slot />
  </div>
</template>

<style scoped>
.glass-card{overflow:hidden;background:var(--ec-card);border:1px solid var(--ec-line);border-radius:var(--ec-r-card);box-shadow:none}
.glass-card:not(.no-hover):hover{background:var(--ec-inset)}
.ec-card--subtle{background:var(--ec-inset)}.ec-card--dense{background:var(--ec-card-white);border-color:var(--ec-ink)}
</style>
