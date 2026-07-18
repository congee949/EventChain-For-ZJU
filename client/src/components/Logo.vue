<script setup>
defineProps({ theme:{type:String,default:'light'}, size:{type:[Number,String],default:22} });
</script>
<template><span class="logo" :class="`logo--${theme}`" :style="{fontSize:`${size}px`}"><i></i><b>EVENTCHAIN</b></span></template>
<style scoped>
.logo{display:inline-flex;align-items:center;gap:.55em;color:var(--ec-ink);font-family:var(--ec-font-display);font-weight:900;line-height:1;letter-spacing:.04em}.logo i{width:.58em;height:.58em;border-radius:2px;background:var(--ec-red);transform:rotate(45deg);flex:none}.logo--dark{color:var(--ec-cream)}.logo--dark i{background:var(--ec-orange)}
</style>
