<script setup>defineProps({ tone:{type:String,default:'clay'} });</script>
<template><span class="badge" :class="`badge--${tone}`"><slot /></span></template>
<style scoped>.badge{display:inline-flex;padding:4px 8px;border-radius:5px;font:700 10px var(--ec-font-mono);text-transform:uppercase;letter-spacing:.05em}.badge--clay{color:var(--ec-clay);background:rgba(198,122,69,.14)}.badge--red{color:var(--ec-red);background:rgba(232,72,44,.1)}</style>
