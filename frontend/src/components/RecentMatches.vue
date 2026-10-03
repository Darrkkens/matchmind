<script setup lang="ts">
import type { Card, Goal, Match, MatchStatistics } from '../types/football'
import { formatLongDay, goalNote, outcome, outcomeLabel, outcomeLetter, roundLabel } from '../utils/format'
import TeamBadge from './TeamBadge.vue'
import LineupsDialog from './LineupsDialog.vue'

defineProps<{ matches: Match[]; teamId: string }>()
const sides = ['home', 'away'] as const
const statLabels: Record<keyof MatchStatistics, string> = { possession: 'Posse de bola', shots: 'Finalizações', shots_on_target: 'Finalizações no gol', corners: 'Escanteios', fouls: 'Faltas', yellow_cards: 'Cartões amarelos', red_cards: 'Cartões vermelhos' }
const goalsOf = (match: Match, side: Goal['side']) => (match.goals ?? []).filter((g) => g.side === side)
const cardsOf = (match: Match, side: Card['side']) => (match.cards ?? []).filter((c) => c.side === side)
const stats = (s: MatchStatistics) => (Object.keys(statLabels) as (keyof MatchStatistics)[]).filter((k) => s[k]).map((k) => ({ key: k, label: statLabels[k], ...s[k], unit: k === 'possession' ? '%' : '' }))
const expandable = (m: Match) => !!(m.goals?.length || m.statistics || m.cards?.length || m.venue || m.referee || m.lineup_ref)
</script>

<template>
  <section aria-labelledby="matches-title">
    <div class="section-head"><h2 id="matches-title">Últimos {{ matches.length }} resultados</h2><span class="muted">Mais recente primeiro</span></div>
    <ul v-if="matches.length" class="list">
      <li v-for="match in matches" :key="match.id">
        <component :is="expandable(match) ? 'details' : 'div'" class="match">
          <component :is="expandable(match) ? 'summary' : 'div'" class="row">
            <span :class="['result', outcome(match, teamId)]" role="img" :aria-label="outcomeLabel[outcome(match, teamId)]">{{ outcomeLetter[outcome(match, teamId)] }}</span>
            <span class="when"><time :datetime="match.date">{{ formatLongDay(match.date) }}</time><small>{{ roundLabel(match.round) }}</small></span>
            <span class="teams num">
              <span :class="['side home', { own: match.home_team.id === teamId }]"><span class="name">{{ match.home_team.name }}</span><TeamBadge :team="match.home_team" size="xs" /></span>
              <b class="score"><span class="sr-only">{{ match.home_team.name }} </span>{{ match.home_score }}<span aria-hidden="true">–</span><span class="sr-only"> a </span>{{ match.away_score }}<span class="sr-only"> {{ match.away_team.name }}</span></b>
              <span :class="['side away', { own: match.away_team.id === teamId }]"><TeamBadge :team="match.away_team" size="xs" /><span class="name">{{ match.away_team.name }}</span></span>
            </span>
          </component>

          <div v-if="expandable(match)" class="details">
            <div v-if="match.goals?.length" class="goals" aria-label="Gols da partida">
              <ul v-for="side in sides" :key="side" :class="side">
                <li v-for="(goal, i) in goalsOf(match, side)" :key="i"><span class="minute num">{{ goal.minute }}'</span><span>{{ goal.player }}<em>{{ goalNote(goal) }}</em><small v-if="goal.assist">assist. {{ goal.assist }}</small></span></li>
              </ul>
            </div>
            <p v-if="match.venue || match.referee" class="place">
              <span v-if="match.venue"><span class="label">Estádio</span> {{ match.venue }}</span>
              <span v-if="match.referee"><span class="label">Árbitro</span> {{ match.referee }}</span>
            </p>
            <dl v-if="match.statistics" class="stats num">
              <div v-for="s in stats(match.statistics)" :key="s.key"><dd>{{ s.home }}{{ s.unit }}</dd><dt>{{ s.label }}</dt><dd>{{ s.away }}{{ s.unit }}</dd></div>
            </dl>
            <p v-else class="muted">Estatísticas indisponíveis</p>
            <div v-if="match.cards?.length" aria-label="Cartões">
              <p class="eyebrow cards-title">Cartões</p>
              <div class="cards">
                <ul v-for="side in sides" :key="side" :class="side">
                  <li v-for="(card, i) in cardsOf(match, side)" :key="i"><span :class="['card-icon', card.color]" role="img" :aria-label="card.color === 'red' ? 'Cartão vermelho' : 'Cartão amarelo'"></span><span class="minute num">{{ card.minute }}'</span> {{ card.player }}</li>
                </ul>
              </div>
            </div>
            <LineupsDialog :match="match" />
          </div>
        </component>
      </li>
    </ul>
    <p v-else class="empty">Nenhuma partida recente disponível.</p>
  </section>
</template>

<style scoped>
.list { border-top: 1px solid var(--line); }
.list > li { border-bottom: 1px solid var(--line); }
.match { interpolate-size: allow-keywords; }

/* One row per result: outcome · date · home score away · chevron. */
.row {
  display: grid; grid-template-columns: auto 120px minmax(0, 1fr) 20px; grid-template-areas: "res when teams chev";
  align-items: center; gap: var(--s-4); min-height: 56px; padding: var(--s-2);
  list-style: none; border-radius: var(--r-sm); transition: background-color var(--dur-fast) ease;
}
summary.row { cursor: pointer; }
summary.row::-webkit-details-marker { display: none; }
summary.row::after {
  content: ''; grid-area: chev; justify-self: center; width: 7px; height: 7px; color: var(--text-3);
  border-right: 1.5px solid currentColor; border-bottom: 1.5px solid currentColor;
  transform: translateY(-2px) rotate(45deg); transition: transform var(--dur) var(--ease-out);
}
details[open] > summary.row::after { transform: translateY(2px) rotate(-135deg); }
@media (hover: hover) and (pointer: fine) { summary.row:hover { background: var(--surface-1); } }

.result { grid-area: res; }
.when { grid-area: when; display: grid; font-size: var(--fs-xs); color: var(--text-2); }
.when small { font-size: var(--fs-2xs); color: var(--text-3); }
.teams { grid-area: teams; display: grid; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr); align-items: center; gap: var(--s-3); }
.side { display: flex; align-items: center; gap: var(--s-2); min-width: 0; font-size: var(--fs-sm); color: var(--text-2); }
.side.home { justify-content: flex-end; text-align: right; }
.side.own { color: var(--text); font-weight: 600; }
.name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.score { min-width: 64px; text-align: center; font-size: var(--fs-md); font-weight: 650; padding: 2px var(--s-2); border-radius: var(--r-sm); background: var(--surface-2); }
.score > span[aria-hidden] { color: var(--text-3); margin: 0 4px; font-weight: 400; }

.match::details-content { height: 0; overflow: clip; transition: height var(--dur) var(--ease-out), content-visibility var(--dur) allow-discrete; }
.match[open]::details-content { height: auto; }
/* Details sit under the scoreline (not under the result square) and share its center line. */
.details { display: grid; justify-items: center; gap: var(--s-3); padding: var(--s-2) calc(var(--s-2) + 20px + var(--s-4)) var(--s-4) calc(var(--s-2) + 28px + 120px + var(--s-4) * 2); }
.details > :not(.lineups-button) { width: min(100%, 560px); }
.goals { display: grid; grid-template-columns: 1fr 1fr; gap: var(--s-3); font-size: var(--fs-xs); color: var(--text-2); }
.goals ul { display: grid; gap: 6px; align-content: start; }
.goals li { display: flex; gap: 6px; line-height: 1.4; overflow-wrap: anywhere; }
.goals .home li { flex-direction: row-reverse; text-align: right; }
.goals em { font-style: normal; color: var(--text-3); }
.goals small { display: block; color: var(--text-3); font-size: var(--fs-2xs); }
.minute { color: var(--accent); white-space: nowrap; }
.place { display: flex; flex-wrap: wrap; justify-content: center; gap: 2px var(--s-4); text-align: center; font-size: var(--fs-xs); color: var(--text-2); }
.place .label { color: var(--text-3); }
.stats { max-width: 440px; }
.details > .stats { width: min(100%, 440px); }
.details > .muted { text-align: center; }
.stats div { display: grid; grid-template-columns: 48px 1fr 48px; align-items: center; padding: 5px 0; border-top: 1px solid var(--line); font-size: var(--fs-xs); }
.stats div:first-child { border-top: 0; }
.stats dt { text-align: center; color: var(--text-3); }
.stats dd { font-weight: 600; }
.stats dd:last-child { text-align: right; }
.cards-title { margin-bottom: var(--s-2); text-align: center; }
.cards { display: grid; grid-template-columns: 1fr 1fr; gap: var(--s-3); }
.cards ul { display: grid; gap: 6px; align-content: start; font-size: var(--fs-xs); color: var(--text-2); }
.cards li { display: flex; align-items: center; gap: 6px; overflow-wrap: anywhere; }
.cards .home li { flex-direction: row-reverse; text-align: right; }
.details :deep(.lineups-button) { justify-self: center; }

@container (max-width: 640px) {
  .row { grid-template-columns: auto minmax(0, 1fr) 20px; grid-template-areas: "res when chev" "teams teams teams"; row-gap: var(--s-2); padding: var(--s-3) 0; }
  .when { display: flex; gap: var(--s-2); align-items: baseline; }
  .side { font-size: var(--fs-xs); }
  .score { min-width: 52px; font-size: var(--fs-base); }
  .details { padding: 0 0 var(--s-4); }
}
@media (prefers-reduced-motion: reduce) { .match::details-content { transition: none; } }
</style>
