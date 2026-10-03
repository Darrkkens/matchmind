<script setup lang="ts">
import type { RecentForm } from '../types/football'
import { outcomeLabel, outcomeLetter } from '../utils/format'
defineProps<{ form: RecentForm }>()
</script>

<template>
  <section aria-labelledby="form-title">
    <div class="section-head"><h2 id="form-title">Forma recente</h2><span class="muted">{{ form.played }} jogos · mais recente primeiro</span></div>
    <div class="strip num">
      <div class="streak">
        <ol v-if="form.played" class="sequence">
          <li v-for="(result, i) in form.sequence" :key="i" :class="['result', result]" role="img" :aria-label="outcomeLabel[result]">{{ outcomeLetter[result] }}</li>
        </ol>
        <p v-else class="muted">Nenhuma partida finalizada</p>
        <p class="record">{{ form.wins }} {{ form.wins === 1 ? 'vitória' : 'vitórias' }} · {{ form.draws }} {{ form.draws === 1 ? 'empate' : 'empates' }} · {{ form.losses }} {{ form.losses === 1 ? 'derrota' : 'derrotas' }}</p>
      </div>
      <div class="metric"><strong>{{ form.goals_scored }}<span class="dim">:{{ form.goals_conceded }}</span></strong><span>Gols pró / contra</span></div>
      <div class="metric"><strong>{{ form.average_goals.toFixed(2) }}</strong><span>Gols por jogo</span></div>
      <div class="metric"><strong class="accent">{{ form.points_percentage }}<span class="unit">%</span></strong><span>Aproveitamento</span></div>
    </div>
  </section>
</template>

<style scoped>
.strip { display: grid; grid-template-columns: 1.3fr repeat(3, 1fr); border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
.strip > div { padding: var(--s-4) var(--s-5); display: flex; flex-direction: column; justify-content: center; gap: var(--s-2); }
.strip > div:first-child { padding-left: 0; }
.strip > div + div { border-left: 1px solid var(--line); }
.sequence { display: flex; gap: 6px; }
.record { font-size: var(--fs-xs); color: var(--text-3); }
.metric strong { font-size: var(--fs-xl); font-weight: 550; line-height: 1.1; letter-spacing: -.02em; }
.metric > span { font-size: var(--fs-xs); color: var(--text-3); }
.dim { color: var(--text-3); }
.unit { font-size: .7em; }
.accent { color: var(--accent); }
@media (max-width: 700px) {
  .strip { grid-template-columns: 1fr 1fr; }
  .strip > div { padding: var(--s-4) 0; }
  .strip > div:nth-child(even) { padding-left: var(--s-4); }
  .strip > div:nth-child(3) { border-left: 0; }
  .strip > div:nth-child(n + 3) { border-top: 1px solid var(--line); }
  .metric strong { font-size: var(--fs-lg); }
}
</style>
