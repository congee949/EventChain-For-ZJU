<script setup>
import { computed } from 'vue';
import BadgeArtwork from './BadgeArtwork.vue';

const props = defineProps({ series: { type: Object, required: true } });
const statusLabels = { DRAFT: '筹备中', ACTIVE: '开放领取', PAUSED: '暂时停领', CLOSED: '发行结束', FINALIZED: '已最终确认' };
const statusLabel = computed(() => statusLabels[props.series.status] || props.series.status || '未知状态');
</script>

<template>
  <RouterLink class="season-card" :to="`/badges/${encodeURIComponent(series.seriesId)}`" :aria-label="`查看${series.title}详情`">
    <BadgeArtwork :src="series.assetUri" :sha256="series.assetSha256" :alt="`${series.title}图案`" />
    <div class="season-card__body">
      <div class="season-card__meta"><span>{{ series.seasonId }}</span><span :class="`is-${series.status?.toLowerCase()}`">{{ statusLabel }}</span></div>
      <h2>{{ series.title }}</h2>
      <p>{{ series.description }}</p>
      <dl><div><dt>固定发行上限</dt><dd>{{ Number(series.maxSupply || 0).toLocaleString('zh-CN') }} 枚</dd></div><div><dt>发行组织</dt><dd>{{ series.issuerOrganization }}</dd></div></dl>
      <small>不可交易 · 不可兑换 · 无现金价值</small>
    </div>
  </RouterLink>
</template>

<style scoped>
.season-card{display:grid;grid-template-columns:148px 1fr;min-width:0;overflow:hidden;border:1px solid var(--ec-line);border-radius:var(--ec-r-card);background:var(--ec-card);color:var(--ec-ink);text-decoration:none}
.season-card:hover,.season-card:focus-visible{color:var(--ec-ink);border-color:var(--ec-ink);background:var(--ec-card-white)}
.season-card:focus-visible{outline:3px solid rgba(232,72,44,.28);outline-offset:3px}
.season-card :deep(.badge-artwork){width:100%;height:100%;min-width:0;min-height:220px;aspect-ratio:auto;border:0;border-right:1px solid var(--ec-line);border-radius:0}
.season-card__body{display:flex;min-width:0;flex-direction:column;padding:20px}
.season-card__meta{display:flex;align-items:center;justify-content:space-between;gap:10px;color:var(--ec-faint);font:700 9px var(--ec-font-mono);letter-spacing:.1em;text-transform:uppercase}
.season-card__meta span:last-child{padding:4px 7px;border-radius:var(--ec-r-pill);background:rgba(110,97,82,.1)}
.season-card__meta .is-active{color:var(--ec-open);background:rgba(72,132,63,.1)}
.season-card__meta .is-paused{color:var(--ec-pending);background:rgba(154,103,0,.1)}
.season-card h2{margin-top:16px;font-size:28px;line-height:1.04}
.season-card p{margin-top:9px;color:var(--ec-muted);font-size:12px;line-height:1.55}
.season-card dl{display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-top:auto;padding-top:18px}
.season-card dl div{min-width:0;padding:9px;border-radius:var(--ec-r-field);background:var(--ec-inset)}
.season-card dt{color:var(--ec-faint);font:700 8px var(--ec-font-mono);letter-spacing:.06em;text-transform:uppercase}
.season-card dd{overflow-wrap:anywhere;margin-top:3px;font:700 14px var(--ec-font-display)}
.season-card small{margin-top:12px;color:var(--ec-red);font:700 9px var(--ec-font-mono);letter-spacing:.04em}
@media(max-width:600px){.season-card{grid-template-columns:108px 1fr}.season-card :deep(.badge-artwork){min-height:235px}.season-card__body{padding:15px}.season-card h2{font-size:23px}.season-card dl{grid-template-columns:1fr}.season-card p{display:-webkit-box;overflow:hidden;-webkit-box-orient:vertical;-webkit-line-clamp:2}}
</style>
