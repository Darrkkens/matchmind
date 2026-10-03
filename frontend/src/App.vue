<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import TeamSearch from './components/TeamSearch.vue'
import TeamHeader from './components/TeamHeader.vue'
import RecentForm from './components/RecentForm.vue'
import RecentMatches from './components/RecentMatches.vue'
import TrophyList from './components/TrophyList.vue'
import SquadList from './components/SquadList.vue'
import ChatPanel from './components/ChatPanel.vue'
import StandingsTable from './components/StandingsTable.vue'
import HistoryPanel from './components/HistoryPanel.vue'
import { api } from './services/api'
import type { Snapshot, Health, LeagueTable } from './types/football'
const data = ref<Snapshot | null>(null)
const health = ref<Health | null>(null)
const table = ref<LeagueTable | null>(null)
const tableError = ref('')
const loading = ref(false)
const error = ref('')
const apiOffline = ref(false)
const searchInput = ref('')
let polling: number
async function refreshHealth() {
  try { health.value = await api.health(); apiOffline.value = false } catch { health.value = null; apiOffline.value = true }
}
async function loadTable() {
  tableError.value = ''
  try { table.value = await api.standings() } catch (cause) { tableError.value = cause instanceof Error ? cause.message : 'Não foi possível carregar a tabela.' }
}
onMounted(() => { void refreshHealth(); void loadTable(); polling = window.setInterval(() => refreshHealth(), 30_000) })
onBeforeUnmount(() => window.clearInterval(polling))
async function resolve(input: string) {
  loading.value = true; error.value = ''; searchInput.value = input
  try { data.value = await api.resolve(input); void refreshHealth() } catch (cause) { error.value = cause instanceof Error ? cause.message : 'Não foi possível carregar o clube.' } finally { loading.value = false }
}
const statsSources: Record<string, string> = { 'api-football': 'API-Football', 'api-futebol': 'API-Futebol', 'api-futebol-teste': 'API-Futebol (dados de exemplo)', almanacstats: 'AlmanacStats' }
const dateTime = (value: string) => new Date(value).toLocaleString('pt-BR')
const date = (value: string) => new Intl.DateTimeFormat('pt-BR', { timeZone: 'UTC' }).format(new Date(value))
</script>

<template>
  <a class="skip-link" href="#main">Pular para o conteúdo</a>
  <div class="app-shell">
    <header class="topbar"><a class="wordmark" href="/" aria-label="Início do MatchMind"><img src="/favicon.svg" width="34" height="34" alt="" /><span>MATCH<span class="wordmark-light">MIND</span><sup>↗</sup></span></a><div class="topbar-center"><span class="active-nav">Brasileirão</span><span class="nav-divider">/</span><span>Futebol brasileiro, em perspectiva.</span></div><button class="connection" @click="refreshHealth()" title="Verificar conexão"><span :class="['status-dot', { connected: health?.ollama }]"></span>{{ apiOffline ? 'API offline' : health?.ollama ? 'IA local conectada' : health ? 'IA local offline' : 'Verificando conexão' }}</button></header>
    <main id="main">
      <template v-if="!data && !loading">
        <section class="hero"><div class="eyebrow hero-eyebrow"><span class="live-dot"></span> INTELIGÊNCIA LOCAL. FUTEBOL BRASILEIRO.</div><h1>Menos ruído.<br />Mais <span>futebol.</span></h1><p class="hero-subtitle">Seu analista de futebol brasileiro com IA local.</p><p class="hero-description">Conheça seu clube além do placar.<br />Explore os números. Faça perguntas melhores. Encontre a história.</p><TeamSearch v-model="searchInput" :loading="loading" @search="resolve" /><div class="hero-note"><span>↳</span> Sem conta. Sem IA na nuvem. Só o seu clube e um pouco de curiosidade.</div></section>
        <StandingsTable v-if="table" class="landing-standings" :standings="table.standings" :title="`Tabela · ${table.competition}`" :subtitle="`Atualizada em ${dateTime(table.fetched_at)} · clique em um clube para analisar`" :disabled="loading" @select="resolve" />
        <div v-else-if="tableError" class="error resolve-error" role="alert">{{ tableError }}<button @click="loadTable">Tentar de novo ↗</button></div>
        <div class="landing-grid"><div class="landing-feature"><span>01 / OS DADOS</span><h2>Uma visão mais clara do seu clube.</h2><p>Resultados reais do Brasileirão a partir da OpenFootball. A cobertura e os campos indisponíveis ficam sempre visíveis.</p></div><div class="landing-feature"><span>02 / A ANÁLISE</span><h2>Fatos antes de opiniões.</h2><p>Forma e tabela calculadas a partir dos placares. Interpretações da IA baseadas apenas nos dados disponíveis.</p></div><div class="landing-feature"><span>03 / A INTELIGÊNCIA</span><h2>Jogando em casa.</h2><p>O Gemma, de pesos abertos, roda via Ollama na sua máquina. Suas perguntas ficam com você.</p></div></div>
      </template>
      <TeamSearch v-model="searchInput" v-if="data" :loading="loading" compact @search="resolve" />
      <div v-if="error" class="error resolve-error" role="alert">{{ error }}<button aria-label="Fechar erro" @click="error = ''">×</button></div>
      <div v-if="loading" class="dashboard-skeleton" role="status" aria-label="Carregando dados do clube"><div class="skeleton skeleton-header"></div><div class="skeleton skeleton-form"></div><div class="skeleton-cards"><div v-for="i in 3" :key="i" class="skeleton"></div></div><span class="muted">Lendo os dados do clube…</span></div>
      <div v-show="!loading" v-if="data" class="dashboard">
        <TeamHeader :team="data.team" :position="data.standings?.find((row) => row.team_id === data!.team.id)?.position" />
        <div class="data-notice real-notice"><span class="badge real">Dados reais</span><p>{{ data.data_notice }}</p></div>
        <div v-if="data.data_metadata" class="data-provenance">
          <a :href="data.data_metadata.source_url" target="_blank" rel="noopener noreferrer">{{ data.data_metadata.competition }} · JSON de origem ↗</a>
          <span>Obtido em {{ dateTime(data.data_metadata.fetched_at) }}</span>
          <span v-if="data.data_metadata.statistics_source === 'almanacstats'">Estatísticas e elenco: <a class="credit" href="https://almanacstats.com" target="_blank" rel="noopener noreferrer">AlmanacStats ↗</a></span>
          <span v-else-if="data.data_metadata.statistics_source">Estatísticas: {{ statsSources[data.data_metadata.statistics_source] ?? data.data_metadata.statistics_source }}</span>
          <span v-if="data.data_metadata.statistics_notice" class="stats-notice">Estatísticas: {{ data.data_metadata.statistics_notice }}</span>
          <span v-if="data.data_metadata.history_notice" class="stats-notice">Histórico: {{ data.data_metadata.history_notice }}</span>
          <span>Última partida disponível: {{ data.data_metadata.latest_match_date ? date(data.data_metadata.latest_match_date) : 'Nenhuma' }}</span>
        </div>
        <RecentForm :form="data.recent_form" />
        <RecentMatches :matches="data.recent_matches" :team-id="data.team.id" />
        <div class="dashboard-columns"><div class="club-column"><StandingsTable v-if="data.standings?.length" :standings="data.standings" title="Tabela do campeonato" :subtitle="data.data_metadata?.competition" :highlight-id="data.team.id" :disabled="loading" @select="resolve" /><section class="panel club-panel"><div class="section-title"><h2>Por dentro do clube</h2><span aria-hidden="true">↗</span></div><dl><div><dt>Técnico</dt><dd>{{ data.team.coach || 'Indisponível' }}</dd></div><div><dt>Fundação</dt><dd>{{ data.team.founded_year || 'Indisponível' }}</dd></div><div><dt>Estádio</dt><dd>{{ data.team.stadium || 'Indisponível' }}</dd></div></dl></section><HistoryPanel v-if="data.history" :history="data.history" :team-name="data.team.name" /><TrophyList :trophies="data.trophies" :covered="!!data.history" /><SquadList :players="data.squad" /></div><ChatPanel :key="data.team.id" :team-id="data.team.id" :team-name="data.team.name" :online="health?.ollama ?? null" /></div>
      </div>
    </main>
    <footer><span><b>MATCHMIND</b> <span class="footer-dash">—</span> Feito por amor ao futebol.</span><span>Código aberto <span class="footer-dot">·</span> Local primeiro <span class="footer-dot">·</span> Dados: OpenFootball · <a href="https://almanacstats.com" target="_blank" rel="noopener noreferrer">AlmanacStats</a> <span class="footer-dot">·</span> Hacktoberfest 2026 <span class="footer-dot">·</span> <a href="/crests/credits.html" target="_blank" rel="noopener">Créditos dos escudos ↗</a></span></footer>
  </div>
</template>
