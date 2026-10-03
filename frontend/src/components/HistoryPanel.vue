<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ClubHistory } from '../types/football'
import { formatDay, pointsPercentage, scorerList } from '../utils/format'

const props = defineProps<{ history: ClubHistory; teamName: string }>()
const period = computed(() => `${props.history.first_season}–${props.history.last_season}`)
const allTime = computed(() => props.history.all_time)
const summary = computed(() => [
  { value: props.history.titles.length, label: props.history.titles.length === 1 ? 'título' : 'títulos' },
  { value: props.history.seasons_played, label: 'temporadas' },
  { value: allTime.value.played, label: 'jogos' },
])
const opponents = computed(() => props.history.head_to_head.filter((r) => r.meetings?.length))
const opponentId = ref(opponents.value[0]?.opponent_id ?? '')
const selected = computed(() => opponents.value.find((r) => r.opponent_id === opponentId.value) ?? opponents.value[0])
const withAverages = computed(() => (props.history.seasons ?? []).some((s) => s.averages))
const coaches = (list?: { name: string; matches: number }[]) => list?.length ? list.map((c) => c.name).join(', ') : '–'
</script>

<template>
  <section class="history" aria-labelledby="history-title">
    <div class="section-head"><h2 id="history-title">Histórico na Série A</h2><span class="muted num">{{ period }}</span></div>
    <p v-if="!history.dataset_name" class="empty">O {{ teamName }} não disputou a Série A entre {{ period }} neste conjunto de dados.</p>
    <template v-else>
      <dl class="summary num">
        <div v-for="item in summary" :key="item.label"><dt>{{ item.label }}</dt><dd>{{ item.value }}</dd></div>
        <div><dt>aproveitamento</dt><dd class="accent">{{ pointsPercentage(allTime) }}<span class="unit">%</span></dd></div>
      </dl>
      <p class="record num">{{ allTime.wins }} vitórias · {{ allTime.draws }} empates · {{ allTime.losses }} derrotas · {{ allTime.goals_for }} gols pró · {{ allTime.goals_against }} contra</p>

      <details v-if="history.seasons?.length" class="disclosure">
        <summary>Temporada a temporada</summary>
        <div class="block">
          <div class="scroll" tabindex="0" aria-label="Temporadas, role para ver mais colunas">
            <table class="num">
              <thead><tr><th scope="col">Ano</th><th scope="col"><abbr title="Posição final">Pos.</abbr></th><th scope="col"><abbr title="Pontos">Pts</abbr></th><th scope="col"><abbr title="Vitórias, empates e derrotas">V-E-D</abbr></th><th scope="col"><abbr title="Gols pró e contra">Gols</abbr></th><th scope="col" class="left">Técnicos</th><th v-if="withAverages" scope="col" class="left"><abbr title="Médias por jogo: posse · finalizações (no gol) · escanteios">Médias</abbr></th></tr></thead>
              <tbody>
                <tr v-for="s in history.seasons" :key="s.season" :class="{ champion: s.position === 1 }">
                  <th scope="row">{{ s.season }}</th>
                  <td>{{ s.position ? `${s.position}º` : '–' }}</td>
                  <td>{{ s.points }}</td>
                  <td>{{ s.wins }}-{{ s.draws }}-{{ s.losses }}</td>
                  <td>{{ s.goals_for }}:{{ s.goals_against }}</td>
                  <td class="left wrap">{{ coaches(s.coaches) }}</td>
                  <td v-if="withAverages" class="left dim">{{ s.averages ? `${s.averages.possession}% · ${s.averages.shots} (${s.averages.shots_on_target}) · ${s.averages.corners}` : '–' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="note">Posição calculada pela tabela final · técnicos em ordem de jogos{{ withAverages ? ' · médias por jogo: posse · finalizações (no gol) · escanteios, disponíveis de 2017 a 2023' : '' }}.</p>
        </div>
      </details>

      <details v-if="history.top_scorers?.length" class="disclosure">
        <summary>Artilheiros na Série A</summary>
        <div class="block">
          <ol class="ranking num">
            <li v-for="(p, i) in history.top_scorers" :key="p.player"><span class="rank">{{ i + 1 }}</span><span class="who">{{ p.player }}<small>{{ p.seasons }}</small></span><b>{{ p.goals }}<span class="sr-only"> gols</span></b><small class="extra">{{ p.penalties ? `${p.penalties} pên.` : '' }}</small></li>
          </ol>
          <p class="note">Gols pelo clube na Série A desde 2014 (sem gols contra). Nomes como registrados na fonte.</p>
        </div>
      </details>

      <details v-if="history.discipline" class="disclosure">
        <summary>Disciplina</summary>
        <div class="block">
          <p class="discipline num"><i class="card-icon yellow" aria-hidden="true"></i>{{ history.discipline.yellow }} amarelos <i class="card-icon red" aria-hidden="true"></i>{{ history.discipline.red }} vermelhos <small>({{ history.discipline.first_season }}–{{ history.discipline.last_season }})</small></p>
          <ol class="ranking num">
            <li v-for="(p, i) in history.discipline.most_booked" :key="p.player"><span class="rank">{{ i + 1 }}</span><span class="who">{{ p.player }}</span><b>{{ p.yellow }}<span class="sr-only"> amarelos</span></b><small class="extra">{{ p.red ? `${p.red} verm.` : '' }}</small></li>
          </ol>
          <p class="note">Jogadores com mais cartões pelo clube na Série A.</p>
        </div>
      </details>

      <details v-if="history.head_to_head.length" class="disclosure">
        <summary>Confrontos com os adversários atuais</summary>
        <div class="block">
          <div class="scroll" tabindex="0" aria-label="Confrontos, role para ver mais colunas">
            <table class="num">
              <thead><tr><th scope="col" class="left">Adversário</th><th scope="col"><abbr title="Jogos">J</abbr></th><th scope="col"><abbr title="Vitórias">V</abbr></th><th scope="col"><abbr title="Empates">E</abbr></th><th scope="col"><abbr title="Derrotas">D</abbr></th><th scope="col"><abbr title="Gols pró e contra">Gols</abbr></th></tr></thead>
              <tbody><tr v-for="row in history.head_to_head" :key="row.opponent_id"><th scope="row" class="left">{{ row.opponent_name }}</th><td>{{ row.played }}</td><td>{{ row.wins }}</td><td>{{ row.draws }}</td><td>{{ row.losses }}</td><td>{{ row.goals_for }}:{{ row.goals_against }}</td></tr></tbody>
            </table>
          </div>
          <div v-if="opponents.length" class="meetings">
            <label :for="`meetings-${teamName}`">Últimos jogos contra</label>
            <select :id="`meetings-${teamName}`" v-model="opponentId">
              <option v-for="row in opponents" :key="row.opponent_id" :value="row.opponent_id">{{ row.opponent_name }}</option>
            </select>
            <ul v-if="selected">
              <li v-for="m in selected.meetings" :key="m.date">
                <p class="meeting"><time :datetime="m.date">{{ formatDay(m.date) }}</time><b class="num">{{ m.home_team }} {{ m.home_score }}–{{ m.away_score }} {{ m.away_team }}</b><small v-if="m.stadium">{{ m.stadium }}</small></p>
                <p v-if="m.goals?.length" class="meeting-goals"><span v-if="scorerList(m.goals, 'home')">{{ m.home_team }}: {{ scorerList(m.goals, 'home') }}</span><span v-if="scorerList(m.goals, 'away')">{{ m.away_team }}: {{ scorerList(m.goals, 'away') }}</span></p>
              </li>
            </ul>
          </div>
        </div>
      </details>
    </template>
    <p v-if="history.unavailable?.length" class="note warn">Parte do histórico está indisponível agora: {{ history.unavailable.join(', ') }}.</p>
    <p class="note source">Fonte: <a class="link" :href="history.source_url" target="_blank" rel="noopener noreferrer">{{ history.source }}</a> (GPL-2.0). Somente Série A; títulos e posições calculados pela tabela final de cada temporada.</p>
  </section>
</template>

<style scoped>
.history { min-width: 0; }
.summary { display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--s-3); }
.summary div { display: flex; flex-direction: column-reverse; justify-content: flex-end; gap: 2px; }
.summary dd { font-size: var(--fs-xl); font-weight: 550; letter-spacing: -.02em; line-height: 1.15; }
.summary dt { font-size: var(--fs-xs); color: var(--text-3); }
.accent { color: var(--accent); }
.unit { font-size: .65em; }
.record { font-size: var(--fs-xs); color: var(--text-2); margin: var(--s-3) 0 var(--s-4); }
.block { padding-bottom: var(--s-4); }
.scroll { overflow-x: auto; border-radius: var(--r-sm); }
table { width: 100%; border-collapse: collapse; font-size: var(--fs-xs); }
th, td { padding: 6px var(--s-2); text-align: right; white-space: nowrap; }
thead th { font-size: var(--fs-2xs); letter-spacing: var(--tracking-label); color: var(--text-3); font-weight: 600; padding-top: 0; }
tbody th, tbody td { border-top: 1px solid var(--line); color: var(--text-2); }
tbody th { font-weight: 500; color: var(--text); }
.left { text-align: left; }
.wrap { white-space: normal; min-width: 140px; }
.dim { color: var(--text-3); }
abbr { text-decoration: none; }
tr.champion th, tr.champion td { color: var(--accent); }
.note { font-size: var(--fs-2xs); color: var(--text-3); margin-top: var(--s-2); line-height: 1.6; }
.warn { color: var(--warn); }
.source { margin-top: var(--s-4); }
.ranking li { display: grid; grid-template-columns: 20px 1fr auto 56px; gap: var(--s-2); align-items: center; padding: 6px 0; border-top: 1px solid var(--line); font-size: var(--fs-xs); color: var(--text-2); }
.ranking li:first-child { border-top: 0; }
.rank { color: var(--text-3); text-align: right; }
.who small { display: block; font-size: var(--fs-2xs); color: var(--text-3); }
.ranking b { font-weight: 650; color: var(--text); }
.extra { font-size: var(--fs-2xs); color: var(--text-3); }
.discipline { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-bottom: var(--s-2); font-size: var(--fs-xs); color: var(--text-2); }
.discipline .card-icon.red { margin-left: var(--s-2); }
.discipline small { color: var(--text-3); }
.meetings { margin-top: var(--s-4); padding-top: var(--s-4); border-top: 1px dashed var(--line-strong); }
.meetings label { display: block; font-size: var(--fs-xs); color: var(--text-3); margin-bottom: var(--s-2); }
select { min-height: 36px; max-width: 100%; padding: 0 var(--s-3); background: var(--surface-2); color: var(--text); border: 1px solid var(--line-strong); border-radius: var(--r-sm); font-size: var(--fs-sm); }
@media (pointer: coarse) { select { min-height: var(--tap); font-size: var(--fs-md); } }
.meetings ul { display: grid; gap: var(--s-3); margin-top: var(--s-3); }
.meeting { display: flex; flex-wrap: wrap; align-items: baseline; gap: 2px var(--s-3); font-size: var(--fs-xs); }
.meeting time, .meeting small { font-size: var(--fs-2xs); color: var(--text-3); }
.meeting b { font-weight: 550; }
.meeting-goals { display: grid; gap: 2px; font-size: var(--fs-2xs); color: var(--text-3); margin-top: 2px; }
@media (max-width: 600px) { .summary { grid-template-columns: repeat(2, 1fr); row-gap: var(--s-4); } }
</style>
