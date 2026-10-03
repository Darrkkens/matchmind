<script setup lang="ts">
import { computed } from 'vue'
import type { Standing } from '../types/football'
import { initials } from '../utils/format'
import TeamBadge from './TeamBadge.vue'

const props = defineProps<{ standings: Standing[]; title: string; subtitle?: string; highlightId?: string; disabled?: boolean }>()
defineEmits<{ select: [name: string] }>()

// Zonas indicativas da Série A com 20 clubes; vagas extras dependem de outras competições.
const zones = [
  { id: 'libertadores', label: 'Libertadores', test: (p: number) => p <= 4 },
  { id: 'pre-libertadores', label: 'Pré-Libertadores', test: (p: number) => p <= 6 },
  { id: 'sul-americana', label: 'Sul-Americana', test: (p: number) => p <= 12 },
  { id: 'rebaixamento', label: 'Rebaixamento', test: (p: number) => p >= 17 },
] as const
const zoned = computed(() => props.standings.length === 20)
const zone = (position: number) => zoned.value ? zones.find((z) => z.test(position)) : undefined
const signed = (n: number) => `${n > 0 ? '+' : ''}${n}`
const columns = [
  { key: 'played', label: 'J', title: 'Jogos' },
  { key: 'wins', label: 'V', title: 'Vitórias' },
  { key: 'draws', label: 'E', title: 'Empates' },
  { key: 'losses', label: 'D', title: 'Derrotas' },
  { key: 'goals_for', label: 'GP', title: 'Gols pró', wide: true },
  { key: 'goals_against', label: 'GC', title: 'Gols contra', wide: true },
] as const
</script>

<template>
  <section class="standings-section" aria-labelledby="standings-title">
    <div class="section-head"><h2 id="standings-title">{{ title }}</h2><span v-if="subtitle" class="muted">{{ subtitle }}</span></div>
    <div v-if="standings.length" class="scroll">
      <table class="num">
        <thead>
          <tr>
            <th scope="col" class="pos"><abbr title="Posição">#</abbr></th>
            <th scope="col" class="club">Clube</th>
            <th scope="col"><abbr title="Pontos">P</abbr></th>
            <th v-for="c in columns" :key="c.key" scope="col" :class="{ wide: 'wide' in c }"><abbr :title="c.title">{{ c.label }}</abbr></th>
            <th scope="col"><abbr title="Saldo de gols">SG</abbr></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in standings" :key="row.team_id" :class="{ current: row.team_id === highlightId }" :aria-current="row.team_id === highlightId ? 'true' : undefined">
            <td class="pos"><span :class="['pos-mark', zone(row.position)?.id]" :title="zone(row.position)?.label">{{ row.position }}</span></td>
            <td class="club">
              <button type="button" :disabled="disabled" @click="$emit('select', row.team_name)">
                <TeamBadge :team="{ name: row.team_name, short_name: initials(row.team_name), logo_url: row.logo_url }" size="xs" />
                <span class="name">{{ row.team_name }}</span>
                <span class="sr-only">— analisar este clube</span>
              </button>
            </td>
            <td class="points">{{ row.points }}</td>
            <td v-for="c in columns" :key="c.key" :class="{ wide: 'wide' in c }">{{ row[c.key] }}</td>
            <td>{{ signed(row.goal_difference) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p v-else class="empty">Tabela indisponível.</p>
    <div v-if="zoned" class="legend">
      <ul><li v-for="z in zones" :key="z.id"><span :class="['swatch', z.id]" aria-hidden="true"></span>{{ z.label }}</li></ul>
      <p>Zonas indicativas · critérios: pontos, vitórias, saldo, gols pró</p>
    </div>
  </section>
</template>

<style scoped>
.standings-section { container-type: inline-size; min-width: 0; }
.scroll { overflow-x: auto; margin: 0 calc(var(--s-2) * -1); padding: 0 var(--s-2); }
table { width: 100%; border-collapse: collapse; font-size: var(--fs-sm); }
abbr { text-decoration: none; }
th { font-size: var(--fs-2xs); font-weight: 600; letter-spacing: var(--tracking-label); color: var(--text-3); padding: 0 var(--s-2) var(--s-2); text-align: center; }
td { height: 40px; padding: 0 var(--s-2); text-align: center; border-top: 1px solid var(--line); color: var(--text-2); white-space: nowrap; }
.club { text-align: left; width: 100%; max-width: 0; }
.pos { width: 1%; padding-left: 0; }
.pos-mark { display: inline-grid; place-items: center; width: 24px; height: 24px; border-radius: 5px; font-weight: 600; color: var(--text-2); }
.pos-mark.libertadores { background: var(--zone-libertadores); color: #fff; }
.pos-mark.pre-libertadores { background: var(--zone-pre-libertadores); color: #fff; }
.pos-mark.sul-americana { background: var(--zone-sul-americana); color: #eefaf2; }
.pos-mark.rebaixamento { background: var(--zone-rebaixamento); color: #fff2ef; }
.club button { display: flex; align-items: center; gap: var(--s-3); width: 100%; min-height: 40px; text-align: left; color: var(--text); border-radius: var(--r-sm); transition: color var(--dur-fast) ease; }
.club button:disabled { cursor: progress; }
.name { min-width: 0; overflow: hidden; text-overflow: ellipsis; }
.points { color: var(--text); font-weight: 700; }
tbody tr { transition: background-color var(--dur-fast) ease; }
@media (hover: hover) and (pointer: fine) {
  tbody tr:hover { background: var(--surface-2); }
  .club button:hover:not(:disabled) { color: var(--accent); }
}
tr.current { background: var(--accent-soft); }
tr.current td { color: var(--text); }
tr.current .club button { color: var(--accent); font-weight: 650; }
.legend { display: flex; flex-direction: column; gap: var(--s-2); margin-top: var(--s-4); font-size: var(--fs-xs); color: var(--text-3); }
.legend ul { display: flex; flex-wrap: wrap; gap: var(--s-2) var(--s-4); color: var(--text-2); }
.legend li { display: inline-flex; align-items: center; gap: var(--s-2); }
.swatch { width: 10px; height: 10px; border-radius: 3px; }
.swatch.libertadores { background: var(--zone-libertadores); }
.swatch.pre-libertadores { background: var(--zone-pre-libertadores); }
.swatch.sul-americana { background: var(--zone-sul-americana); }
.swatch.rebaixamento { background: var(--zone-rebaixamento); }
@container (max-width: 520px) {
  .wide { display: none; }
  th, td { padding-left: 5px; padding-right: 5px; }
  .club button { gap: var(--s-2); }
}
</style>
