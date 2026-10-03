<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../services/api'
import type { Lineup, LineupPlayer, Match, MatchLineups } from '../types/football'

const props = defineProps<{ match: Match }>()
const dialog = ref<HTMLDialogElement>()
const data = ref<MatchLineups | null>(null)
const loading = ref(false)
const error = ref('')

async function open() {
  dialog.value?.showModal()
  if (data.value || loading.value || !props.match.lineup_ref) return
  loading.value = true; error.value = ''
  try { data.value = await api.lineups(props.match.lineup_ref) } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Não foi possível carregar as escalações.' } finally { loading.value = false }
}
const close = () => dialog.value?.close()
const played = (players: LineupPlayer[]) => players.filter((p) => p.minutes > 0)
const unused = (players: LineupPlayer[]) => players.filter((p) => !p.minutes)
const sides = (d: MatchLineups): [string, Lineup][] => [[props.match.home_team.name, d.home], [props.match.away_team.name, d.away]]
const short: Record<string, string> = { Goleiro: 'GOL', Defensor: 'DEF', 'Meio-campo': 'MEI', Atacante: 'ATA' }
</script>

<template>
  <button v-if="match.lineup_ref" type="button" class="lineups-button" @click="open">
    <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="5.5" cy="5" r="2.2" fill="none" stroke="currentColor" stroke-width="1.2"/><circle cx="11" cy="5.6" r="1.8" fill="none" stroke="currentColor" stroke-width="1.2"/><path d="M1.6 13c.4-2.4 2-3.7 3.9-3.7s3.5 1.3 3.9 3.7M9.6 9.4c.4-.1.9-.2 1.4-.2 1.7 0 3 1.1 3.4 3.3" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
    Ver escalações
  </button>
  <dialog ref="dialog" class="lineups-dialog" :aria-label="`Escalações de ${match.home_team.name} x ${match.away_team.name}`" @click.self="close">
    <div class="lineups-content">
      <header><div><h2>{{ match.home_team.name }} {{ match.home_score }} x {{ match.away_score }} {{ match.away_team.name }}</h2><p>Escalações · dados AlmanacStats</p></div><button type="button" class="lineups-close" aria-label="Fechar" @click="close">×</button></header>
      <div v-if="loading" class="lineups-state" role="status">Carregando jogadores…</div>
      <div v-else-if="error" class="error" role="alert">{{ error }}</div>
      <div v-else-if="data" class="lineups-grid">
        <section v-for="[team, lineup] in sides(data)" :key="team">
          <h3>{{ team }}</h3>
          <p class="lineup-meta"><span v-if="lineup.formation">Formação {{ lineup.formation }}</span><span v-if="lineup.coach">Técnico: {{ lineup.coach }}</span></p>
          <p class="lineup-label">Titulares</p>
          <ol>
            <li v-for="p in lineup.starters" :key="p.number + p.name">
              <span class="lineup-number">{{ p.number }}</span>
              <span class="lineup-name">{{ p.name }} <small>{{ short[p.position] ?? '' }}</small></span>
              <span class="lineup-tags"><b v-for="g in p.goals ?? 0" :key="'g' + g" class="tag-goal" title="Gol">G</b><b v-if="p.assists" class="tag-assist" title="Assistências">A{{ p.assists > 1 ? p.assists : '' }}</b><i v-if="p.yellow" class="card-icon yellow" title="Cartão amarelo"></i><i v-if="p.red" class="card-icon red" title="Cartão vermelho"></i></span>
              <span class="lineup-min" title="Minutos jogados">{{ p.minutes ? `${p.minutes}'` : '' }}</span>
              <span class="lineup-rating" :title="'Nota'">{{ p.rating ? p.rating.toFixed(1) : '–' }}</span>
            </li>
          </ol>
          <template v-if="played(lineup.substitutes).length">
            <p class="lineup-label">Entraram</p>
            <ol>
              <li v-for="p in played(lineup.substitutes)" :key="p.number + p.name">
                <span class="lineup-number">{{ p.number }}</span>
                <span class="lineup-name">{{ p.name }} <small>{{ short[p.position] ?? '' }}</small></span>
                <span class="lineup-tags"><b v-for="g in p.goals ?? 0" :key="'g' + g" class="tag-goal" title="Gol">G</b><b v-if="p.assists" class="tag-assist" title="Assistências">A{{ p.assists > 1 ? p.assists : '' }}</b><i v-if="p.yellow" class="card-icon yellow" title="Cartão amarelo"></i><i v-if="p.red" class="card-icon red" title="Cartão vermelho"></i></span>
                <span class="lineup-min">{{ p.minutes }}'</span>
                <span class="lineup-rating">{{ p.rating ? p.rating.toFixed(1) : '–' }}</span>
              </li>
            </ol>
          </template>
          <p v-if="unused(lineup.substitutes).length" class="lineup-bench">No banco: {{ unused(lineup.substitutes).map((p) => p.name).join(', ') }}</p>
        </section>
      </div>
    </div>
  </dialog>
</template>

<style scoped>
.lineups-button{display:inline-flex;align-items:center;gap:6px;margin-top:12px;border:1px solid #36462d;background:#20291e;border-radius:6px;padding:6px 10px;font-size:10px;color:#bbcdb0}
.lineups-button:hover{border-color:var(--lime);color:var(--lime)}
.lineups-button svg{width:13px;height:13px}
.lineups-dialog{width:min(920px,calc(100vw - 32px));max-height:calc(100vh - 48px);padding:0;border:1px solid var(--line);border-radius:12px;background:var(--panel,#161d17);color:#e4eddf}
.lineups-dialog::backdrop{background:#000a}
.lineups-content{padding:22px}
header{display:flex;justify-content:space-between;gap:16px;align-items:start;margin-bottom:18px}
header h2{font-size:17px}
header p{font-size:10px;color:var(--muted);margin-top:4px}
.lineups-close{border:1px solid var(--line);background:transparent;border-radius:6px;width:30px;height:30px;font-size:18px;line-height:1;color:#c3ceb9}
.lineups-close:hover{border-color:var(--lime);color:var(--lime)}
.lineups-state{font-size:12px;color:var(--muted);padding:30px 0;text-align:center}
.lineups-grid{display:grid;grid-template-columns:1fr 1fr;gap:26px}
h3{font-size:14px}
.lineup-meta{display:flex;flex-wrap:wrap;gap:4px 12px;font-size:10px;color:#93a08b;margin:4px 0 12px}
.lineup-label{font-size:9px;letter-spacing:1px;text-transform:uppercase;color:#6f8362;margin:12px 0 6px}
ol{list-style:none;margin:0;padding:0}
li{display:grid;grid-template-columns:24px 1fr auto 34px 30px;align-items:center;gap:8px;padding:5px 0;border-top:1px solid #283124;font-size:11px}
.lineup-number{color:#9eb390;font-variant-numeric:tabular-nums;text-align:right}
.lineup-name{overflow-wrap:anywhere}
.lineup-name small{font-size:8px;color:#6f8362;margin-left:3px;letter-spacing:.5px}
.lineup-tags{display:inline-flex;gap:3px;align-items:center}
.tag-goal,.tag-assist{font-size:8px;font-weight:700;border-radius:3px;padding:1px 3px}
.tag-goal{background:var(--lime);color:#17220e}
.tag-assist{background:#2f3d25;color:#c4f66b}
.card-icon{width:7px;height:10px;border-radius:1px;display:inline-block}
.card-icon.yellow{background:#f2c94c}
.card-icon.red{background:#e5484d}
.lineup-min{font-size:10px;color:#84917d;text-align:right;font-variant-numeric:tabular-nums}
.lineup-rating{font-size:10px;font-weight:650;text-align:right;color:#dfe8d8;font-variant-numeric:tabular-nums}
.lineup-bench{font-size:10px;color:#84917d;margin-top:10px;line-height:1.6}
@media(max-width:780px){.lineups-grid{grid-template-columns:1fr}.lineups-content{padding:16px}}
</style>
