<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '../services/api'
import type { Fixture, Simulation, SimulationFactor } from '../types/football'
import { errorMessage, outcomeLabel, outcomeLetter } from '../utils/format'
import { useRotatingText } from '../composables/useRotatingText'

const props = defineProps<{ fixture: Fixture; teamId: string }>()
const emit = defineEmits<{ explain: [question: string] }>()

const RUN_OPTIONS = [50, 1000, 10000]
const factorLabels: Record<string, string> = { season: 'Força na temporada (base)', home_edge: 'Mando de campo (média da liga)', season_detail: 'Estatísticas da temporada', venue: 'Campanha em casa e fora', form: 'Últimos 5 jogos', rest: 'Sequência e descanso', injuries: 'Lesionados', suspensions: 'Suspensões por cartão', head_to_head: 'Confrontos históricos', ai_analyst: 'Análise da IA (Gemma local)', randomness: 'Aleatoriedade' }
const runs = ref(RUN_OPTIONS[0])
const result = ref<Simulation | null>(null)
const loading = ref(false)
// Shown in turn while the local model works; roughly the order the backend goes through.
const WAITING_PHRASES = [
  'A IA local está analisando os dados dos dois times…',
  'Comparando ataque e defesa com a média da liga…',
  'Conferindo a campanha em casa e fora de casa…',
  'Revendo os últimos 5 jogos de cada lado…',
  'Checando o descanso e a sequência de jogos…',
  'Puxando o histórico de confrontos desde 2003…',
  'Pesando finalizações no alvo e defesas do goleiro…',
  'Procurando o que os números ainda não contam…',
  'O Gemma está pensando aqui no seu processador, sem nuvem…',
  'Ajustando a força de cada lado antes do sorteio…',
  'Aquecendo os dados para o sorteio das simulações…',
  'Quase lá: no processador isso pode levar 1–2 minutos.',
] as const
const waiting = useRotatingText(WAITING_PHRASES, loading)
const error = ref('')
const runsLabel = (n: number) => n.toLocaleString('pt-BR')

const home = computed(() => props.fixture.home_team.id === props.teamId)
const club = computed(() => home.value ? props.fixture.home_team : props.fixture.away_team)
const opponent = computed(() => home.value ? props.fixture.away_team : props.fixture.home_team)
const outcomes = computed(() => result.value && [
  { key: 'W', label: `Vitória ${club.value.name}`, value: result.value.win_pct },
  { key: 'D', label: 'Empate', value: result.value.draw_pct },
  { key: 'L', label: `Vitória ${opponent.value.name}`, value: result.value.loss_pct },
])
const effect = (m: number) => Math.round((m - 1) * 100) === 0 ? '0%' : `${m > 1 ? '+' : '−'}${Math.round(Math.abs(m - 1) * 100)}%`
// What each factor cell shows: base expected goals, the noise range, or the % change; '—' when not used.
function cell(f: SimulationFactor, side: 'club' | 'opponent') {
  if (!f.available) return '—'
  if (f.key === 'season') return `${decimal(side === 'club' ? f.club_value ?? 0 : f.opponent_value ?? 0)} gol`
  if (f.key === 'randomness') return '±20%'
  return effect(f[side])
}
const analyst = computed(() => result.value?.factors.find((f) => f.key === 'ai_analyst' && f.available))
const decimal = (n: number) => n.toLocaleString('pt-BR', { maximumFractionDigits: 2 })

async function simulate() {
  loading.value = true; error.value = ''
  try { result.value = await api.simulate(props.teamId, runs.value) } catch (cause) { error.value = errorMessage(cause, 'Não foi possível simular a partida.') } finally { loading.value = false }
}
// A different club or fixture invalidates the previous result.
watch(() => [props.teamId, props.fixture.date], () => { result.value = null; error.value = '' })

function explain() {
  if (!result.value) return
  emit('explain', `Explique a simulação do próximo jogo contra o ${opponent.value.name} com ${runsLabel(result.value.runs)} simulações: as chances em %, os placares mais prováveis e os fatores que mais pesaram.`)
}
</script>

<template>
  <section class="simulation" aria-labelledby="sim-title">
    <div class="head">
      <div>
        <h3 id="sim-title">Simulação do jogo</h3>
        <p class="muted">A IA local revisa os dados da temporada, forma, sequência e confrontos; o app sorteia os jogos · estimativa, não certeza</p>
      </div>
      <div class="controls">
        <div class="runs" role="radiogroup" aria-label="Número de simulações">
          <button v-for="n in RUN_OPTIONS" :key="n" type="button" role="radio" :aria-checked="runs === n" :class="{ active: runs === n }" :disabled="loading" @click="runs = n">{{ runsLabel(n) }}</button>
        </div>
        <!-- aria-disabled instead of disabled: a disabled button drops keyboard focus to the page. -->
        <button type="button" class="btn btn-primary" :aria-disabled="loading" :aria-busy="loading" @click="!loading && simulate()">
          <span v-if="loading" class="spinner" aria-hidden="true"></span>{{ loading ? 'Simulando' : `Simular ${runsLabel(runs)}×` }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="waiting">
      <!-- Screen readers get one stable message; the rotating phrases are visual only. -->
      <p class="sr-only" role="status">A IA local está analisando os dados dos dois times. No processador isso pode levar 1 a 2 minutos.</p>
      <span class="dots" aria-hidden="true"><i></i><i></i><i></i></span>
      <Transition name="phrase" mode="out-in">
        <span :key="waiting.index.value" class="phrase" aria-hidden="true">{{ WAITING_PHRASES[waiting.index.value] }}</span>
      </Transition>
      <span class="elapsed num" aria-hidden="true">{{ waiting.elapsed.value }}s</span>
    </div>
    <div v-if="error" class="banner banner-error" role="alert">{{ error }}<button type="button" class="btn btn-secondary" @click="simulate">Tentar de novo</button></div>

    <div v-if="result && outcomes" :key="result.seed" class="outcome-panel" aria-live="polite">
      <div class="bar" role="img" :aria-label="outcomes.map((o) => `${o.label} ${o.value}%`).join(', ')">
        <span v-for="o in outcomes" :key="o.key" :class="['seg', o.key]" :style="{ flexGrow: o.value }"></span>
      </div>
      <ul class="outcomes num">
        <li v-for="o in outcomes" :key="o.key"><span class="pct"><span :class="['dot', o.key]" aria-hidden="true"></span><strong>{{ decimal(o.value) }}%</strong></span><span class="lbl">{{ o.label }}</span></li>
      </ul>

      <blockquote v-if="analyst" class="analyst">
        <p class="eyebrow">Análise da IA · ajuste {{ effect(analyst.club) }} / {{ effect(analyst.opponent) }}</p>
        <p>{{ analyst.detail }}</p>
        <p class="analyst-note">Texto gerado pela IA local (Gemma); números conferidos com os dados, mas a redação pode conter imprecisões.</p>
      </blockquote>

      <div class="facts">
        <p class="num"><span class="label">Gols esperados</span> {{ club.name }} {{ decimal(result.expected_goals_club) }} × {{ decimal(result.expected_goals_opponent) }} {{ opponent.name }}</p>
        <div class="last5">
          <span class="label">Últimos 5</span>
          <span class="seq"><span class="who">{{ club.name }}</span><span v-for="(r, i) in result.club_last5" :key="'c' + i" :class="['result sm', r]" role="img" :aria-label="outcomeLabel[r]">{{ outcomeLetter[r] }}</span></span>
          <span class="seq"><span class="who">{{ opponent.name }}</span><span v-for="(r, i) in result.opponent_last5" :key="'o' + i" :class="['result sm', r]" role="img" :aria-label="outcomeLabel[r]">{{ outcomeLetter[r] }}</span></span>
        </div>
        <div v-if="result.top_scorelines.length" class="scorelines">
          <span class="label">Placares mais simulados</span>
          <ul class="num"><li v-for="s in result.top_scorelines" :key="`${s.club}-${s.opponent}`"><b>{{ s.club }}–{{ s.opponent }}</b> {{ decimal(s.percent) }}%</li></ul>
        </div>
      </div>

      <table class="factors">
        <caption class="eyebrow">Fatores · efeito nos gols esperados (0% = sem efeito, motivo no texto)</caption>
        <thead><tr><th scope="col">Fator</th><th scope="col" class="num">{{ club.short_name || club.name }}</th><th scope="col" class="num">{{ opponent.short_name || opponent.name }}</th></tr></thead>
        <tbody>
          <tr v-for="f in result.factors" :key="f.key" :class="{ off: !f.available }">
            <th scope="row"><span class="name">{{ factorLabels[f.key] ?? f.key }}<small v-if="!f.available"> · não usado</small></span><span class="detail">{{ f.detail }}</span></th>
            <td :class="['num', { up: f.club > 1.005, down: f.club < 0.995 }]">{{ cell(f, 'club') }}</td>
            <td :class="['num', { up: f.opponent > 1.005, down: f.opponent < 0.995 }]">{{ cell(f, 'opponent') }}</td>
          </tr>
        </tbody>
      </table>

      <ul v-if="result.notes.length" class="notes"><li v-for="n in result.notes" :key="n">{{ n }}</li></ul>
      <button type="button" class="btn btn-secondary explain" @click="explain"><span aria-hidden="true">✳</span> Explicar com a IA</button>
    </div>
  </section>
</template>

<style scoped>
.simulation { grid-column: 1 / -1; display: grid; gap: var(--s-4); padding-top: var(--s-4); border-top: 1px solid var(--line); }
.head { display: flex; flex-wrap: wrap; align-items: flex-end; justify-content: space-between; gap: var(--s-3); }
h3 { font-size: var(--fs-base); font-weight: 600; }
.controls { display: flex; flex-wrap: wrap; align-items: center; gap: var(--s-2); }
.runs { display: inline-flex; padding: 2px; border: 1px solid var(--line-strong); border-radius: var(--r-sm); }
.runs button { min-height: 30px; padding: 0 var(--s-3); border-radius: 4px; font-size: var(--fs-xs); color: var(--text-2); font-variant-numeric: tabular-nums; transition: background-color var(--dur-fast) ease, color var(--dur-fast) ease; }
.btn[aria-disabled="true"] { opacity: .6; cursor: progress; }
.runs button.active { background: var(--surface-3); color: var(--text); font-weight: 600; }
@media (pointer: coarse) { .runs button { min-height: 40px; } }
.spinner { width: 14px; height: 14px; border-radius: 50%; border: 2px solid currentColor; border-right-color: transparent; animation: spin .6s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.outcome-panel { display: grid; gap: var(--s-4); animation: rise var(--dur) var(--ease-out); }
.bar { display: flex; gap: 2px; height: 12px; border-radius: var(--r-full); overflow: hidden; }
.seg { flex-basis: 0; min-width: 2px; }
.seg.W, .dot.W { background: var(--result-w); }
.seg.D, .dot.D { background: var(--result-d); }
.seg.L, .dot.L { background: var(--result-l); }
.outcomes { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--s-3); }
.outcomes li { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.outcomes li:nth-child(2) { align-items: center; text-align: center; }
.outcomes li:last-child { align-items: flex-end; text-align: right; }
.pct { display: inline-flex; align-items: center; gap: 6px; }
.outcomes strong { font-size: var(--fs-xl); font-weight: 600; letter-spacing: -.02em; }
.lbl { font-size: var(--fs-xs); color: var(--text-3); overflow-wrap: anywhere; }
.dot { width: 8px; height: 8px; border-radius: 50%; }

.facts { display: grid; gap: var(--s-3); font-size: var(--fs-sm); }
.label { display: inline-block; min-width: 168px; font-size: var(--fs-xs); color: var(--text-3); }
.last5 { display: flex; flex-wrap: wrap; align-items: center; gap: var(--s-2) var(--s-4); }
.last5 .label { min-width: 152px; }
.seq { display: inline-flex; align-items: center; gap: 4px; }
.who { font-size: var(--fs-xs); color: var(--text-2); margin-right: 4px; }
.scorelines { display: flex; flex-wrap: wrap; align-items: center; gap: var(--s-2); }
.scorelines .label { min-width: 152px; }
.scorelines ul { display: flex; flex-wrap: wrap; gap: var(--s-2); }
.scorelines li { padding: 2px var(--s-2); border-radius: var(--r-full); background: var(--surface-2); font-size: var(--fs-xs); color: var(--text-2); }
.scorelines b { color: var(--text); margin-right: 2px; }

.factors { width: 100%; border-collapse: collapse; font-size: var(--fs-xs); }
.factors caption { text-align: left; margin-bottom: var(--s-2); }
.factors thead th { text-align: right; font-size: var(--fs-2xs); color: var(--text-3); font-weight: 600; padding: 0 0 var(--s-2) var(--s-3); white-space: nowrap; }
.factors thead th:first-child { text-align: left; padding-left: 0; }
.factors tbody th, .factors td { border-top: 1px solid var(--line); padding: var(--s-2) 0; vertical-align: top; }
.factors tbody th { text-align: left; font-weight: 400; }
.factors .name { display: block; color: var(--text); font-weight: 550; }
.factors .name small { color: var(--text-3); font-weight: 400; }
.factors .detail { display: block; color: var(--text-3); margin-top: 2px; line-height: 1.5; }
.factors td { text-align: right; padding-left: var(--s-3); white-space: nowrap; color: var(--text-3); width: 1%; }
.factors td.up { color: var(--accent); }
.factors td.down { color: var(--danger); }
.factors tr.off .name { color: var(--text-3); }
.notes { display: grid; gap: var(--s-1); font-size: var(--fs-2xs); color: var(--warn); }
.explain { justify-self: start; }
.waiting { display: flex; align-items: center; gap: var(--s-2); min-height: 20px; font-size: var(--fs-xs); color: var(--text-2); }
.phrase { flex: 1; min-width: 0; }
.elapsed { color: var(--text-3); }
/* Each phrase leaves upward and the next rises in: short, interruptible, movement-free when reduced. */
.phrase-enter-active { transition: opacity 220ms var(--ease-out), transform 220ms var(--ease-out); }
.phrase-leave-active { transition: opacity 140ms ease, transform 140ms ease; }
.phrase-enter-from { opacity: 0; transform: translateY(6px); }
.phrase-leave-to { opacity: 0; transform: translateY(-6px); }
@media (prefers-reduced-motion: reduce) { .phrase-enter-from, .phrase-leave-to { transform: none; } }
.dots { display: inline-flex; gap: 3px; }
.dots i { width: 5px; height: 5px; border-radius: 50%; background: var(--accent); animation: pulse 1.2s ease-in-out infinite; }
.dots i:nth-child(2) { animation-delay: .15s; }
.dots i:nth-child(3) { animation-delay: .3s; }
@keyframes pulse { 0%, 100% { opacity: .25; } 50% { opacity: 1; } }
.analyst { margin: 0; padding: var(--s-3) var(--s-4); border-left: 2px solid var(--accent); background: var(--accent-soft); border-radius: 0 var(--r-sm) var(--r-sm) 0; display: grid; gap: var(--s-1); }
.analyst .eyebrow { color: var(--accent); }
.analyst p:nth-child(2) { font-size: var(--fs-sm); line-height: 1.6; }
.analyst-note { font-size: var(--fs-2xs); color: var(--text-3); }
@container (max-width: 640px) {
  .label, .last5 .label, .scorelines .label { min-width: 0; display: block; width: 100%; }
  .outcomes strong { font-size: var(--fs-lg); }
}
</style>
