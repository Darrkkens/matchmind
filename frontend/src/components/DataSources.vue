<script setup lang="ts">
import type { DataMetadata } from '../types/football'
import { formatDateTime, formatDay } from '../utils/format'

defineProps<{ notice: string; metadata?: DataMetadata }>()
const statsSources: Record<string, string> = { 'api-football': 'API-Football', 'api-futebol': 'API-Futebol', 'api-futebol-teste': 'API-Futebol (dados de exemplo)', almanacstats: 'AlmanacStats' }
</script>

<template>
  <section class="sources" aria-label="Origem dos dados">
    <div class="line">
      <span class="real"><span class="status-dot online" aria-hidden="true"></span>Dados reais</span>
      <template v-if="metadata">
        <a class="link" :href="metadata.source_url" target="_blank" rel="noopener noreferrer">{{ metadata.competition }} · JSON de origem<span class="sr-only"> (abre em nova aba)</span></a>
        <span>Obtido em {{ formatDateTime(metadata.fetched_at) }}</span>
        <span>Última partida: {{ metadata.latest_match_date ? formatDay(metadata.latest_match_date) : 'nenhuma' }}</span>
        <span v-if="metadata.statistics_source === 'almanacstats'">Estatísticas e elenco: <a class="link" href="https://almanacstats.com" target="_blank" rel="noopener noreferrer">AlmanacStats</a></span>
        <span v-else-if="metadata.statistics_source">Estatísticas: {{ statsSources[metadata.statistics_source] ?? metadata.statistics_source }}</span>
      </template>
    </div>
    <p v-if="metadata?.statistics_notice" class="warn">Estatísticas: {{ metadata.statistics_notice }}</p>
    <p v-if="metadata?.history_notice" class="warn">Histórico: {{ metadata.history_notice }}</p>
    <details class="disclosure about">
      <summary>Sobre a cobertura dos dados</summary>
      <p>{{ notice }}</p>
    </details>
  </section>
</template>

<style scoped>
.sources { font-size: var(--fs-xs); color: var(--text-3); }
.line { display: flex; flex-wrap: wrap; align-items: center; gap: var(--s-2) var(--s-4); }
.real { display: inline-flex; align-items: center; gap: var(--s-2); color: var(--accent); font-weight: 600; }
.warn { margin-top: var(--s-2); color: var(--warn); }
.about { margin-top: var(--s-3); }
.about > summary { font-size: var(--fs-xs); min-height: 40px; }
.about p { padding-bottom: var(--s-3); max-width: 72ch; color: var(--text-2); line-height: 1.7; }
</style>
