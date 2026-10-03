<script setup lang="ts">
import type { Team } from '../types/football'
import TeamBadge from './TeamBadge.vue'
defineProps<{ team: Team; position?: number }>()
</script>
<template>
  <section class="team-header">
    <TeamBadge :team="team" size="large" />
    <div class="team-title"><div class="eyebrow">RAIO-X DO CLUBE <span class="badge real">OpenFootball · Dados reais</span><span v-if="position" class="badge real">{{ position }}º na tabela</span></div><h1>{{ team.name }}</h1><p>{{ team.country === 'Brazil' ? 'Brasil' : team.country }} <template v-if="team.stadium"><span class="separator">/</span> {{ team.stadium }}</template></p></div>
    <div class="team-header-note"><span class="live-dot"></span> SEU CLUBE EM FOCO<p>Entenda o jogo. Enxergue a história.</p></div>
  </section>
</template>
