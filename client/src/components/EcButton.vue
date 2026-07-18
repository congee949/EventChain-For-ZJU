<script setup>
defineProps({ variant:{type:String,default:'primary'}, size:{type:String,default:'md'}, disabled:Boolean, type:{type:String,default:'button'} });
</script>
<template><button :type="type" class="ec-button" :class="[`ec-button--${variant}`,`ec-button--${size}`]" :disabled="disabled"><slot /></button></template>
<style scoped>
.ec-button{display:inline-flex;align-items:center;justify-content:center;border:1px solid transparent;border-radius:var(--ec-r-field);padding:0 18px;font:700 16px var(--ec-font-display);letter-spacing:.03em;text-transform:uppercase;cursor:pointer}.ec-button--sm{height:40px}.ec-button--md{height:44px}.ec-button--primary{background:var(--ec-orange);color:var(--ec-ink)}.ec-button--primary:hover{background:var(--ec-orange-hi)}.ec-button--ink{background:var(--ec-ink);color:var(--ec-cream)}.ec-button--outline{border-color:var(--ec-ink);background:transparent;color:var(--ec-ink)}.ec-button--outline:hover,.ec-button--ghost:hover{background:var(--ec-inset)}.ec-button--ghost{background:transparent;color:var(--ec-muted)}.ec-button:active{opacity:.78}.ec-button:disabled{opacity:.45;cursor:not-allowed}
</style>
