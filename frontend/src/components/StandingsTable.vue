<script setup lang="ts">
import { computed } from 'vue'
import type { Standing } from '../types/football'
import TeamBadge from './TeamBadge.vue'

const props = defineProps<{ standings: Standing[]; title: string; subtitle?: string; highlightId?: string; disabled?: boolean }>()
defineEmits<{ select: [name: string] }>()
// Zonas indicativas da Série A com 20 clubes; vagas extras dependem de outras competições.
const zoned = computed(() => props.standings.length === 20)
function zone(position: number) {
  if (!zoned.value) return ''
  if (position <= 4) return 'libertadores'
  if (position <= 6) return 'pre-libertadores'
  if (position <= 12) return 'sul-americana'
  if (position >= 17) return 'rebaixamento'
  return ''
}
function badgeTeam(row: Standing) { return { id: row.team_id, name: row.team_name, short_name: row.team_name.split(/\s+/).map((w) => w[0]).join('').slice(0, 3).toUpperCase(), logo_url: row.logo_url, country: '', stadium: '', coach: '', founded_year: 0 } }
</script>

<template>
  <section class="panel standings-panel">
    <div class="section-title"><h2>{{ title }}</h2><span v-if="subtitle" class="muted">{{ subtitle }}</span></div>
    <div class="standings-scroll">
      <table class="standings">
        <thead><tr><th scope="col" class="pos">#</th><th scope="col" class="club">Clube</th><th scope="col" title="Pontos">P</th><th scope="col" title="Jogos">J</th><th scope="col" title="Vitórias">V</th><th scope="col" title="Empates">E</th><th scope="col" title="Derrotas">D</th><th scope="col" class="wide" title="Gols pró">GP</th><th scope="col" class="wide" title="Gols contra">GC</th><th scope="col" title="Saldo de gols">SG</th></tr></thead>
        <tbody>
          <tr v-for="row in standings" :key="row.team_id" :class="[zone(row.position), { current: row.team_id === highlightId }]" :aria-current="row.team_id === highlightId ? 'true' : undefined">
            <td class="pos"><span>{{ row.position }}</span></td>
            <td class="club"><button type="button" :disabled="disabled" :title="`Analisar ${row.team_name}`" @click="$emit('select', row.team_name)"><TeamBadge :team="badgeTeam(row)" /><span>{{ row.team_name }}</span></button></td>
            <td class="points">{{ row.points }}</td><td>{{ row.played }}</td><td>{{ row.wins }}</td><td>{{ row.draws }}</td><td>{{ row.losses }}</td><td class="wide">{{ row.goals_for }}</td><td class="wide">{{ row.goals_against }}</td><td>{{ row.goal_difference > 0 ? '+' : '' }}{{ row.goal_difference }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p v-if="!standings.length" class="empty">Tabela indisponível.</p>
    <div v-if="zoned" class="standings-legend"><span class="libertadores">Libertadores</span><span class="pre-libertadores">Pré-Libertadores</span><span class="sul-americana">Sul-Americana</span><span class="rebaixamento">Rebaixamento</span><small>Zonas indicativas · critérios: pontos, vitórias, saldo, gols pró</small></div>
  </section>
</template>

<style scoped>
.standings-scroll{overflow-x:auto;margin:0 -4px}
.standings{width:100%;border-collapse:collapse;font-size:12px;font-variant-numeric:tabular-nums}
.standings th{font-size:9px;letter-spacing:1px;color:var(--muted);font-weight:600;padding:0 6px 10px;text-align:center}
.standings td{padding:5px 6px;text-align:center;border-top:1px solid var(--line);color:#b9c6b1}
.standings .club{text-align:left;width:100%}
.standings .club button{display:flex;align-items:center;gap:10px;border:0;background:none;padding:0;text-align:left;color:#dfe8d8;font-size:12px;min-width:150px}
.standings .club button:hover:not(:disabled) span{color:var(--lime)}
.standings .club :deep(.team-badge){width:22px;height:25px;font-size:8px}
.standings .points{color:#eef3e9;font-weight:650}
.pos span{display:inline-grid;place-items:center;width:22px;height:22px;border-radius:5px;font-weight:600}
tr.libertadores .pos span{background:#2c5bc4;color:#fff}
tr.pre-libertadores .pos span{background:#5f86d8;color:#fff}
tr.sul-americana .pos span{background:#2f6f45;color:#e9f7ee}
tr.rebaixamento .pos span{background:#9c3b30;color:#fde9e5}
tr.current td{background:#25391d}
tr.current .club button{color:var(--lime);font-weight:650}
.standings-legend{display:flex;flex-wrap:wrap;gap:8px 16px;margin-top:14px;font-size:10px;color:var(--muted);align-items:center}
.standings-legend span::before{content:'';display:inline-block;width:8px;height:8px;border-radius:2px;margin-right:6px}
.standings-legend .libertadores::before{background:#2c5bc4}
.standings-legend .pre-libertadores::before{background:#5f86d8}
.standings-legend .sul-americana::before{background:#2f6f45}
.standings-legend .rebaixamento::before{background:#9c3b30}
.standings-legend small{flex-basis:100%;font-size:9px;color:#6f8362}
@media(max-width:780px){.standings .wide{display:none}.standings td,.standings th{padding-left:4px;padding-right:4px}}
</style>
