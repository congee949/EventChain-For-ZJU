<script setup>
import { ref, watch, onMounted } from 'vue';
import QRCodeLib from 'qrcode';

const props = defineProps({
  value: { type: String, required: true },
  size: { type: Number, default: 200 },
});

const canvasRef = ref(null);

async function render() {
  if (!canvasRef.value || !props.value) return;
  await QRCodeLib.toCanvas(canvasRef.value, props.value, {
    width: props.size,
    margin: 2,
    color: { dark: '#0F2A43', light: '#FFFFFF' },
  });
}

onMounted(render);
watch(() => props.value, render);
watch(() => props.size, render);
</script>

<template>
  <canvas ref="canvasRef" class="qrcode-canvas" />
</template>

<style scoped>
.qrcode-canvas {
  border-radius: var(--radius-sm);
}
</style>
