<script setup lang="ts">
import type { RecentForm } from '../types/football'
defineProps<{ form: RecentForm }>()
const letter: Record<string, string> = { W: 'V', D: 'E', L: 'D' }
const label: Record<string, string> = { W: 'Vitória', D: 'Empate', L: 'Derrota' }
</script>
<template>
  <section class="panel form-panel">
    <div class="section-title"><h2>Forma recente</h2><span class="muted">{{ form.played }} jogos · mais recente primeiro</span></div>
    <div class="form-grid">
      <div class="form-streak"><div class="badges"><span v-for="(result, i) in form.sequence" :key="i" :class="['result', result]" :aria-label="label[result]">{{ letter[result] }}</span><span v-if="!form.played" class="muted">Nenhuma partida finalizada</span></div><p>{{ form.wins }} vitórias <span>·</span> {{ form.draws }} empates <span>·</span> {{ form.losses }} derrotas</p></div>
      <div class="form-number"><strong>{{ form.goals_scored }}<small> : {{ form.goals_conceded }}</small></strong><span>Gols pró / contra</span></div>
      <div class="form-number"><strong>{{ form.average_goals.toFixed(2) }}</strong><span>Gols por jogo</span></div>
      <div class="form-number"><strong class="lime">{{ form.points_percentage }}<small>%</small></strong><span>Aproveitamento</span></div>
    </div>
  </section>
</template>
