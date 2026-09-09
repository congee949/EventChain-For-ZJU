<script setup>
import { onBeforeUnmount, ref, watch } from 'vue';

const props = defineProps({
  src: { type: String, default: '' },
  sha256: { type: String, default: '' },
  alt: { type: String, default: '赛季纪念徽章图案' },
});

const verifiedUrl = ref('');
const state = ref('loading');
let objectUrl = '';
let requestVersion = 0;

function revokeObjectUrl() {
  if (objectUrl) URL.revokeObjectURL(objectUrl);
  objectUrl = '';
  verifiedUrl.value = '';
}

function bytesToHex(buffer) {
  return Array.from(new Uint8Array(buffer), (byte) => byte.toString(16).padStart(2, '0')).join('');
}

async function verifyAsset() {
  const version = ++requestVersion;
  revokeObjectUrl();
  state.value = 'loading';
  if (!props.src || !/^[a-f0-9]{64}$/.test(props.sha256)) {
    state.value = 'unverified';
    return;
  }
  try {
    const response = await fetch(props.src, { credentials: 'same-origin' });
    if (!response.ok) throw new Error('asset unavailable');
    const blob = await response.blob();
    const digest = bytesToHex(await crypto.subtle.digest('SHA-256', await blob.arrayBuffer()));
    if (version !== requestVersion) return;
    if (digest !== props.sha256) {
      state.value = 'mismatch';
      return;
    }
    objectUrl = URL.createObjectURL(blob);
    verifiedUrl.value = objectUrl;
    state.value = 'verified';
  } catch {
    if (version === requestVersion) state.value = 'unverified';
  }
}

watch(() => [props.src, props.sha256], verifyAsset, { immediate: true });
onBeforeUnmount(() => {
  requestVersion += 1;
  revokeObjectUrl();
});
</script>

<template>
  <div class="badge-artwork" :class="`badge-artwork--${state}`">
    <img v-if="state === 'verified'" :src="verifiedUrl" :alt="alt">
    <div v-else class="badge-artwork__placeholder" role="img" :aria-label="`${alt}，内容无法验证`">
      <span aria-hidden="true">EC</span>
      <strong>{{ state === 'loading' ? '正在核验内容' : '内容无法验证' }}</strong>
      <small v-if="state === 'mismatch'">文件与链上哈希不一致</small>
      <small v-else-if="state !== 'loading'">已使用安全占位图</small>
    </div>
    <span v-if="state === 'verified'" class="badge-artwork__verified">内容哈希已核验</span>
  </div>
</template>

<style scoped>
.badge-artwork{position:relative;display:grid;place-items:center;aspect-ratio:1;overflow:hidden;border:1px solid var(--ec-line);border-radius:var(--ec-r-card);background:var(--ec-inset)}
.badge-artwork img{width:100%;height:100%;object-fit:cover}
.badge-artwork__placeholder{display:grid;place-items:center;gap:7px;padding:24px;text-align:center;color:var(--ec-muted)}
.badge-artwork__placeholder>span{display:grid;place-items:center;width:72px;height:72px;border:2px solid currentColor;border-radius:50%;font:800 28px var(--ec-font-display)}
.badge-artwork__placeholder strong{font:800 18px var(--ec-font-display);text-transform:uppercase}
.badge-artwork__placeholder small{font-size:11px}
.badge-artwork__verified{position:absolute;right:10px;bottom:10px;padding:5px 8px;border-radius:var(--ec-r-pill);background:rgba(33,25,19,.86);color:var(--ec-cream);font:700 9px var(--ec-font-mono);letter-spacing:.04em}
</style>
