<script setup lang="ts">
import { computed } from 'vue'
import type { Team } from '../types/football'
import TeamBadge from './TeamBadge.vue'

const props = defineProps<{ team: Team; position?: number; competition?: string }>()
const facts = computed(() => [
  { label: 'Técnico', value: props.team.coach },
  { label: 'Fundação', value: props.team.founded_year || '' },
  { label: 'Estádio', value: props.team.stadium },
])
const country = computed(() => props.team.country === 'Brazil' ? 'Brasil' : props.team.country)
</script>

<template>
  <header class="team-header">
    <TeamBadge :team="team" size="large" />
    <div class="title">
      <p class="eyebrow">Raio-x do clube<template v-if="country"> · {{ country }}</template></p>
      <h1>{{ team.name }}</h1>
      <p v-if="position" class="standing"><strong>{{ position }}º</strong> na tabela<template v-if="competition"> · {{ competition }}</template></p>
    </div>
    <dl class="facts">
      <div v-for="fact in facts" :key="fact.label"><dt>{{ fact.label }}</dt><dd :class="{ missing: !fact.value }">{{ fact.value || 'Indisponível' }}</dd></div>
    </dl>
  </header>
</template>

<style scoped>
.team-header { display: grid; grid-template-columns: auto 1fr; align-items: center; gap: var(--s-2) var(--s-5); }
.title { min-width: 0; display: grid; gap: var(--s-1); }
h1 { font-size: var(--fs-2xl); line-height: 1.1; font-weight: 650; letter-spacing: -.035em; overflow-wrap: anywhere; }
.standing { font-size: var(--fs-sm); color: var(--text-2); }
.standing strong { color: var(--accent); font-weight: 650; }
.facts { grid-column: 1 / -1; display: flex; flex-wrap: wrap; gap: var(--s-2) var(--s-6); margin-top: var(--s-4); padding-top: var(--s-4); border-top: 1px solid var(--line); }
.facts div { display: grid; gap: 2px; min-width: 0; }
dt { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; letter-spacing: var(--tracking-label); font-weight: 600; }
dd { font-size: var(--fs-sm); font-weight: 500; }
dd.missing { color: var(--text-3); font-weight: 400; }
@media (max-width: 600px) {
  .team-header { column-gap: var(--s-4); }
  h1 { font-size: var(--fs-xl); }
  .facts { column-gap: var(--s-5); }
}
</style>
