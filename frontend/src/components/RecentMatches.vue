<script setup lang="ts">
import type { Card, Goal, Match } from '../types/football'
import TeamBadge from './TeamBadge.vue'
import LineupsDialog from './LineupsDialog.vue'
defineProps<{ matches: Match[]; teamId: string }>()
const sides = ['home', 'away'] as const
const letter: Record<string, string> = { W: 'V', D: 'E', L: 'D' }
const statLabels: Record<string, string> = { possession: 'Posse de bola', shots: 'Finalizações', shots_on_target: 'Finalizações no gol', corners: 'Escanteios', fouls: 'Faltas', yellow_cards: 'Cartões amarelos', red_cards: 'Cartões vermelhos' }
const goalsOf = (match: Match, side: Goal['side']) => (match.goals ?? []).filter((g) => g.side === side)
const cardsOf = (match: Match, side: Card['side']) => (match.cards ?? []).filter((c) => c.side === side)
const note = (g: Goal) => g.kind === 'own_goal' ? ' (contra)' : g.kind === 'penalty' ? ' (pên.)' : ''
function date(value: string) { return new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC' }).format(new Date(value)) }
function outcome(match: Match, id: string) { const difference = (match.home_score - match.away_score) * (match.home_team.id === id ? 1 : -1); return difference > 0 ? 'W' : difference < 0 ? 'L' : 'D' }
</script>
<template>
  <section class="matches-section">
    <div class="section-title"><h2>Últimas {{ matches.length }} partidas disponíveis</h2><span class="muted">Neste conjunto de dados</span></div>
    <div class="match-grid"><article v-for="match in matches" :key="match.id" class="match-card">
      <div class="match-meta"><span>{{ match.competition }}</span><span :class="['result', 'small', outcome(match, teamId)]">{{ letter[outcome(match, teamId)] }}</span></div>
      <p class="match-date">{{ date(match.date) }} <span>· Final</span></p>
      <p v-if="match.venue || match.referee" class="match-place"><span v-if="match.venue" title="Estádio"><svg class="place-icon" viewBox="0 0 16 16" aria-hidden="true"><ellipse cx="8" cy="6" rx="6.5" ry="2.6" fill="none" stroke="currentColor" stroke-width="1.2"/><path d="M1.5 6v4c0 1.4 2.9 2.6 6.5 2.6s6.5-1.2 6.5-2.6V6" fill="none" stroke="currentColor" stroke-width="1.2"/><ellipse cx="8" cy="6" rx="3" ry="1" fill="currentColor" opacity=".55"/></svg>{{ match.venue }}</span><span v-if="match.referee" title="Árbitro"><svg class="place-icon" viewBox="0 0 16 16" aria-hidden="true"><circle cx="6" cy="9.5" r="3.6" fill="none" stroke="currentColor" stroke-width="1.2"/><path d="M8.6 7 14 5.2v2.6L9.6 9.2" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/><circle cx="6" cy="9.5" r="1.1" fill="currentColor"/></svg>{{ match.referee }}</span></p>
      <div class="score-line"><div><TeamBadge :team="match.home_team" /><strong>{{ match.home_team.name }}</strong><small>MANDANTE</small></div><b>{{ match.home_score }}<span>:</span>{{ match.away_score }}</b><div><TeamBadge :team="match.away_team" /><strong>{{ match.away_team.name }}</strong><small>VISITANTE</small></div></div>
      <div v-if="match.goals?.length" class="goal-list" aria-label="Gols da partida">
        <ul v-for="side in sides" :key="side" :class="side">
          <li v-for="(goal, i) in goalsOf(match, side)" :key="i"><span class="goal-min">{{ goal.minute }}'</span><svg class="goal-icon" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="6.6" fill="none" stroke="currentColor" stroke-width="1.3"/><path d="M8 5.1 10.6 7 9.6 10H6.4L5.4 7Z" fill="currentColor"/><path d="M8 5.1V1.6M10.6 7l3.2-1.1M9.6 10l1.9 2.9M6.4 10l-1.9 2.9M5.4 7 2.2 5.9" stroke="currentColor" stroke-width="1.1" fill="none"/></svg>{{ goal.player }}<em>{{ note(goal) }}</em><small v-if="goal.assist">assist. {{ goal.assist }}</small></li>
        </ul>
      </div>
      <details v-if="match.statistics || match.cards?.length" class="match-details"><summary>Estatísticas da partida <span aria-hidden="true">+</span></summary>
        <template v-if="match.statistics"><div v-for="(pair, key) in match.statistics" :key="key" class="stat-row"><b>{{ pair.home }}{{ key === 'possession' ? '%' : '' }}</b><span>{{ statLabels[key] ?? key }}</span><b>{{ pair.away }}{{ key === 'possession' ? '%' : '' }}</b></div></template>
        <div v-if="match.cards?.length" class="card-list" aria-label="Cartões">
          <p class="card-title">Cartões</p>
          <div class="card-columns">
            <ul v-for="side in sides" :key="side" :class="side">
              <li v-for="(card, i) in cardsOf(match, side)" :key="i"><span :class="['card-icon', card.color]" :aria-label="card.color === 'red' ? 'Cartão vermelho' : 'Cartão amarelo'"></span><span class="goal-min">{{ card.minute }}'</span> {{ card.player }}</li>
            </ul>
          </div>
        </div>
      </details>
      <p v-else class="muted">Estatísticas indisponíveis</p>
      <LineupsDialog :match="match" />
    </article></div><p v-if="!matches.length" class="empty">Nenhuma partida recente disponível.</p>
  </section>
</template>
