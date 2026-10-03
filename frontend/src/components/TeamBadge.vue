<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Team } from '../types/football'

type BadgeTeam = Pick<Team, 'name' | 'short_name' | 'logo_url'>
const props = withDefaults(defineProps<{ team: BadgeTeam; size?: 'xs' | 'small' | 'large' }>(), { size: 'small' })
const failed = ref(false)
watch(() => props.team.logo_url, () => { failed.value = false })
</script>

<template>
  <span :class="['team-badge', size, { 'has-image': team.logo_url && !failed }]">
    <img v-if="team.logo_url && !failed" :src="team.logo_url" :alt="`Escudo do ${team.name}`" decoding="async" loading="lazy" @error="failed = true" />
    <span v-else role="img" :aria-label="`Escudo do ${team.name}`">{{ team.short_name.slice(0, 3) }}</span>
  </span>
</template>

<style scoped>
.team-badge { --w: 32px; width: var(--w); height: calc(var(--w) * 1.12); display: inline-grid; place-items: center; flex-shrink: 0; border: 1px solid var(--line-strong); border-radius: 7px 7px 45% 45% / 7px 7px 35% 35%; background: var(--surface-3); color: var(--accent); font-size: 10px; font-weight: 700; letter-spacing: .02em; }
.team-badge.xs { --w: 22px; font-size: 8px; }
.team-badge.large { --w: 72px; font-size: var(--fs-lg); }
.team-badge.has-image { background: transparent; border-color: transparent; }
.team-badge img { width: 100%; height: 100%; object-fit: contain; filter: drop-shadow(0 2px 3px #0005); }
@media (max-width: 600px) { .team-badge.large { --w: 56px; font-size: var(--fs-md); } }
</style>
