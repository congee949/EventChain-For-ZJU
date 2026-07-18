<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { ElMessage } from 'element-plus';
import GlassCard from '../components/GlassCard.vue';
import CountdownTimer from '../components/CountdownTimer.vue';
import QRCode from '../components/QRCode.vue';
import { useActivityV2Store } from '../stores/activityV2.js';
import { useAuthStore } from '../stores/auth.js';

const store = useActivityV2Store();
const auth = useAuthStore();
const clock = ref(Date.now());
const busy = ref('');
const timer = setInterval(() => { clock.value = Date.now(); }, 1000);
onBeforeUnmount(() => clearInterval(timer));
onMounted(async () => { await store.refresh(); await store.hydrateMine(); });

const activeActivities = computed(() => store.activities.filter((item) => !['CANCELLED', 'COMPLETED'].includes(item.status)));
const applications = computed(() => activeActivities.value.filter((item) => store.applications[item.id]).map((item) => ({ activity: item, application: store.applications[item.id] })));
const tickets = computed(() => activeActivities.value.filter((item) => store.tickets[item.id]).map((item) => ({ activity: item, ticket: store.tickets[item.id] })));
const canApply = (item) => auth.user?.role === 'student' && item.status === 'APPLICATION_OPEN' && new Date(item.applicationCloseAt).getTime() > Date.now() && !store.applications[item.id];
const qrProof = (id) => { clock.value; return store.qrProof(id); };
async function apply(id) { busy.value = id; try { await store.apply(id); ElMessage.success('申请已写入私有账本'); } finally { busy.value = ''; } }
async function claim(id) { busy.value = id; try { await store.claim(id); ElMessage.success('票据已领取；秘密只保存在本浏览器'); } finally { busy.value = ''; } }
</script>

<template>
  <div class="ticket-hall">
    <section class="ticket-hero"><div><p>COMMIT–REVEAL LOTTERY</p><h1>活动票务大厅</h1><span>报名不是先到先得。验证者预先承诺种子，截止后由组织者揭示并确定中签名单。</span></div><div class="hero-count"><strong>{{ activeActivities.length }}</strong><small>进行中活动</small></div></section>

    <section class="section">
      <div class="section-title"><div><p>AVAILABLE</p><h2>可以报名的活动</h2></div><span>申请记录仅本人可见</span></div>
      <div class="ticket-events-grid">
        <GlassCard v-for="item in activeActivities" :key="item.id" class="ticket-event-card">
          <div class="te-header"><span class="category-pill">{{ item.categoryId }}</span><span class="status-pill">{{ item.status }}</span></div>
          <h3>{{ item.title }}</h3>
          <div class="capacity"><div><b>{{ item.applicationCount }}</b><small>已申请</small></div><div><b>{{ item.capacity }}</b><small>容量</small></div></div>
          <div class="time-row"><span>报名截止</span><CountdownTimer :target-time="item.applicationCloseAt" /></div>
          <p class="start-time">活动开始：{{ new Date(item.startsAt).toLocaleString('zh-CN') }}</p>
          <button v-if="canApply(item)" class="apply-btn" :disabled="busy === item.id" @click="apply(item.id)">{{ busy === item.id ? '链上提交中…' : '提交抽签申请' }}</button>
          <div v-else-if="store.applications[item.id]" class="application-state"><span>我的申请</span><b>{{ store.applications[item.id].status }}</b></div>
          <button v-else class="closed-btn" disabled>{{ auth.user?.role !== 'student' ? '仅学生可以报名' : new Date(item.applicationCloseAt).getTime() <= Date.now() ? '报名已截止，等待链上推进' : '当前不可报名' }}</button>
        </GlassCard>
      </div>
      <GlassCard v-if="!store.loading && !activeActivities.length" :hoverable="false" class="empty-card">暂无活动</GlassCard>
    </section>

    <div class="lower-grid">
      <section class="section">
        <div class="section-title"><div><p>MY APPLICATIONS</p><h2>我的申请</h2></div></div>
        <div class="application-list">
          <GlassCard v-for="entry in applications" :key="entry.activity.id" variant="subtle" padding="16px" :hoverable="false" class="app-row">
            <span class="app-icon">{{ entry.application.status === 'WON' ? '★' : entry.application.status === 'LOST' ? '◌' : '◷' }}</span>
            <div><b>{{ entry.activity.title }}</b><small>{{ entry.activity.id }}</small></div>
            <span :class="['app-status', entry.application.status.toLowerCase()]">{{ entry.application.status }}</span>
            <button v-if="entry.application.status === 'WON' && !store.tickets[entry.activity.id]" :disabled="busy === entry.activity.id" @click="claim(entry.activity.id)">领取票据</button>
          </GlassCard>
          <p v-if="!applications.length" class="empty-copy">暂无申请记录</p>
        </div>
      </section>

      <section class="section">
        <div class="section-title"><div><p>PRIVATE TICKETS</p><h2>我的动态票据</h2></div></div>
        <div class="tickets-grid">
          <GlassCard v-for="entry in tickets" :key="entry.ticket.ticketId" class="ticket-card" :hoverable="false">
            <div><span class="status-pill">{{ entry.ticket.status }}</span><h3>{{ entry.activity.title }}</h3><p>{{ entry.ticket.ticketId }}</p></div>
            <QRCode v-if="qrProof(entry.activity.id)" :value="qrProof(entry.activity.id)" :size="150" />
            <small>二维码每 30 秒变化，包含本机秘密，请勿转发截图</small>
          </GlassCard>
          <p v-if="!tickets.length" class="empty-copy">中签并领取后，动态二维码会显示在这里</p>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.ticket-hall{padding:30px 30px 48px}.ticket-hero{display:flex;justify-content:space-between;align-items:end;padding:0 0 24px;border-bottom:2px solid var(--ec-ink);margin-bottom:28px}.ticket-hero p,.section-title p{color:var(--ec-red);font:600 10px var(--ec-font-mono);letter-spacing:.18em;margin-bottom:5px}.ticket-hero h1{font-size:clamp(42px,6vw,56px);letter-spacing:-.01em}.ticket-hero span{display:block;max-width:680px;color:var(--ec-ink-2);line-height:1.7;margin-top:12px}.hero-count{text-align:right;min-width:120px}.hero-count strong,.hero-count small{display:block}.hero-count strong{font:800 52px/1 var(--ec-font-display);color:var(--ec-orange)}.hero-count small{color:var(--ec-faint);font:500 9px var(--ec-font-mono);text-transform:uppercase}.section{margin-bottom:32px}.section-title{display:flex;align-items:end;justify-content:space-between;margin-bottom:16px;padding-bottom:9px;border-bottom:2px solid var(--ec-ink)}.section-title h2{font-size:22px}.section-title>span{color:var(--ec-faint);font:500 10px var(--ec-font-mono)}.ticket-events-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}.ticket-event-card{padding:21px!important}.te-header{display:flex;justify-content:space-between}.category-pill,.status-pill{border-radius:5px;padding:4px 9px;font:700 9px var(--ec-font-mono);background:rgba(198,122,69,.14);color:var(--ec-clay);text-transform:uppercase}.status-pill{border-radius:var(--ec-r-pill);color:var(--ec-open);background:rgba(72,132,63,.1)}.ticket-event-card h3{font-size:21px;margin:14px 0}.capacity{display:grid;grid-template-columns:1fr 1fr;gap:8px}.capacity div{padding:11px;border-radius:10px;background:var(--ec-inset)}.capacity b,.capacity small{display:block}.capacity b{font:700 24px var(--ec-font-display)}.capacity small{color:var(--ec-faint);font:500 9px var(--ec-font-mono);text-transform:uppercase}.time-row{display:flex;justify-content:space-between;align-items:center;margin:14px 0 5px;font-size:11px;color:var(--ec-faint)}.start-time{color:var(--ec-muted);font:500 10px var(--ec-font-mono);margin-bottom:15px}.apply-btn,.closed-btn{width:100%;border:0;border-radius:11px;padding:11px;font:700 15px var(--ec-font-display);text-transform:uppercase}.apply-btn{color:var(--ec-ink);background:var(--ec-orange);cursor:pointer}.apply-btn:hover{background:var(--ec-orange-hi)}.apply-btn:disabled,.closed-btn{opacity:.5}.application-state{display:flex;justify-content:space-between;padding:11px;border-radius:11px;background:rgba(255,178,36,.1);font-size:12px}.application-state b{color:var(--ec-pending)}.lower-grid{display:grid;grid-template-columns:1fr 1fr;gap:24px}.application-list,.tickets-grid{display:grid;gap:10px}.app-row{display:grid;grid-template-columns:auto 1fr auto auto;gap:12px;align-items:center}.app-icon{color:var(--ec-orange);font-family:var(--ec-font-mono)}.app-row small{display:block;color:var(--ec-faint);margin-top:3px;font-family:var(--ec-font-mono)}.app-row button{border:0;border-radius:8px;padding:7px 10px;background:var(--ec-ink);color:var(--ec-cream);cursor:pointer}.app-status{font:700 10px var(--ec-font-mono)}.app-status.pending{color:var(--ec-pending)}.app-status.won{color:var(--ec-open)}.app-status.lost{color:var(--ec-faint)}.ticket-card{display:grid;grid-template-columns:1fr auto;gap:16px;align-items:center;padding:20px!important;background:var(--ec-card-white);border-color:var(--ec-ink)}.ticket-card h3{margin:9px 0 3px}.ticket-card p{color:var(--ec-faint);font:500 9px var(--ec-font-mono);word-break:break-all}.ticket-card small{grid-column:1/-1;color:var(--ec-muted);font-size:10px}.empty-copy,.empty-card{color:var(--ec-faint);text-align:center;padding:22px}
@media(max-width:950px){.ticket-events-grid{grid-template-columns:repeat(2,1fr)}.lower-grid{grid-template-columns:1fr}}@media(max-width:620px){.ticket-hall{padding:22px 16px}.ticket-hero{align-items:start}.hero-count{display:none}.ticket-events-grid{grid-template-columns:1fr}.app-row{grid-template-columns:auto 1fr auto}.app-row button{grid-column:2/-1}.ticket-card{grid-template-columns:1fr}}
</style>
