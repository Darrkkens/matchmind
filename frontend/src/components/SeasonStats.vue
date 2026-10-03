<script setup lang="ts">
import { computed } from 'vue'
import type { SeasonStats } from '../types/football'
import { formatDay } from '../utils/format'
import { formatMetric, formatNumber, groups, metricInfo, rankTier, recordLine } from '../utils/season'

const props = defineProps<{ stats: SeasonStats }>()
const grouped = computed(() => groups.map((g) => ({ ...g, metrics: props.stats.metrics.filter((m) => m.group === g.id && metricInfo[m.key]) })).filter((g) => g.metrics.length))
const splits = computed(() => [
  { label: 'Em casa', record: props.stats.home },
  { label: 'Fora de casa', record: props.stats.away },
].filter((s) => s.record))
</script>

<template>
  <section aria-labelledby="season-title">
    <div class="section-head">
      <h2 id="season-title">Temporada</h2>
      <span class="muted">{{ stats.matches }} jogos · dados até {{ formatDay(stats.as_of) }} · <a class="link" :href="stats.source_url" target="_blank" rel="noopener noreferrer">{{ stats.source }}</a></span>
    </div>

    <div class="groups">
      <div v-for="group in grouped" :key="group.id" class="group">
        <h3 class="eyebrow">{{ group.label }}</h3>
        <dl>
          <div v-for="m in group.metrics" :key="m.key" class="metric">
            <dt>{{ metricInfo[m.key].label }}</dt>
            <dd class="value num">{{ formatMetric(m.key, m.value) }}</dd>
            <dd class="league num"><span class="sr-only">Média da liga: </span>liga {{ formatMetric(m.key, m.league) }}</dd>
            <dd :class="['rank num', rankTier(m)]" :title="m.lower_is_better ? 'Menor é melhor' : 'Maior é melhor'"><span class="sr-only">Posição: </span>{{ m.rank }}º<span class="sr-only"> de {{ m.clubs }}</span></dd>
          </div>
        </dl>
      </div>
    </div>

    <div class="extras">
      <div v-if="splits.length" class="splits">
        <div v-for="s in splits" :key="s.label" class="split">
          <h3 class="eyebrow">{{ s.label }}</h3>
          <p class="split-ppm num"><strong>{{ formatMetric('ppm', s.record!.points_per_match) }}</strong> pts/jogo</p>
          <p class="split-line num">{{ recordLine(s.record!) }} · {{ s.record!.goals_for }}:{{ s.record!.goals_against }} gols</p>
        </div>
      </div>
      <dl class="leaders">
        <div v-if="stats.top_scorer"><dt>Artilheiro</dt><dd>{{ stats.top_scorer.player }} <span class="num">· {{ stats.top_scorer.value }} gols</span></dd></div>
        <div v-if="stats.top_assists"><dt>Mais assistências</dt><dd>{{ stats.top_assists.player }} <span class="num">· {{ stats.top_assists.value }}</span></dd></div>
        <div v-if="stats.goalkeeper"><dt>Goleiro</dt><dd>{{ stats.goalkeeper.player }} <span class="num">· {{ formatMetric('save_pct', stats.goalkeeper.save_pct) }} de defesas · {{ stats.goalkeeper.clean_sheets }} sem sofrer gol</span></dd></div>
        <div v-if="stats.average_attendance"><dt>Público médio em casa</dt><dd class="num">{{ formatNumber(stats.average_attendance) }}<template v-if="stats.stadium"> · {{ stats.stadium }}</template></dd></div>
      </dl>
    </div>
    <p class="note">Totais da Série A {{ stats.as_of.slice(0, 4) }} numa cópia local; podem estar atrás dos resultados acima. Posição entre {{ stats.metrics[0]?.clubs ?? 20 }} clubes, 1º = melhor (em gols sofridos, finalizações cedidas, faltas e cartões, menos é melhor).</p>
  </section>
</template>

<style scoped>
.groups { display: grid; grid-template-columns: minmax(0, 1fr); gap: var(--s-5); }
@container (min-width: 760px) { .groups { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--s-5); } }
.group h3 { padding-bottom: var(--s-2); border-bottom: 1px solid var(--line); }
.metric { display: grid; grid-template-columns: minmax(0, 1fr) auto 32px; grid-template-areas: "label value rank" "league league rank"; column-gap: var(--s-3); align-items: center; padding: var(--s-2) 0; border-bottom: 1px solid var(--line); }
dt { grid-area: label; font-size: var(--fs-xs); color: var(--text-2); }
.value { grid-area: value; font-size: var(--fs-md); font-weight: 650; text-align: right; }
.league { grid-area: league; font-size: var(--fs-2xs); color: var(--text-3); }
.rank { grid-area: rank; justify-self: end; display: inline-grid; place-items: center; min-width: 30px; height: 22px; padding: 0 4px; border-radius: 5px; font-size: var(--fs-2xs); font-weight: 650; background: var(--surface-2); color: var(--text-2); }
.rank.good { background: var(--accent-soft); color: var(--accent); }
.rank.bad { background: var(--danger-bg); color: var(--danger); }

.extras { display: grid; grid-template-columns: minmax(0, 1fr); gap: var(--s-5); margin-top: var(--s-5); }
@container (min-width: 760px) { .extras { grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr); } }
.splits { display: grid; grid-template-columns: 1fr 1fr; gap: var(--s-3); }
.split { padding: var(--s-3) var(--s-4); background: var(--surface-1); border: 1px solid var(--line); border-radius: var(--r-md); display: grid; gap: 2px; }
.split-ppm { font-size: var(--fs-xs); color: var(--text-3); }
.split-ppm strong { font-size: var(--fs-xl); font-weight: 550; color: var(--text); letter-spacing: -.02em; margin-right: 2px; }
.split-line { font-size: var(--fs-xs); color: var(--text-2); }
.leaders { display: grid; gap: 0; align-content: start; }
.leaders div { display: grid; grid-template-columns: 150px minmax(0, 1fr); gap: var(--s-3); padding: var(--s-2) 0; border-bottom: 1px solid var(--line); font-size: var(--fs-sm); }
.leaders dt { grid-area: auto; color: var(--text-3); font-size: var(--fs-xs); }
.leaders dd { font-weight: 550; overflow-wrap: anywhere; }
.leaders dd span { font-weight: 400; color: var(--text-2); }
.note { font-size: var(--fs-2xs); color: var(--text-3); margin-top: var(--s-3); line-height: 1.6; }
@container (max-width: 480px) { .leaders div { grid-template-columns: 1fr; gap: 0; } }
</style>
