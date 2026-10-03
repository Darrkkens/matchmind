<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ClubHistory, HistoricMatch } from '../types/football'

const props = defineProps<{ history: ClubHistory; teamName: string }>()
const period = computed(() => `${props.history.first_season}–${props.history.last_season}`)
const allTime = computed(() => props.history.all_time)
const percentage = (r: { played: number; wins: number; draws: number }) => r.played ? Math.round((r.wins * 3 + r.draws) / (r.played * 3) * 1000) / 10 : 0
const date = (value: string) => new Intl.DateTimeFormat('pt-BR', { timeZone: 'UTC' }).format(new Date(value))
const scorers = (m: HistoricMatch, side: 'home' | 'away') => (m.goals ?? []).filter((g) => g.side === side).map((g) => `${g.player} ${g.minute}'${g.kind === 'own_goal' ? ' (contra)' : g.kind === 'penalty' ? ' (pên.)' : ''}`).join(', ')
const opponents = computed(() => props.history.head_to_head.filter((r) => r.meetings?.length))
const opponentId = ref(opponents.value[0]?.opponent_id ?? '')
const selected = computed(() => opponents.value.find((r) => r.opponent_id === opponentId.value) ?? opponents.value[0])
const withAverages = computed(() => (props.history.seasons ?? []).some((s) => s.averages))
</script>

<template>
  <section class="panel history-panel">
    <div class="section-title"><h2>Histórico na Série A</h2><span class="eyebrow">{{ period }}</span></div>
    <p v-if="!history.dataset_name" class="empty">O {{ teamName }} não disputou a Série A entre {{ period }} neste conjunto de dados.</p>
    <template v-else>
      <div class="history-summary">
        <div><strong>{{ history.titles.length }}</strong><span>{{ history.titles.length === 1 ? 'título' : 'títulos' }}</span></div>
        <div><strong>{{ history.seasons_played }}</strong><span>temporadas</span></div>
        <div><strong>{{ allTime.played }}</strong><span>jogos</span></div>
        <div><strong class="lime">{{ percentage(allTime) }}<small>%</small></strong><span>aproveitamento</span></div>
      </div>
      <p class="history-record">{{ allTime.wins }} vitórias · {{ allTime.draws }} empates · {{ allTime.losses }} derrotas · {{ allTime.goals_for }} gols pró · {{ allTime.goals_against }} contra</p>

      <details v-if="history.seasons?.length" class="history-block">
        <summary>Temporada a temporada <span aria-hidden="true">+</span></summary>
        <div class="h-scroll">
          <table>
            <thead><tr><th scope="col">Ano</th><th scope="col" title="Posição final">Pos.</th><th scope="col" title="Pontos">Pts</th><th scope="col" title="Vitórias, empates e derrotas">V-E-D</th><th scope="col" title="Gols pró e contra">Gols</th><th scope="col">Técnico</th><th v-if="withAverages" scope="col" title="Médias por jogo: posse · finalizações (no gol) · escanteios">Médias</th></tr></thead>
            <tbody>
              <tr v-for="s in history.seasons" :key="s.season" :class="{ champion: s.position === 1 }">
                <td>{{ s.season }}</td>
                <td>{{ s.position ? `${s.position}º` : '–' }}</td>
                <td>{{ s.points }}</td>
                <td>{{ s.wins }}-{{ s.draws }}-{{ s.losses }}</td>
                <td>{{ s.goals_for }}:{{ s.goals_against }}</td>
                <td class="wrap" :title="s.coaches?.map((c) => `${c.name} (${c.matches} jogos)`).join('\n')">{{ s.coaches?.[0]?.name ?? '–' }}<small v-if="(s.coaches?.length ?? 0) > 1"> +{{ (s.coaches?.length ?? 1) - 1 }}</small></td>
                <td v-if="withAverages" class="avg">{{ s.averages ? `${s.averages.possession}% · ${s.averages.shots} (${s.averages.shots_on_target}) · ${s.averages.corners}` : '–' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="history-note">Pos. calculada pela tabela final · técnico com mais jogos (passe o mouse para ver todos){{ withAverages ? ' · médias por jogo: posse · finalizações (no gol) · escanteios, disponíveis de 2017 a 2023' : '' }}.</p>
      </details>

      <details v-if="history.top_scorers?.length" class="history-block">
        <summary>Artilheiros na Série A <span aria-hidden="true">+</span></summary>
        <ol class="scorers">
          <li v-for="(p, i) in history.top_scorers" :key="p.player"><span class="rank">{{ i + 1 }}</span><span class="who">{{ p.player }}<small>{{ p.seasons }}</small></span><b>{{ p.goals }}</b><small class="pens">{{ p.penalties ? `${p.penalties} pên.` : '' }}</small></li>
        </ol>
        <p class="history-note">Gols pelo clube na Série A desde 2014 (sem gols contra). Nomes como registrados na fonte.</p>
      </details>

      <details v-if="history.discipline" class="history-block">
        <summary>Disciplina <span aria-hidden="true">+</span></summary>
        <p class="discipline-total"><i class="card-icon yellow"></i>{{ history.discipline.yellow }} amarelos <i class="card-icon red"></i>{{ history.discipline.red }} vermelhos <small>({{ history.discipline.first_season }}–{{ history.discipline.last_season }})</small></p>
        <ol class="scorers">
          <li v-for="(p, i) in history.discipline.most_booked" :key="p.player"><span class="rank">{{ i + 1 }}</span><span class="who">{{ p.player }}</span><b>{{ p.yellow }}</b><small class="pens">{{ p.red ? `${p.red} verm.` : '' }}</small></li>
        </ol>
        <p class="history-note">Jogadores com mais cartões pelo clube na Série A.</p>
      </details>

      <details v-if="history.head_to_head.length" class="history-block">
        <summary>Confrontos com os adversários atuais <span aria-hidden="true">+</span></summary>
        <div class="h-scroll">
          <table>
            <thead><tr><th scope="col">Adversário</th><th scope="col" title="Jogos">J</th><th scope="col" title="Vitórias">V</th><th scope="col" title="Empates">E</th><th scope="col" title="Derrotas">D</th><th scope="col" title="Gols pró e contra">Gols</th></tr></thead>
            <tbody><tr v-for="row in history.head_to_head" :key="row.opponent_id"><td>{{ row.opponent_name }}</td><td>{{ row.played }}</td><td>{{ row.wins }}</td><td>{{ row.draws }}</td><td>{{ row.losses }}</td><td>{{ row.goals_for }}:{{ row.goals_against }}</td></tr></tbody>
          </table>
        </div>
        <div v-if="opponents.length" class="meetings">
          <label :for="`meetings-${teamName}`">Últimos jogos contra</label>
          <select :id="`meetings-${teamName}`" v-model="opponentId">
            <option v-for="row in opponents" :key="row.opponent_id" :value="row.opponent_id">{{ row.opponent_name }}</option>
          </select>
          <ul v-if="selected">
            <li v-for="m in selected.meetings" :key="m.date">
              <div class="meeting-line"><span>{{ date(m.date) }}</span><b>{{ m.home_team }} {{ m.home_score }}–{{ m.away_score }} {{ m.away_team }}</b><small v-if="m.stadium">{{ m.stadium }}</small></div>
              <p v-if="m.goals?.length" class="meeting-goals"><span v-if="scorers(m, 'home')">{{ m.home_team }}: {{ scorers(m, 'home') }}</span><span v-if="scorers(m, 'away')">{{ m.away_team }}: {{ scorers(m, 'away') }}</span></p>
            </li>
          </ul>
        </div>
      </details>
    </template>
    <p v-if="history.unavailable?.length" class="history-note">Parte do histórico está indisponível agora: {{ history.unavailable.join(', ') }}.</p>
    <p class="history-source">Fonte: <a :href="history.source_url" target="_blank" rel="noopener noreferrer">{{ history.source }} ↗</a> (GPL-2.0). Somente Série A; títulos e posições calculados pela tabela final de cada temporada.</p>
  </section>
</template>

<style scoped>
.history-summary{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}
.history-summary div{display:flex;flex-direction:column;gap:4px}
.history-summary strong{font-size:24px;font-weight:500;letter-spacing:-.5px;font-variant-numeric:tabular-nums}
.history-summary small{font-size:16px}
.history-summary span{font-size:10px;color:var(--muted)}
.history-record{font-size:11px;color:#99a691;margin-top:12px}
.history-block{border-top:1px solid var(--line);margin-top:14px;padding-top:12px;font-size:11px;color:#8c9b82}
.history-block>summary{cursor:pointer;list-style:none;display:flex;justify-content:space-between}
.history-block>summary::-webkit-details-marker{display:none}
.history-block>summary:hover{color:var(--lime)}
.h-scroll{overflow-x:auto;margin-top:10px}
table{width:100%;border-collapse:collapse;font-variant-numeric:tabular-nums}
th{font-size:9px;letter-spacing:1px;color:var(--muted);font-weight:600;text-align:left;padding:4px 6px;white-space:nowrap}
td{border-top:1px solid #283124;padding:6px;color:#c3ceb9;white-space:nowrap}
td.wrap{white-space:normal;min-width:110px}
td small{color:#6f8362}
td.avg{font-size:10px;color:#93a08b}
tr.champion td{color:var(--lime)}
.history-note{font-size:9px;color:#6f8362;margin-top:8px;line-height:1.6}
.scorers{list-style:none;margin:10px 0 0;padding:0}
.scorers li{display:grid;grid-template-columns:18px 1fr auto 52px;gap:8px;align-items:center;padding:5px 0;border-top:1px solid #283124;color:#c3ceb9}
.scorers .rank{color:#6f8362;text-align:right}
.scorers .who small{display:block;font-size:9px;color:#6f8362}
.scorers b{font-weight:650;color:#dfe8d8}
.scorers .pens{font-size:9px;color:#6f8362}
.discipline-total{display:flex;align-items:center;gap:6px;margin-top:10px;color:#c3ceb9}
.discipline-total small{color:#6f8362}
.card-icon{width:7px;height:10px;border-radius:1px;display:inline-block;margin-left:6px}
.card-icon:first-child{margin-left:0}
.card-icon.yellow{background:#f2c94c}
.card-icon.red{background:#e5484d}
.meetings{margin-top:12px;border-top:1px dashed #283124;padding-top:10px}
.meetings label{font-size:10px;color:#93a08b;margin-right:8px}
.meetings select{background:#20291e;color:#dce8d3;border:1px solid #425637;border-radius:6px;padding:5px 8px;font:inherit;font-size:11px;max-width:100%}
.meetings select:focus-visible{outline:2px solid var(--lime);outline-offset:2px}
.meetings ul{list-style:none;margin:8px 0 0;padding:0;display:grid;gap:8px}
.meeting-line{display:flex;flex-wrap:wrap;gap:4px 10px;align-items:baseline;color:#c3ceb9}
.meeting-line span,.meeting-line small{font-size:9px;color:#6f8362}
.meeting-goals{display:grid;gap:2px;font-size:9px;color:#93a08b;margin-top:2px}
.history-source{font-size:9px;color:#6f8362;margin-top:14px;line-height:1.6}
.history-source a{color:var(--lime);text-decoration:underline;text-underline-offset:2px}
@media(max-width:780px){.history-summary{grid-template-columns:repeat(2,1fr)}}
</style>
