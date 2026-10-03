<script setup lang="ts">
import { computed } from 'vue'
import type { Availability, Fixture, Match, MatchAvailability, SeasonStats } from '../types/football'
import { formatWeekday, outcome, outcomeLabel, outcomeLetter, relativeDay, roundLabel } from '../utils/format'
import MatchSimulation from './MatchSimulation.vue'
import { formatMetric, metric, metricInfo, recordLine } from '../utils/season'
import TeamBadge from './TeamBadge.vue'

const props = defineProps<{ fixture?: Fixture; teamId: string; season?: SeasonStats; opponentSeason?: SeasonStats; recent?: Match[]; opponentRecent?: Match[]; availability?: MatchAvailability }>()
const suspendedLine = (a?: Availability) => a ? (a.suspended.length ? a.suspended.map((s) => s.player).join(', ') : 'Nenhum') : '—'
// Most used first (sorted by the API); the rest are summarized to keep the row short.
const AT_RISK_SHOWN = 4
const atRiskLine = (a?: Availability) => {
  if (!a) return '—'
  if (!a.at_risk.length) return 'Nenhum'
  const rest = a.at_risk.length - AT_RISK_SHOWN
  return a.at_risk.slice(0, AT_RISK_SHOWN).join(', ') + (rest > 0 ? ` +${rest}` : '')
}
defineEmits<{ explain: [question: string] }>()
const home = computed(() => props.fixture?.home_team.id === props.teamId)
const club = computed(() => props.fixture && (home.value ? props.fixture.home_team : props.fixture.away_team))
const opponent = computed(() => props.fixture && (home.value ? props.fixture.away_team : props.fixture.home_team))

// "How they arrive": each club's record in this match's venue role, then season averages.
const PREVIEW_KEYS = ['goals', 'goals_against', 'shots_on_target', 'save_pct']
const last5 = (matches: Match[] | undefined, id: string | undefined) => id ? (matches ?? []).slice(0, 5).map((m) => outcome(m, id)) : []
const preview = computed(() => {
  const ours = props.season, theirs = props.opponentSeason
  const ourLast = last5(props.recent, props.teamId), theirLast = last5(props.opponentRecent, opponent.value?.id)
  if (!ours || !theirs) return ourLast.length || theirLast.length ? { ourSplit: undefined, theirSplit: undefined, rows: [], ourLast, theirLast } : null
  const ourSplit = home.value ? ours.home : ours.away
  const theirSplit = home.value ? theirs.away : theirs.home
  const rows = PREVIEW_KEYS.map((key) => {
    const a = metric(ours, key), b = metric(theirs, key)
    if (!a || !b) return null
    const better = a.value === b.value ? '' : (a.value > b.value) !== !!a.lower_is_better ? 'ours' : 'theirs'
    return { key, label: metricInfo[key].label, ours: formatMetric(key, a.value), theirs: formatMetric(key, b.value), better }
  }).filter((r) => r !== null)
  return { ourSplit, theirSplit, rows, ourLast, theirLast }
})
</script>

<template>
  <section aria-labelledby="next-title">
    <div class="section-head"><h2 id="next-title">Próximo jogo</h2><span v-if="fixture" class="muted">{{ fixture.competition }}</span></div>
    <div v-if="fixture && opponent" class="next">
      <div class="when">
        <p class="relative">{{ relativeDay(fixture.date) }}</p>
        <p class="date num"><time :datetime="fixture.date">{{ formatWeekday(fixture.date) }}</time><template v-if="fixture.time"> · {{ fixture.time }}</template></p>
        <p class="meta">{{ [roundLabel(fixture.round), fixture.time ? 'horário local' : 'horário a definir'].filter(Boolean).join(' · ') }}</p>
      </div>
      <div class="versus">
        <TeamBadge :team="opponent" />
        <div>
          <p class="label">{{ home ? 'Em casa contra' : 'Fora de casa contra' }}</p>
          <p class="opponent">{{ opponent.name }}</p>
        </div>
      </div>
      <p class="fixture-line">{{ fixture.home_team.name }} <span>x</span> {{ fixture.away_team.name }}</p>

      <div v-if="preview && club" class="preview">
        <table class="num">
          <caption class="eyebrow">Como chegam</caption>
          <thead><tr><th scope="col"><div class="th-team"><TeamBadge :team="club" size="xs" /><span>{{ club.name }}</span></div></th><th scope="col"><span class="sr-only">Métrica</span></th><th scope="col"><div class="th-team end"><TeamBadge :team="opponent!" size="xs" /><span>{{ opponent!.name }}</span></div></th></tr></thead>
          <tbody>
            <tr v-if="preview.ourLast.length || preview.theirLast.length">
              <td><span class="seq"><span v-for="(r, i) in preview.ourLast" :key="i" :class="['result sm', r]" role="img" :aria-label="outcomeLabel[r]">{{ outcomeLetter[r] }}</span></span></td>
              <th scope="row">Últimos 5 jogos</th>
              <td><span class="seq end"><span v-for="(r, i) in preview.theirLast" :key="i" :class="['result sm', r]" role="img" :aria-label="outcomeLabel[r]">{{ outcomeLetter[r] }}</span></span></td>
            </tr>
            <tr v-if="preview.ourSplit && preview.theirSplit">
              <td :class="{ better: preview.ourSplit.points_per_match > preview.theirSplit.points_per_match }">{{ preview.ourSplit.points_per_match.toLocaleString('pt-BR') }} <small>{{ recordLine(preview.ourSplit) }}</small></td>
              <th scope="row">Pts/jogo {{ home ? 'em casa × fora' : 'fora × em casa' }}</th>
              <td :class="{ better: preview.theirSplit.points_per_match > preview.ourSplit.points_per_match }">{{ preview.theirSplit.points_per_match.toLocaleString('pt-BR') }} <small>{{ recordLine(preview.theirSplit) }}</small></td>
            </tr>
            <template v-if="availability">
              <tr class="text-row">
                <td :class="{ warn: availability.club?.suspended.length }">{{ suspendedLine(availability.club) }}</td>
                <th scope="row">Suspensos</th>
                <td :class="{ warn: availability.opponent?.suspended.length }">{{ suspendedLine(availability.opponent) }}</td>
              </tr>
              <tr class="text-row">
                <td :title="availability.club?.at_risk.join(', ')">{{ atRiskLine(availability.club) }}</td>
                <th scope="row"><abbr title="Com mais um amarelo ficam suspensos">Pendurados</abbr></th>
                <td :title="availability.opponent?.at_risk.join(', ')">{{ atRiskLine(availability.opponent) }}</td>
              </tr>
            </template>
            <tr v-for="r in preview.rows" :key="r.key">
              <td :class="{ better: r.better === 'ours' }">{{ r.ours }}</td>
              <th scope="row">{{ r.label }}</th>
              <td :class="{ better: r.better === 'theirs' }">{{ r.theirs }}</td>
            </tr>
          </tbody>
        </table>
        <p v-if="availability" class="availability-note">Suspensões estimadas pelos cartões do Brasileirão (vermelho ou a cada 3 amarelos); decisões do STJD e lesões não entram.<template v-if="availability.club?.note || availability.opponent?.note"> {{ [availability.club?.note, availability.opponent?.note].filter(Boolean).join(' ') }}</template></p>
      </div>
      <MatchSimulation :fixture="fixture" :team-id="teamId" @explain="$emit('explain', $event)" />
    </div>
    <p v-else class="empty">Nenhum jogo agendado para este clube neste conjunto de dados.</p>
  </section>
</template>

<style scoped>
.next { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: var(--s-3) var(--s-6); padding: var(--s-4) var(--s-5); background: var(--surface-1); border: 1px solid var(--line); border-radius: var(--r-lg); }
.when { display: grid; gap: 2px; padding-right: var(--s-6); border-right: 1px solid var(--line); }
.relative { font-size: var(--fs-2xs); font-weight: 600; letter-spacing: var(--tracking-label); text-transform: uppercase; color: var(--accent); }
.date { font-size: var(--fs-lg); font-weight: 600; letter-spacing: -.01em; }
.meta { font-size: var(--fs-xs); color: var(--text-3); }
.versus { display: flex; align-items: center; gap: var(--s-3); min-width: 0; }
.versus :deep(.team-badge) { --w: 40px; }
.label { font-size: var(--fs-xs); color: var(--text-3); }
.opponent { font-size: var(--fs-md); font-weight: 600; overflow-wrap: anywhere; }
.fixture-line { font-size: var(--fs-xs); color: var(--text-3); text-align: right; }
.fixture-line span { margin: 0 4px; }
.preview { grid-column: 1 / -1; padding-top: var(--s-4); border-top: 1px solid var(--line); }
table { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); table-layout: fixed; }
caption { text-align: left; margin-bottom: var(--s-2); }
thead th { padding-bottom: var(--s-2); font-size: var(--fs-xs); font-weight: 600; }
thead th:first-child, thead th:last-child { width: 32%; }
.th-team { display: flex; align-items: center; gap: 6px; min-width: 0; }
.th-team.end { flex-direction: row-reverse; }
.th-team span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
tbody th { font-size: var(--fs-xs); font-weight: 400; color: var(--text-3); text-align: center; padding: 6px var(--s-2); }
tbody td { padding: 6px 0; font-weight: 600; color: var(--text-2); }
tbody td:last-child { text-align: right; }
tbody tr + tr > * { border-top: 1px solid var(--line); }
td.better { color: var(--accent); }
.seq { display: inline-flex; gap: 3px; }
.seq.end { justify-content: flex-end; }
tr.text-row td { font-weight: 500; font-size: var(--fs-xs); line-height: 1.5; overflow-wrap: anywhere; white-space: normal; }
tr.text-row td.warn { color: var(--warn); }
abbr { text-decoration: none; }
.availability-note { font-size: var(--fs-2xs); color: var(--text-3); margin-top: var(--s-2); line-height: 1.5; }
td small { display: block; font-size: var(--fs-2xs); font-weight: 400; color: var(--text-3); }
@container (max-width: 480px) {
  .seq { gap: 2px; }
  .seq :deep(.result.sm), .seq .result.sm { width: 17px; height: 17px; font-size: 9px; border-radius: 4px; }
  thead th:first-child, thead th:last-child { width: 34%; }
}
@container (max-width: 640px) {
  .next { grid-template-columns: 1fr; padding: var(--s-4); }
  .when { padding: 0 0 var(--s-3); border-right: 0; border-bottom: 1px solid var(--line); }
  .fixture-line { text-align: left; }
}
</style>
