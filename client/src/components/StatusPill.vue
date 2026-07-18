<script setup>
import { computed } from 'vue';
const props=defineProps({status:{type:String,default:'closed'},label:{type:String,default:''},theme:{type:String,default:'light'}});
const state=computed(()=>props.status.toLowerCase());
const glyph=computed(()=>({open:'●',waiting:'◷',pending:'◷',won:'★',lost:'◌',closed:'◌'}[state.value]||'◌'));
</script>
<template><span class="status-pill" :class="[`status-pill--${state}`,`status-pill--${theme}`]"><i>{{ glyph }}</i>{{ label || status }}</span></template>
<style scoped>.status-pill{display:inline-flex;align-items:center;gap:6px;border-radius:var(--ec-r-pill);padding:5px 10px;background:rgba(110,97,82,.09);color:var(--ec-muted);font:700 10px var(--ec-font-mono);text-transform:uppercase;white-space:nowrap}.status-pill i{font-style:normal}.status-pill--open,.status-pill--won{color:var(--ec-open);background:rgba(72,132,63,.1)}.status-pill--open i{animation:ecPulse 1.5s infinite}.status-pill--waiting,.status-pill--pending{color:var(--ec-pending);background:rgba(255,178,36,.12)}.status-pill--lost{color:var(--ec-danger);background:rgba(180,35,24,.08)}@keyframes ecPulse{50%{opacity:.35}}</style>
