<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import QrScanner from 'qr-scanner';
import api from '../api/index.js';
import GlassCard from '../components/GlassCard.vue';
import { useActivityV2Store } from '../stores/activityV2.js';
import { formatAmount } from '../stores/finance.js';
import { parseCheckInProof } from '../utils/checkInProof.js';

const router = useRouter();
const activityStore = useActivityV2Store();
const videoRef = ref(null);
const fileRef = ref(null);
const manualProof = ref('');
const state = ref('idle');
const message = ref('点击按钮启用摄像头，二维码进入取景框后会自动核验。');
const result = ref(null);
let scanner = null;

const isScanning = computed(() => state.value === 'scanning');
const isVerifying = computed(() => state.value === 'verifying');
const activityName = computed(() => {
  const activityId = result.value?.receipt?.activityId;
  return activityStore.activities.find((item) => item.id === activityId)?.title || activityId || '—';
});

function readableError(error) {
  const text = error?.message || String(error || '');
  if (/permission|denied|notallowed/i.test(text)) return '无法使用摄像头，请允许浏览器访问相机，或改为上传二维码截图。';
  if (/notfound|no camera/i.test(text)) return '未检测到可用摄像头，请改为上传二维码截图。';
  if (/secure|https/i.test(text)) return '摄像头需要 HTTPS 或 localhost 环境，请改用安全地址访问。';
  return text || '扫码失败，请重试。';
}

async function ensureScanner() {
  await nextTick();
  if (scanner || !videoRef.value) return;
  scanner = new QrScanner(videoRef.value, ({ data }) => handleDecoded(data), {
    preferredCamera: 'environment',
    maxScansPerSecond: 8,
    highlightScanRegion: true,
    highlightCodeOutline: true,
    returnDetailedScanResult: true,
    onDecodeError: () => {},
  });
}

async function startCamera() {
  result.value = null;
  state.value = 'starting';
  message.value = '正在请求摄像头权限…';
  try {
    if (!await QrScanner.hasCamera()) throw new Error('No camera found');
    await ensureScanner();
    await scanner.start();
    state.value = 'scanning';
    message.value = '请将学生动态二维码完整放入取景框';
  } catch (error) {
    await scanner?.stop();
    state.value = 'error';
    message.value = readableError(error);
  }
}

async function stopCamera() {
  if (scanner) await scanner.stop();
  if (state.value === 'scanning') {
    state.value = 'idle';
    message.value = '摄像头已暂停。';
  }
}

async function verify(raw) {
  if (isVerifying.value) return;
  let proof;
  try {
    proof = parseCheckInProof(raw);
  } catch (error) {
    await scanner?.stop();
    state.value = 'error';
    message.value = readableError(error);
    return;
  }

  manualProof.value = '';
  state.value = 'verifying';
  message.value = '二维码已识别，正在提交链上核验…';
  await scanner?.stop();
  try {
    result.value = await api.post('/activities/check-in/verify', proof);
    state.value = 'success';
    message.value = result.value.rewardPending
      ? '签到成功，奖励积分暂未发放，请稍后由运营端补发。'
      : '签到成功，奖励积分已发放。';
    ElMessage.success('签到核验成功');
  } catch (error) {
    state.value = 'error';
    message.value = readableError(error);
  }
}

async function handleDecoded(data) {
  if (!isScanning.value) return;
  await verify(data);
}

async function handleFile(event) {
  const file = event.target.files?.[0];
  event.target.value = '';
  if (!file) return;
  result.value = null;
  state.value = 'verifying';
  message.value = '正在识别图片中的二维码…';
  await scanner?.stop();
  try {
    const scan = await QrScanner.scanImage(file, { returnDetailedScanResult: true });
    state.value = 'idle';
    await verify(scan.data);
  } catch (error) {
    state.value = 'error';
    message.value = error instanceof Error && !/No QR code found/i.test(error.message)
      ? readableError(error)
      : '图片中没有识别到二维码，请换一张更清晰的截图。';
  }
}

async function scanNext() {
  result.value = null;
  state.value = 'idle';
  message.value = '准备扫描下一张动态二维码。';
  await startCamera();
}

onMounted(() => activityStore.refresh().catch(() => {}));
onBeforeUnmount(() => scanner?.destroy());
</script>

<template>
  <div class="check-in-page">
    <button class="back-link" @click="router.push('/admin')">← 返回运营控制台</button>

    <section class="check-in-hero">
      <div>
        <p>活动入场核验</p>
        <h1>签到扫码</h1>
        <span>仅核验 EventChain 学生端生成的 30 秒动态二维码。扫描成功后会立即完成票据核销。</span>
      </div>
      <div class="security-note"><b>隐私提示</b><span>二维码秘密仅提交给核验接口，不会显示在页面或签到结果中。</span></div>
    </section>

    <div class="scanner-layout">
      <GlassCard class="scanner-card" padding="20px" :hoverable="false">
        <div class="camera-stage" :class="`camera-stage--${state}`">
          <video ref="videoRef" muted playsinline />
          <div v-if="!isScanning" class="camera-placeholder">
            <span v-if="state === 'success'" class="state-icon state-icon--success">✓</span>
            <span v-else-if="state === 'verifying'" class="state-icon state-icon--loading">↻</span>
            <span v-else class="state-icon">▣</span>
            <b>{{ state === 'success' ? '核验完成' : state === 'verifying' ? '链上核验中' : state === 'starting' ? '正在启动摄像头' : '摄像头尚未启用' }}</b>
          </div>
        </div>

        <div class="scanner-status" :class="`scanner-status--${state}`" role="status" aria-live="polite">
          <i></i><span>{{ message }}</span>
        </div>

        <div class="primary-actions">
          <button v-if="state === 'success'" class="primary-btn" @click="scanNext">继续扫描</button>
          <button v-else-if="isScanning" class="secondary-btn" @click="stopCamera">暂停摄像头</button>
          <button v-else class="primary-btn" :disabled="isVerifying || state === 'starting'" @click="startCamera">启用摄像头扫码</button>
          <button class="secondary-btn" :disabled="isVerifying || state === 'starting'" @click="fileRef?.click()">上传二维码截图</button>
          <input ref="fileRef" class="visually-hidden" type="file" accept="image/*" @change="handleFile">
        </div>
      </GlassCard>

      <aside class="side-column">
        <GlassCard v-if="result" class="result-card" padding="24px" :hoverable="false">
          <div class="result-heading"><span>✓</span><div><p>核验成功</p><h2>{{ activityName }}</h2></div></div>
          <dl>
            <div><dt>票据编号</dt><dd>{{ result.receipt?.ticketId }}</dd></div>
            <div><dt>签到时间</dt><dd>{{ new Date(result.receipt?.checkedInAt).toLocaleString('zh-CN') }}</dd></div>
            <div><dt>活动分类</dt><dd>{{ result.receipt?.categoryId }}</dd></div>
            <div><dt>奖励状态</dt><dd>{{ result.rewardPending ? '签到已确认，奖励待补发' : `已发放 ${formatAmount(result.reward?.amount)} B_bonus` }}</dd></div>
          </dl>
        </GlassCard>

        <GlassCard v-else padding="24px" :hoverable="false">
          <p class="panel-kicker">备用核验方式</p>
          <h2>粘贴二维码内容</h2>
          <p class="helper">摄像头不可用时，可由可信设备读取二维码文本并粘贴。不要保存或转发其中的票据秘密。</p>
          <el-input v-model="manualProof" type="textarea" :rows="5" placeholder='粘贴 {"ticketId": "…", "secret": "…", "timeSlice": …}' />
          <button class="manual-btn" :disabled="isVerifying || !manualProof.trim()" @click="verify(manualProof)">提交核验</button>
        </GlassCard>

        <GlassCard variant="subtle" padding="20px" :hoverable="false" class="steps-card">
          <p class="panel-kicker">操作说明</p>
          <ol><li><b>学生打开票务大厅</b><span>出示“我的动态票据”二维码。</span></li><li><b>运营员扫描</b><span>二维码识别后自动提交，不需手工输入。</span></li><li><b>确认核验结果</b><span>看到成功回执后再放行，并继续扫描下一位。</span></li></ol>
        </GlassCard>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.check-in-page{width:min(1080px,100%);margin:0 auto;padding:clamp(24px,4vw,40px) clamp(20px,4vw,48px) 48px}.back-link{width:fit-content;border:0;background:transparent;color:var(--ec-red);cursor:pointer;font:600 15px var(--ec-font-display);margin-bottom:18px}.check-in-hero{display:flex;justify-content:space-between;gap:30px;align-items:end;padding:28px;border:1px solid var(--ec-line);border-left:5px solid var(--ec-red);border-radius:var(--ec-r-card);background:var(--ec-inset);margin-bottom:22px}.check-in-hero p,.panel-kicker{color:var(--ec-red);font:700 10px var(--ec-font-mono);letter-spacing:.13em;margin-bottom:6px}.check-in-hero h1{font-size:clamp(36px,6vw,50px);letter-spacing:-.03em}.check-in-hero>div>span{display:block;max-width:630px;color:var(--ec-muted);line-height:1.7;margin-top:10px}.security-note{max-width:250px;padding:13px 15px;border-radius:var(--ec-r-field);background:#fff}.security-note b,.security-note span{display:block}.security-note b{font-size:12px}.security-note span{font-size:10px!important;line-height:1.55!important;margin-top:4px!important}.scanner-layout{display:grid;grid-template-columns:minmax(0,1.35fr) minmax(290px,.65fr);gap:18px;align-items:start}.scanner-card,.side-column{display:grid;gap:14px}.camera-stage{position:relative;display:grid;place-items:center;aspect-ratio:4/3;overflow:hidden;border-radius:var(--ec-r-field);background:#111;isolation:isolate}.camera-stage video{width:100%;height:100%;object-fit:cover}.camera-stage :deep(.scan-region-highlight){outline:2px solid #fff!important;border-radius:18px;box-shadow:0 0 0 9999px rgba(0,0,0,.3)}.camera-stage :deep(.code-outline-highlight){stroke:var(--ec-orange-hi)!important;stroke-width:4!important}.camera-placeholder{position:absolute;inset:0;display:grid;place-content:center;justify-items:center;gap:10px;background:var(--ec-inset);color:var(--ec-muted);z-index:2}.state-icon{display:grid;place-items:center;width:62px;height:62px;border-radius:50%;background:#fff;color:var(--ec-red);font:700 28px var(--ec-font-mono)}.state-icon--success{color:var(--ec-open);background:rgba(36,138,91,.1)}.state-icon--loading{animation:spin 1s linear infinite}.scanner-status{display:flex;align-items:center;gap:9px;min-height:44px;padding:10px 12px;border-radius:var(--ec-r-field);background:var(--ec-inset);color:var(--ec-muted);font-size:12px}.scanner-status i{flex:0 0 auto;width:8px;height:8px;border-radius:50%;background:var(--ec-faint)}.scanner-status--scanning i{background:var(--ec-open);box-shadow:0 0 0 4px rgba(36,138,91,.12)}.scanner-status--verifying i{background:var(--ec-amber)}.scanner-status--success{color:var(--ec-open);background:rgba(36,138,91,.08)}.scanner-status--success i{background:var(--ec-open)}.scanner-status--error{color:var(--ec-danger);background:rgba(196,59,59,.08)}.scanner-status--error i{background:var(--ec-danger)}.primary-actions{display:grid;grid-template-columns:1fr 1fr;gap:10px}.primary-btn,.secondary-btn,.manual-btn{border:0;border-radius:11px;padding:12px;font:700 14px var(--ec-font-display);cursor:pointer}.primary-btn,.manual-btn{color:#fff;background:var(--ec-red)}.secondary-btn{color:var(--ec-red);background:rgba(0,113,227,.08)}.primary-btn:disabled,.secondary-btn:disabled,.manual-btn:disabled{opacity:.45;cursor:not-allowed}.side-column{gap:18px}.side-column h2{font-size:23px;margin-bottom:8px}.helper{color:var(--ec-muted);font-size:12px;line-height:1.65;margin:8px 0 14px}.manual-btn{width:100%;margin-top:12px}.result-card{border-color:rgba(36,138,91,.35);background:rgba(36,138,91,.04)}.result-heading{display:flex;gap:12px;align-items:center;padding-bottom:16px;border-bottom:1px solid rgba(36,138,91,.2)}.result-heading>span{display:grid;place-items:center;width:38px;height:38px;border-radius:50%;background:var(--ec-open);color:#fff;font-weight:800}.result-heading p{color:var(--ec-open);font:700 10px var(--ec-font-mono);letter-spacing:.12em}.result-heading h2{margin:2px 0 0}.result-card dl{display:grid;gap:12px;margin-top:16px}.result-card dl>div{display:grid;grid-template-columns:86px 1fr;gap:10px}.result-card dt{color:var(--ec-faint);font:500 10px var(--ec-font-mono)}.result-card dd{min-width:0;color:var(--ec-ink-2);font-size:12px;word-break:break-all}.steps-card ol{display:grid;gap:13px;list-style:none;counter-reset:steps;margin-top:14px}.steps-card li{display:grid;grid-template-columns:27px 1fr;gap:2px 10px;counter-increment:steps}.steps-card li::before{content:counter(steps);grid-row:1/3;display:grid;place-items:center;align-self:start;width:25px;height:25px;border-radius:50%;background:rgba(0,113,227,.1);color:var(--ec-red);font:700 11px var(--ec-font-mono)}.steps-card b{font-size:12px}.steps-card span{color:var(--ec-muted);font-size:11px}.visually-hidden{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:800px){.scanner-layout{grid-template-columns:1fr}.check-in-hero{align-items:start}.security-note{display:none}}
@media(max-width:540px){.check-in-page{padding:22px 16px 40px}.check-in-hero{padding:22px}.primary-actions{grid-template-columns:1fr}.camera-stage{aspect-ratio:1}.result-card dl>div{grid-template-columns:1fr;gap:2px}}
</style>
