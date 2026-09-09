<script setup>
import { computed, onMounted } from 'vue';
import EcButton from '../components/EcButton.vue';
import SeasonBadgeCard from '../components/SeasonBadgeCard.vue';
import { useBadgesStore } from '../stores/badges.js';
import { isVisibleBadgeSeries } from '../utils/badgeVisibility.js';

const badges = useBadgesStore();
const publishedSeries = computed(() => badges.series.filter(isVisibleBadgeSeries));
async function refresh() {
  try { await badges.loadSeriesList(); } catch { /* The page renders the store error. */ }
}
onMounted(refresh);
</script>

<template>
  <div class="badge-gallery-page">
    <header class="badge-hero">
      <p class="eyebrow">ON-CHAIN MEMENTOS</p>
      <h1>赛季徽章</h1>
      <p>记录你亲自参与过的校园活动。每个系列有激活后不可提高的固定发行上限，编号只代表链上登记顺序。</p>
      <div class="non-financial" role="note">不可交易 · 不可兑换 · 无现金价值</div>
    </header>

    <div v-if="badges.loading" class="gallery-state" role="status" aria-live="polite">正在读取链上系列…</div>
    <section v-else-if="badges.error" class="gallery-state gallery-state--error" role="alert">
      <h2>系列暂时无法加载</h2><p>{{ badges.error }}</p><EcButton variant="outline" @click="refresh">重新加载</EcButton>
    </section>
    <section v-else-if="!publishedSeries.length" class="gallery-state">
      <h2>尚无赛季徽章</h2><p>系列发布后会显示在这里。</p>
    </section>
    <section v-else class="badge-grid" aria-label="赛季徽章系列">
      <SeasonBadgeCard v-for="item in publishedSeries" :key="item.seriesId" :series="item" />
    </section>

    <footer class="gallery-footnote">
      <strong>关于“限量”</strong>
      <p>固定上限用于让发行规则可审计，不代表稀有度、价格或投资收益。系列开放期间不显示发行进度、剩余量或领取倒计时。</p>
    </footer>
  </div>
</template>

<style scoped>
.badge-gallery-page{display:grid;gap:24px;padding:30px}
.badge-hero{display:grid;max-width:800px;padding:10px 0 22px;border-bottom:2px solid var(--ec-ink)}
.badge-hero h1{font-size:clamp(44px,7vw,68px);letter-spacing:-.03em}
.badge-hero>p:last-of-type{max-width:690px;margin-top:12px;color:var(--ec-ink-2)}
.non-financial{width:max-content;max-width:100%;margin-top:16px;padding:7px 11px;border:1px solid var(--ec-red);border-radius:var(--ec-r-pill);color:var(--ec-red);font:700 10px var(--ec-font-mono);letter-spacing:.05em}
.badge-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}
.gallery-state{display:grid;justify-items:center;gap:12px;padding:56px 20px;border:1px dashed var(--ec-line);border-radius:var(--ec-r-card);color:var(--ec-muted);text-align:center}
.gallery-state h2{color:var(--ec-ink);font-size:28px}.gallery-state--error{border-color:rgba(180,35,24,.35)}
.gallery-footnote{padding:18px 20px;border-radius:var(--ec-r-card);background:var(--ec-dark);color:var(--ec-cream)}
.gallery-footnote strong{font:800 18px var(--ec-font-display);text-transform:uppercase}.gallery-footnote p{max-width:750px;margin-top:5px;color:var(--ec-cream-4);font-size:12px}
@media(max-width:820px){.badge-grid{grid-template-columns:1fr}}
@media(max-width:600px){.badge-gallery-page{padding:20px 16px 40px}.badge-hero h1{font-size:46px}.non-financial{font-size:9px}}
@media(prefers-reduced-motion:reduce){.badge-gallery-page *{scroll-behavior:auto!important;transition:none!important}}
</style>
