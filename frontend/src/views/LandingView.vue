<script lang="ts">
// The hero entrance plays on the first visit only, not every time the landing remounts (e.g. after a failed search).
let introPlayed = false
</script>

<script setup lang="ts">
import type { LeagueTable } from '../types/football'
import { formatDateTime } from '../utils/format'
import TeamSearch from '../components/TeamSearch.vue'
import StandingsTable from '../components/StandingsTable.vue'

defineProps<{ loading: boolean; error: string; table: LeagueTable | null; tableError: string; tableLoading: boolean }>()
defineEmits<{ search: [input: string]; 'retry-table': []; 'dismiss-error': [] }>()
const query = defineModel<string>('query', { default: '' })
const intro = !introPlayed
introPlayed = true
const features = [
  { tag: '01 / Os dados', title: 'Uma visão mais clara do seu clube.', text: 'Resultados reais do Brasileirão a partir da OpenFootball. A cobertura e os campos indisponíveis ficam sempre visíveis.' },
  { tag: '02 / A análise', title: 'Fatos antes de opiniões.', text: 'Forma e tabela calculadas a partir dos placares. Interpretações da IA baseadas apenas nos dados disponíveis.' },
  { tag: '03 / A inteligência', title: 'Jogando em casa.', text: 'O Gemma, de pesos abertos, roda via Ollama na sua máquina. Suas perguntas ficam com você.' },
]
</script>

<template>
  <div class="landing">
    <section :class="['hero', { intro }]" aria-labelledby="hero-title">
      <p class="eyebrow kicker"><span class="status-dot online" aria-hidden="true"></span>Inteligência local · Futebol brasileiro</p>
      <h1 id="hero-title">Menos ruído.<br />Mais <span>futebol.</span></h1>
      <p class="lede">Seu analista de futebol brasileiro com IA local. Conheça seu clube além do placar, explore os números e faça perguntas melhores.</p>
      <div class="search-wrap">
        <TeamSearch v-model="query" :loading="loading" @search="$emit('search', $event)" />
        <div v-if="error" class="banner banner-error" role="alert">{{ error }}<button type="button" class="btn btn-secondary" @click="$emit('dismiss-error')">Fechar</button></div>
      </div>
      <p class="note">Sem conta. Sem IA na nuvem. Só o seu clube e um pouco de curiosidade.</p>
    </section>

    <div class="table-wrap">
      <StandingsTable v-if="table" :standings="table.standings" :title="`Tabela · ${table.competition}`" :subtitle="`Atualizada em ${formatDateTime(table.fetched_at)} · escolha um clube para analisar`" :disabled="loading" @select="$emit('search', $event)" />
      <div v-else-if="tableError" class="banner banner-error" role="alert">{{ tableError }}<button type="button" class="btn btn-secondary" :disabled="tableLoading" @click="$emit('retry-table')">{{ tableLoading ? 'Tentando…' : 'Tentar de novo' }}</button></div>
      <div v-else class="table-skeleton" role="status" aria-label="Carregando a tabela"><div v-for="i in 8" :key="i" class="skeleton"></div></div>
    </div>

    <ul class="features">
      <li v-for="f in features" :key="f.tag"><p class="eyebrow tag">{{ f.tag }}</p><h2>{{ f.title }}</h2><p>{{ f.text }}</p></li>
    </ul>
  </div>
</template>

<style scoped>
.landing { overflow-x: clip; }
.hero { position: relative; max-width: 760px; margin: 0 auto; padding: var(--s-8) 0 var(--s-7); text-align: center; isolation: isolate; }
/* A single static glow behind the headline: identity, not decoration-in-motion. */
.hero::before { content: ''; position: absolute; z-index: -1; inset: 0 -20% auto; height: 420px; background: radial-gradient(50% 55% at 50% 40%, #c4f66b14, transparent 70%); pointer-events: none; }
.kicker { display: inline-flex; align-items: center; gap: var(--s-2); color: var(--text-2); }
h1 { font-size: clamp(44px, 7.5vw, 88px); line-height: 1.02; letter-spacing: -.055em; font-weight: 650; margin: var(--s-5) 0; }
h1 span { color: var(--accent); }
.lede { font-size: var(--fs-md); line-height: 1.6; color: var(--text-2); max-width: 52ch; margin: 0 auto var(--s-6); }
.search-wrap { display: grid; gap: var(--s-3); text-align: left; }
.note { margin-top: var(--s-5); font-size: var(--fs-xs); color: var(--text-3); }

/* First-visit entrance: staggered rise, once. */
.intro > * { animation: rise 520ms var(--ease-out) both; }
.intro > :nth-child(2) { animation-delay: 60ms; }
.intro > :nth-child(3) { animation-delay: 120ms; }
.intro > :nth-child(4) { animation-delay: 180ms; }
.intro > :nth-child(5) { animation-delay: 240ms; }

.table-wrap { max-width: 880px; margin: 0 auto var(--s-8); }
.table-skeleton { display: grid; gap: var(--s-2); }
.table-skeleton .skeleton { height: 40px; border-radius: var(--r-sm); }

.features { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--s-6); padding: var(--s-6) 0 var(--s-7); border-top: 1px solid var(--line); }
.tag { color: var(--accent); }
.features h2 { font-size: var(--fs-md); font-weight: 600; margin: var(--s-3) 0 var(--s-2); }
.features li > p:last-child { font-size: var(--fs-sm); line-height: 1.7; color: var(--text-2); max-width: 40ch; }
@media (max-width: 900px) { .features { grid-template-columns: 1fr; gap: var(--s-5); } }
@media (max-width: 600px) {
  .hero { padding: var(--s-7) 0 var(--s-6); }
  h1 { letter-spacing: -.045em; }
  .lede { font-size: var(--fs-base); }
}
</style>
