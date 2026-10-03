<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Team } from '../types/football'

const props = withDefaults(defineProps<{ team: Team; size?: 'large' | 'small' }>(), { size: 'small' })
const failed = ref(false)
watch(() => props.team.logo_url, () => { failed.value = false })
</script>

<template>
  <span :class="['team-badge', size, { 'has-image': team.logo_url && !failed }]">
    <img v-if="team.logo_url && !failed" :src="team.logo_url" :alt="`Escudo do ${team.name}`" decoding="async" @error="failed = true" />
    <span v-else :aria-label="`Iniciais do ${team.name}`">{{ team.short_name.slice(0, 3) }}</span>
  </span>
</template>

<style scoped>
.team-badge{width:34px;height:38px;display:inline-grid;place-items:center;flex-shrink:0;border:1px solid #607750;border-radius:8px 8px 14px 14px;background:#293721;color:var(--lime);font-size:10px;font-weight:700}
.team-badge.large{width:83px;height:93px;font-size:26px;border-radius:14px}
.team-badge.has-image{background:transparent;border-color:transparent;border-radius:0}
.team-badge img{display:block;width:100%;height:100%;object-fit:contain;filter:drop-shadow(0 2px 3px #0004)}
@media(max-width:780px){.team-badge.large{width:64px;height:74px;font-size:20px}}
</style>
