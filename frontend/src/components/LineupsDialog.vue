<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { api } from '../services/api'
import type { Lineup, LineupPlayer, Match, MatchLineups } from '../types/football'
import { errorMessage } from '../utils/format'

const props = defineProps<{ match: Match }>()
const dialog = ref<HTMLDialogElement>()
const data = ref<MatchLineups | null>(null)
const loading = ref(false)
const error = ref('')

async function load() {
  if (data.value || loading.value || !props.match.lineup_ref) return
  loading.value = true; error.value = ''
  try { data.value = await api.lineups(props.match.lineup_ref) } catch (cause) { error.value = errorMessage(cause, 'Não foi possível carregar as escalações.') } finally { loading.value = false }
}
function open() { dialog.value?.showModal(); void load() }
const close = () => dialog.value?.close()
onBeforeUnmount(close)

const sides = (d: MatchLineups): [string, Lineup][] => [[props.match.home_team.name, d.home], [props.match.away_team.name, d.away]]
const groups = (lineup: Lineup) => [
  { label: 'Titulares', players: lineup.starters },
  { label: 'Entraram', players: lineup.substitutes.filter((p) => p.minutes > 0) },
].filter((g) => g.players.length)
const bench = (lineup: Lineup) => lineup.substitutes.filter((p) => !p.minutes).map((p) => p.name).join(', ')
const short: Record<string, string> = { Goleiro: 'GOL', Defensor: 'DEF', 'Meio-campo': 'MEI', Atacante: 'ATA' }
const key = (p: LineupPlayer) => `${p.number}-${p.name}`
const title = `${props.match.home_team.name} ${props.match.home_score} x ${props.match.away_score} ${props.match.away_team.name}`
const titleId = `lineups-${props.match.id}`
</script>

<template>
  <button v-if="match.lineup_ref" type="button" class="btn btn-secondary lineups-button" aria-haspopup="dialog" @click="open">
    <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="5.5" cy="5" r="2.2" fill="none" stroke="currentColor" stroke-width="1.3" /><circle cx="11" cy="5.6" r="1.8" fill="none" stroke="currentColor" stroke-width="1.3" /><path d="M1.6 13c.4-2.4 2-3.7 3.9-3.7s3.5 1.3 3.9 3.7M9.6 9.4c.4-.1.9-.2 1.4-.2 1.7 0 3 1.1 3.4 3.3" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" /></svg>
    Ver escalações
  </button>
  <dialog v-if="match.lineup_ref" ref="dialog" class="sheet" :aria-labelledby="titleId" @click.self="close">
    <div class="content">
      <header>
        <div><h2 :id="titleId">{{ title }}</h2><p>Escalações · dados AlmanacStats</p></div>
        <button type="button" class="btn btn-icon" aria-label="Fechar" @click="close"><svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true"><path d="m3.5 3.5 9 9m0-9-9 9" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" /></svg></button>
      </header>
      <div class="body">
        <div v-if="loading" class="grid" role="status" aria-label="Carregando escalações">
          <div v-for="i in 2" :key="i"><div class="skeleton sk-title"></div><div v-for="j in 8" :key="j" class="skeleton sk-row"></div></div>
        </div>
        <div v-else-if="error" class="banner banner-error" role="alert">{{ error }}<button type="button" class="btn btn-secondary" @click="load">Tentar de novo</button></div>
        <div v-else-if="data" class="grid">
          <section v-for="[team, lineup] in sides(data)" :key="team">
            <h3>{{ team }}</h3>
            <p class="meta"><span v-if="lineup.formation">Formação {{ lineup.formation }}</span><span v-if="lineup.coach">Técnico: {{ lineup.coach }}</span></p>
            <template v-for="group in groups(lineup)" :key="group.label">
              <p class="eyebrow group">{{ group.label }}</p>
              <ol class="num">
                <li v-for="p in group.players" :key="key(p)">
                  <span class="number">{{ p.number }}</span>
                  <span class="name">{{ p.name }} <abbr :title="p.position">{{ short[p.position] ?? '' }}</abbr></span>
                  <span class="tags">
                    <b v-for="g in p.goals ?? 0" :key="'g' + g" class="tag goal" role="img" aria-label="Gol">G</b>
                    <b v-if="p.assists" class="tag assist" role="img" :aria-label="`${p.assists} assistência${p.assists > 1 ? 's' : ''}`">A{{ p.assists > 1 ? p.assists : '' }}</b>
                    <i v-if="p.yellow" class="card-icon yellow" role="img" aria-label="Cartão amarelo"></i>
                    <i v-if="p.red" class="card-icon red" role="img" aria-label="Cartão vermelho"></i>
                  </span>
                  <span class="minutes"><span class="sr-only">Minutos: </span>{{ p.minutes ? `${p.minutes}'` : '' }}</span>
                  <span class="rating"><span class="sr-only">Nota: </span>{{ p.rating ? p.rating.toFixed(1) : '–' }}</span>
                </li>
              </ol>
            </template>
            <p v-if="bench(lineup)" class="bench">No banco: {{ bench(lineup) }}</p>
          </section>
        </div>
      </div>
    </div>
  </dialog>
</template>

<style scoped>
.lineups-button { min-height: 32px; font-size: var(--fs-xs); font-weight: 500; padding: 0 var(--s-3); }
.lineups-button svg { width: 14px; height: 14px; }

/* Centered modal on desktop, bottom sheet on phones. Opens with a short scale/rise, closes faster. */
.sheet {
  width: min(920px, calc(100vw - 32px)); max-height: min(860px, calc(100dvh - 48px));
  padding: 0; border: 0; border-radius: var(--r-lg); background: var(--surface-1); color: var(--text); box-shadow: var(--shadow-overlay);
  opacity: 0; transform: translateY(8px) scale(.98);
  transition: opacity var(--dur-fast) ease, transform var(--dur-fast) var(--ease-out), overlay var(--dur-fast) allow-discrete, display var(--dur-fast) allow-discrete;
}
.sheet[open] { opacity: 1; transform: none; transition-duration: var(--dur); }
@starting-style { .sheet[open] { opacity: 0; transform: translateY(8px) scale(.98); } }
.sheet::backdrop { background: #000; opacity: 0; transition: opacity var(--dur-fast) ease, overlay var(--dur-fast) allow-discrete, display var(--dur-fast) allow-discrete; }
.sheet[open]::backdrop { opacity: .7; }
@starting-style { .sheet[open]::backdrop { opacity: 0; } }

.content { display: flex; flex-direction: column; max-height: inherit; }
header { display: flex; justify-content: space-between; align-items: start; gap: var(--s-4); padding: var(--s-5) var(--s-5) var(--s-4); border-bottom: 1px solid var(--line); }
header h2 { font-size: var(--fs-md); font-weight: 600; }
header p { font-size: var(--fs-xs); color: var(--text-3); margin-top: 2px; }
.body { padding: var(--s-4) var(--s-5) var(--s-5); overflow-y: auto; overscroll-behavior: contain; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--s-6); }
h3 { font-size: var(--fs-base); font-weight: 600; }
.meta { display: flex; flex-wrap: wrap; gap: 2px var(--s-3); font-size: var(--fs-xs); color: var(--text-3); margin-top: 2px; }
.group { margin: var(--s-4) 0 var(--s-1); }
li { display: grid; grid-template-columns: 24px 1fr auto 34px 30px; align-items: center; gap: var(--s-2); min-height: 32px; border-top: 1px solid var(--line); font-size: var(--fs-xs); }
.number { color: var(--text-3); text-align: right; }
.name { overflow-wrap: anywhere; }
abbr { text-decoration: none; font-size: 10px; letter-spacing: .04em; color: var(--text-3); margin-left: 2px; }
.tags { display: inline-flex; gap: 3px; align-items: center; }
.tag { font-size: 10px; font-weight: 700; border-radius: 3px; padding: 0 4px; line-height: 16px; }
.tag.goal { background: var(--accent); color: var(--accent-ink); }
.tag.assist { background: var(--surface-3); color: var(--accent); }
.minutes { color: var(--text-3); text-align: right; }
.rating { font-weight: 650; text-align: right; }
.bench { font-size: var(--fs-xs); color: var(--text-3); margin-top: var(--s-3); line-height: 1.6; }
.sk-title { height: 20px; width: 50%; margin-bottom: var(--s-4); border-radius: var(--r-sm); }
.sk-row { height: 24px; margin-top: var(--s-2); border-radius: var(--r-sm); }

@media (max-width: 700px) {
  .grid { grid-template-columns: 1fr; gap: var(--s-5); }
  .sheet { width: 100vw; max-width: 100vw; max-height: 88dvh; margin: auto 0 0; border-radius: var(--r-lg) var(--r-lg) 0 0; transform: translateY(100%); transition-timing-function: var(--ease-drawer); }
  .sheet[open] { transform: none; transition-duration: var(--dur-slow); }
  @starting-style { .sheet[open] { transform: translateY(100%); } }
  header { padding: var(--s-4); }
  .body { padding: var(--s-3) var(--s-4) calc(var(--s-5) + env(safe-area-inset-bottom)); }
}
@media (prefers-reduced-motion: reduce) {
  .sheet, .sheet[open] { transform: none; }
  @starting-style { .sheet[open] { transform: none; } }
}
</style>
