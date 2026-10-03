<script setup lang="ts">
import { computed, onMounted } from 'vue'
import TeamSearch from './components/TeamSearch.vue'
import LandingView from './views/LandingView.vue'
import ClubDashboard from './views/ClubDashboard.vue'
import { useHealth } from './composables/useHealth'
import { useClub, useStandings } from './composables/useClub'

const health = useHealth()
const club = useClub()
const standings = useStandings()
const { data, loading, error, query } = club
onMounted(() => { void standings.load() })

const onDashboard = computed(() => !!data.value || loading.value)
async function search(input: string) {
  await club.resolve(input)
  void health.refresh()
}
</script>

<template>
  <a class="skip-link" href="#main">Pular para o conteúdo</a>
  <div class="shell">
    <header :class="['topbar', { searching: onDashboard }]">
      <a class="wordmark" href="/" aria-label="MatchMind, início"><img src="/favicon.svg" width="30" height="30" alt="" /><span>MATCH<span class="light">MIND</span></span></a>
      <p v-if="!onDashboard" class="tagline"><span>Brasileirão</span> · Futebol brasileiro, em perspectiva.</p>
      <div v-else class="topbar-search"><TeamSearch v-model="query" :loading="loading" compact @search="search" /></div>
      <button type="button" class="connection" :aria-busy="health.checking.value" title="Verificar conexão novamente" @click="health.refresh()">
        <span :class="['status-dot', health.status.value]" aria-hidden="true"></span>
        <span aria-live="polite">{{ health.label.value }}</span>
      </button>
    </header>

    <main id="main" tabindex="-1">
      <ClubDashboard v-if="onDashboard" :data="data" :loading="loading" :error="error" :online="health.health.value?.ollama ?? null" @search="search" @dismiss-error="error = ''" />
      <LandingView v-else v-model:query="query" :loading="loading" :error="error" :table="standings.table.value" :table-error="standings.error.value" :table-loading="standings.loading.value" @search="search" @retry-table="standings.load()" @dismiss-error="error = ''" />
    </main>

    <footer>
      <p><b>MATCHMIND</b> — Feito por amor ao futebol.</p>
      <ul>
        <li>Código aberto</li>
        <li>Local primeiro</li>
        <li>Dados: OpenFootball · <a class="link" href="https://almanacstats.com" target="_blank" rel="noopener noreferrer">AlmanacStats</a></li>
        <li>Hacktoberfest 2026</li>
        <li><a class="link" href="/crests/credits.html" target="_blank" rel="noopener">Créditos dos escudos</a></li>
      </ul>
    </footer>
  </div>
</template>

<style scoped>
.shell { max-width: 1440px; margin: 0 auto; padding: 0 max(var(--gutter), env(safe-area-inset-right)) 0 max(var(--gutter), env(safe-area-inset-left)); min-height: 100dvh; display: flex; flex-direction: column; }
.topbar { display: flex; align-items: center; gap: var(--s-6); min-height: 72px; padding-top: env(safe-area-inset-top); border-bottom: 1px solid var(--line); }
.wordmark { display: flex; align-items: center; gap: 10px; font-size: 17px; font-weight: 800; letter-spacing: -.04em; white-space: nowrap; border-radius: var(--r-sm); }
.wordmark img { border-radius: 7px; }
.light { font-weight: 400; }
.tagline { font-size: var(--fs-xs); color: var(--text-3); }
.tagline span { color: var(--text-2); }
.topbar-search { flex: 1; max-width: 560px; }
.connection { margin-left: auto; display: inline-flex; align-items: center; gap: var(--s-2); min-height: 32px; padding: 0 var(--s-3); border: 1px solid var(--line); border-radius: var(--r-full); font-size: var(--fs-xs); color: var(--text-2); white-space: nowrap; transition: border-color var(--dur-fast) ease, transform var(--dur-press) var(--ease-out); }
.connection:active { transform: scale(.97); }
@media (hover: hover) and (pointer: fine) { .connection:hover { border-color: var(--line-strong); color: var(--text); } }
main { flex: 1; outline: none; }
footer { display: flex; justify-content: space-between; flex-wrap: wrap; gap: var(--s-3) var(--s-5); padding: var(--s-6) 0 calc(var(--s-6) + env(safe-area-inset-bottom)); margin-top: var(--s-7); border-top: 1px solid var(--line); font-size: var(--fs-xs); color: var(--text-3); }
footer b { color: var(--text-2); letter-spacing: .04em; }
footer ul { display: flex; flex-wrap: wrap; gap: var(--s-1) var(--s-4); }
@media (max-width: 900px) {
  .tagline { display: none; }
  .topbar { gap: var(--s-4); }
  .topbar.searching { flex-wrap: wrap; padding-bottom: var(--s-3); row-gap: var(--s-3); }
  .topbar.searching .wordmark, .topbar.searching .connection { margin-top: var(--s-3); }
  .topbar.searching .topbar-search { order: 3; flex-basis: 100%; max-width: none; }
}
@media (max-width: 600px) {
  .topbar { min-height: 60px; }
  .wordmark { font-size: 15px; }
  .wordmark img { width: 26px; height: 26px; }
  .connection { font-size: var(--fs-2xs); }
  footer { flex-direction: column; }
}
</style>
