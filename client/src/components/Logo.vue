<script setup>
defineProps({ theme:{type:String,default:'light'}, size:{type:[Number,String],default:22} });
</script>
<template><span class="logo" :class="`logo--${theme}`" :style="{fontSize:`${size}px`}"><i></i><b>赛事链</b></span></template>
<style scoped>
.logo{display:inline-flex;align-items:center;gap:.55em;color:var(--ec-dark);font-family:var(--ec-font-display);font-weight:900;line-height:1;letter-spacing:.04em}.logo i{width:.7em;height:.7em;border:3px solid var(--ec-red);border-radius:3px;box-shadow:inset 0 -3px 0 var(--ec-amber);transform:rotate(45deg);flex:none}.logo--dark{color:var(--ec-cream)}.logo--dark i{border-color:var(--ec-cream);box-shadow:inset 0 -3px 0 var(--ec-orange)}
</style>
