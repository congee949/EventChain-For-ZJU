<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import api, { idempotencyHeaders } from '../api/index.js';
import GlassCard from '../components/GlassCard.vue';
import { formatAmount, useFinanceStore } from '../stores/finance.js';
import { useActivityV2Store } from '../stores/activityV2.js';
import { useAuthStore } from '../stores/auth.js';

const auth = useAuthStore(); const finance = useFinanceStore(); const activityStore = useActivityV2Store();
const role = computed(() => auth.user?.role || '');
const marketForm = reactive({ marketId:'',eventId:'',categoryId:'',labels:'主队胜,平局,客队胜',closeAt:'',marketCap:4000 });
const activityForm = reactive({ id:'',categoryId:'',title:'',capacity:100,applicationCloseAt:'',startsAt:'',endsAt:'' });
const serviceForm = reactive({ offerId:'',categoryId:'',offerType:'VENUE',title:'',price:50,cancellationBps:15000,guaranteeBudget:500,bonusBudget:300,inventory:20 });
const action = reactive({ marketId:'',outcomeId:'',evidence:'',activityId:'',seed:'',activityStatus:'' });
const claimReport = ref(null); const busy = ref(false);
const canCreate = computed(() => role.value === 'organizer');
const canManageMarket = computed(() => ['organizer','admin'].includes(role.value));
const canVerify = computed(() => ['verifier','admin'].includes(role.value));
const canOperate = computed(() => ['operator','admin'].includes(role.value));
const canArbitrate = computed(() => role.value === 'arbitrator');
const toISO = (value, label) => { if (!value) throw new Error(`请选择${label}`); return new Date(value).toISOString(); };

async function refresh() { await Promise.all([finance.refresh(), activityStore.refresh()]); marketForm.categoryId ||= finance.categories[0]?.id || ''; activityForm.categoryId ||= marketForm.categoryId; serviceForm.categoryId ||= marketForm.categoryId; }
onMounted(refresh);
async function run(fn, success='链上操作成功') { busy.value=true; try { const result=await fn(); ElMessage.success(success); await refresh(); return result; } catch(error) { if (error instanceof Error) ElMessage.error(error.message); throw error; } finally { busy.value=false; } }
async function createMarket(){const outcomes=marketForm.labels.split(',').map((label,index)=>({id:`o${index+1}`,label:label.trim()})).filter(x=>x.label);await run(()=>api.post('/finance/markets',{...marketForm,outcomes,closeAt:toISO(marketForm.closeAt,'锁盘时间'),stakeBucket:'BONUS'}),'B_bonus 市场已创建');}
async function createActivity(){await run(()=>api.post('/activities',{...activityForm,applicationCloseAt:toISO(activityForm.applicationCloseAt,'报名截止时间'),startsAt:toISO(activityForm.startsAt,'开始时间'),endsAt:toISO(activityForm.endsAt,'结束时间')}),'活动草稿已创建');}
async function createService(){await run(()=>api.post('/finance/offers',{...serviceForm},{headers:idempotencyHeaders('offer')}),'服务商品已创建并锁定赔付预算');}
async function post(url,body={}){return run(()=>api.post(url,body,{headers:idempotencyHeaders('ops')}));}
async function processClaims(){claimReport.value=await post('/finance/admin/claims/process');}
</script>

<template>
  <div class="admin-page">
    <section class="admin-hero"><div><p>ROLE-BOUND OPERATIONS</p><h1>运营控制台</h1><span>当前角色：<b>{{ role }}</b>。按钮可见性只是交互提示，最终权限由 Fabric 证书属性和链码再次校验。</span></div><div class="network-state"><i></i><span>V2 NETWORK</span><b>finance · activity</b></div></section>
    <div class="overview-grid"><GlassCard :hoverable="false"><strong>{{ finance.markets.length }}</strong><span>预测市场</span></GlassCard><GlassCard :hoverable="false"><strong>{{ activityStore.activities.length }}</strong><span>活动</span></GlassCard><GlassCard :hoverable="false"><strong>{{ finance.offers.length }}</strong><span>服务商品</span></GlassCard><GlassCard :hoverable="false"><strong>{{ role }}</strong><span>证书角色</span></GlassCard></div>

    <section v-if="canCreate" class="section"><div class="section-heading"><div><p>CREATE</p><h2>创建业务对象</h2></div><span>组织者保证金与预算在链码中校验</span></div>
      <div class="form-grid">
        <GlassCard :hoverable="false"><h3>预测市场</h3><div class="form-stack"><el-input v-model="marketForm.marketId" placeholder="market ID"/><el-input v-model="marketForm.eventId" placeholder="关联 event / activity ID"/><el-select v-model="marketForm.categoryId"><el-option v-for="c in finance.categories" :key="c.id" :label="c.name" :value="c.id"/></el-select><el-input v-model="marketForm.labels" placeholder="结果标签，逗号分隔"/><el-date-picker v-model="marketForm.closeAt" type="datetime" placeholder="锁盘时间"/><el-input-number v-model="marketForm.marketCap" :min="2000"/><el-button type="primary" :loading="busy" @click="createMarket">创建 B_bonus 市场</el-button></div></GlassCard>
        <GlassCard :hoverable="false"><h3>活动与抽签</h3><div class="form-stack"><el-input v-model="activityForm.id" placeholder="activity ID"/><el-input v-model="activityForm.title" placeholder="活动标题"/><el-select v-model="activityForm.categoryId"><el-option v-for="c in finance.categories" :key="c.id" :label="c.name" :value="c.id"/></el-select><el-input-number v-model="activityForm.capacity" :min="1"/><el-date-picker v-model="activityForm.applicationCloseAt" type="datetime" placeholder="报名截止"/><el-date-picker v-model="activityForm.startsAt" type="datetime" placeholder="开始时间"/><el-date-picker v-model="activityForm.endsAt" type="datetime" placeholder="结束时间"/><el-button type="primary" :loading="busy" @click="createActivity">创建活动草稿</el-button></div></GlassCard>
        <GlassCard :hoverable="false"><h3>服务兑换商品</h3><div class="form-stack"><el-input v-model="serviceForm.offerId" placeholder="offer ID"/><el-input v-model="serviceForm.title" placeholder="服务标题"/><el-select v-model="serviceForm.categoryId"><el-option v-for="c in finance.categories" :key="c.id" :label="c.name" :value="c.id"/></el-select><el-select v-model="serviceForm.offerType"><el-option label="场馆" value="VENUE"/><el-option label="器材" value="EQUIPMENT"/><el-option label="活动通行" value="EVENT_PASS"/></el-select><el-input-number v-model="serviceForm.price" :min=".000001"/><el-input-number v-model="serviceForm.inventory" :min="1"/><el-button type="primary" :loading="busy" @click="createService">创建并预存保证金</el-button></div></GlassCard>
      </div>
    </section>

    <section class="section"><div class="section-heading"><div><p>CONTROL PLANE</p><h2>状态、结果与结算</h2></div></div>
      <div class="action-grid">
        <GlassCard :hoverable="false"><h3>市场动作</h3><div class="form-stack"><el-select v-model="action.marketId" filterable placeholder="选择市场"><el-option v-for="m in finance.markets" :key="m.marketId" :label="`${m.marketId} · ${m.status}`" :value="m.marketId"/></el-select><el-input v-model="action.outcomeId" placeholder="outcome ID / void"/><el-input v-model="action.evidence" type="textarea" :rows="2" placeholder="证据或挑战摘要"/><div class="button-grid"><el-button v-if="canManageMarket" @click="post(`/finance/markets/${action.marketId}/open`)">开放</el-button><el-button v-if="canManageMarket" @click="post(`/finance/markets/${action.marketId}/lock`)">锁盘</el-button><el-button v-if="canCreate" @click="post(`/finance/markets/${action.marketId}/result`,{outcomeId:action.outcomeId,evidence:action.evidence})">提交结果</el-button><el-button v-if="canVerify" @click="post(`/finance/markets/${action.marketId}/confirm`)">二次确认</el-button><el-button v-if="canOperate" @click="post(`/finance/markets/${action.marketId}/finalize`)">临时结算</el-button><el-button v-if="canArbitrate" @click="post(`/finance/markets/${action.marketId}/vote`,{outcomeId:action.outcomeId})">仲裁投票</el-button></div></div></GlassCard>
        <GlassCard :hoverable="false"><h3>抽签与活动状态</h3><div class="form-stack"><el-select v-model="action.activityId" filterable placeholder="选择活动"><el-option v-for="a in activityStore.activities" :key="a.id" :label="`${a.title} · ${a.status}`" :value="a.id"/></el-select><el-input v-model="action.seed" placeholder="至少 32 字符独立种子"/><el-select v-model="action.activityStatus" placeholder="目标状态"><el-option v-for="s in ['APPLICATION_OPEN','APPLICATION_CLOSED','ACTIVE','COMPLETED','CANCELLED']" :key="s" :label="s" :value="s"/></el-select><div class="button-grid"><el-button v-if="canVerify" @click="post(`/activities/${action.activityId}/draw-commitment`,{seed:action.seed})">承诺种子</el-button><el-button v-if="canCreate" @click="post(`/activities/${action.activityId}/draw`,{seed:action.seed})">揭示并抽签</el-button><el-button v-if="canCreate || role==='admin'" @click="run(()=>api.put(`/activities/${action.activityId}/status`,{status:action.activityStatus}))">更新状态</el-button></div></div></GlassCard>
        <GlassCard v-if="canOperate" :hoverable="false"><h3>分批 Claim Worker</h3><p class="helper">每批扫描 100 个参与者、并发 5，MVCC 冲突自动重试。收益先进入 7 天 Pending 状态。</p><el-button type="primary" :loading="busy" @click="processClaims">处理所有待结算批次</el-button><pre v-if="claimReport">{{ JSON.stringify(claimReport,null,2) }}</pre></GlassCard>
      </div>
    </section>

    <section class="section"><div class="section-heading"><div><p>LIVE OBJECTS</p><h2>当前对象</h2></div></div><div class="object-table"><div v-for="m in finance.markets" :key="m.marketId"><b>{{ m.marketId }}</b><span>{{ m.categoryId }}</span><em>{{ m.status }}</em><span>池 {{ formatAmount(m.displayedPool) }}</span></div><div v-for="a in activityStore.activities" :key="a.id"><b>{{ a.title }}</b><span>{{ a.categoryId }}</span><em>{{ a.status }}</em><span>{{ a.applicationCount }}/{{ a.capacity }}</span></div></div></section>
  </div>
</template>

<style scoped>
.admin-page{padding-bottom:48px}.admin-hero{display:flex;justify-content:space-between;align-items:end;padding:40px 8px 28px}.admin-hero p,.section-heading p{color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.18em}.admin-hero h1{font-size:clamp(38px,6vw,60px);letter-spacing:-.05em}.admin-hero span{display:block;color:var(--color-text-secondary);margin-top:9px}.network-state{text-align:right}.network-state i{display:inline-block;width:8px;height:8px;border-radius:50%;background:var(--color-success);margin-right:7px}.network-state span,.network-state b{font-size:11px}.overview-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:32px}.overview-grid>div{padding:18px!important}.overview-grid strong,.overview-grid span{display:block}.overview-grid strong{font-size:27px}.overview-grid span{color:var(--color-text-tertiary);font-size:11px;margin-top:4px}.section{margin-bottom:34px}.section-heading{display:flex;justify-content:space-between;align-items:end;margin-bottom:15px}.section-heading h2{font-size:23px;margin-top:3px}.section-heading>span{color:var(--color-text-tertiary);font-size:11px}.form-grid,.action-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}.form-grid h3,.action-grid h3{margin-bottom:14px}.form-stack{display:grid;gap:10px}.form-stack :deep(.el-date-editor),.form-stack :deep(.el-input-number),.form-stack :deep(.el-select){width:100%}.button-grid{display:grid;grid-template-columns:1fr 1fr;gap:8px}.helper{color:var(--color-text-secondary);font-size:12px;line-height:1.6;margin-bottom:16px}pre{max-height:230px;background:rgba(15,23,42,.06);padding:12px;border-radius:10px;margin-top:12px}.object-table{display:grid;gap:7px}.object-table>div{display:grid;grid-template-columns:2fr 1fr 1fr 1fr;gap:12px;padding:12px 15px;border-radius:11px;background:rgba(255,255,255,.23);font-size:12px}.object-table em{font-style:normal;color:#087f5b;font-weight:700}.object-table span{color:var(--color-text-secondary)}@media(max-width:980px){.form-grid,.action-grid{grid-template-columns:1fr 1fr}.overview-grid{grid-template-columns:1fr 1fr}}@media(max-width:650px){.admin-hero{align-items:start}.network-state{display:none}.form-grid,.action-grid{grid-template-columns:1fr}.object-table>div{grid-template-columns:1fr 1fr}}
.admin-page{padding:0 30px 48px}.admin-hero{padding-left:0;padding-right:0;border-bottom:2px solid var(--ec-ink);margin-bottom:24px}.overview-grid>div,.form-grid>div,.action-grid>div{background:var(--ec-card);border-color:var(--ec-line)}.overview-grid strong{font-family:var(--ec-font-display)}.object-table>div{background:var(--ec-inset);border-radius:var(--ec-r-field)}@media(max-width:650px){.admin-page{padding:0 16px 40px}}
</style>
