<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Snapshot } from '../types/football'
import { useInView } from '../composables/useInView'
import TeamHeader from '../components/TeamHeader.vue'
import DataSources from '../components/DataSources.vue'
import RecentForm from '../components/RecentForm.vue'
import NextMatch from '../components/NextMatch.vue'
import SeasonStats from '../components/SeasonStats.vue'
import RecentMatches from '../components/RecentMatches.vue'
import StandingsTable from '../components/StandingsTable.vue'
import HistoryPanel from '../components/HistoryPanel.vue'
import TrophyList from '../components/TrophyList.vue'
import SquadList from '../components/SquadList.vue'
import ChatPanel from '../components/ChatPanel.vue'

const props = defineProps<{ data: Snapshot | null; loading: boolean; error: string; online: boolean | null }>()
defineEmits<{ search: [input: string]; 'dismiss-error': [] }>()

const position = computed(() => props.data?.standings?.find((row) => row.team_id === props.data!.team.id)?.position)
const aside = ref<HTMLElement>()
const chat = ref<InstanceType<typeof ChatPanel>>()
const chatVisible = useInView(aside)

// The simulation's "explain" hands a question to the chat and brings it into view.
function askChat(question: string) {
  if (!chatVisible.value) goToChat()
  void chat.value?.ask(question)
}

// On single-column layouts the chat sits below the data; this shortcut jumps straight to it.
function goToChat() {
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  aside.value?.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'start' })
  chat.value?.focus()
}
</script>

<template>
  <div class="dashboard">
    <div class="main">
      <div v-if="error" class="banner banner-error" role="alert">{{ error }}<button type="button" class="btn btn-secondary" @click="$emit('dismiss-error')">Fechar</button></div>

      <div v-if="loading" class="loading" role="status" aria-label="Carregando dados do clube">
        <div class="sk-header"><div class="skeleton sk-badge"></div><div class="sk-lines"><div class="skeleton"></div><div class="skeleton"></div></div></div>
        <div class="skeleton sk-strip"></div>
        <div class="sk-rows"><div v-for="i in 5" :key="i" class="skeleton"></div></div>
        <p class="muted">Lendo os dados do clube…</p>
      </div>

      <div v-else-if="data" :key="data.team.id" class="content">
        <TeamHeader :team="data.team" :position="position" :competition="data.data_metadata?.competition" />
        <DataSources :notice="data.data_notice" :metadata="data.data_metadata" />
        <NextMatch :fixture="data.next_match" :team-id="data.team.id" :season="data.season_stats" :opponent-season="data.next_opponent_season" :recent="data.recent_matches" :opponent-recent="data.next_opponent_recent" :availability="data.next_match_availability" @explain="askChat" />
        <RecentForm :form="data.recent_form" />
        <RecentMatches :matches="data.recent_matches" :team-id="data.team.id" />
        <SeasonStats v-if="data.season_stats" :stats="data.season_stats" />
        <StandingsTable v-if="data.standings?.length" :standings="data.standings" title="Tabela do campeonato" :subtitle="data.data_metadata?.competition" :highlight-id="data.team.id" :disabled="loading" @select="$emit('search', $event)" />
        <div class="split">
          <HistoryPanel v-if="data.history" :history="data.history" :team-name="data.team.name" />
          <div class="stack">
            <TrophyList :trophies="data.trophies" :covered="!!data.history" />
            <SquadList :players="data.squad" />
          </div>
        </div>
      </div>
    </div>

    <aside v-if="data && !loading" ref="aside" class="aside" aria-label="Assistente">
      <ChatPanel ref="chat" :key="data.team.id" :team-id="data.team.id" :team-name="data.team.name" :online="online" />
    </aside>

    <button v-if="data && !loading" type="button" :class="['ask-shortcut btn btn-primary', { hidden: chatVisible }]" :tabindex="chatVisible ? -1 : 0" :aria-hidden="chatVisible" @click="goToChat">
      <span aria-hidden="true">✳</span> Perguntar<span class="sr-only"> ao MatchMind</span>
    </button>
  </div>
</template>

<style scoped>
.dashboard { display: grid; grid-template-columns: minmax(0, 1fr) 400px; gap: var(--s-6); align-items: start; padding-top: var(--s-6); }
.main { min-width: 0; display: grid; gap: var(--s-5); container-type: inline-size; }
.content { display: grid; gap: var(--s-7); min-width: 0; animation: rise var(--dur-slow) var(--ease-out); }
.content > :first-child { margin-bottom: calc(var(--s-7) * -1 + var(--s-4)); }
.split { display: grid; gap: var(--s-7); min-width: 0; }
.stack { display: grid; gap: var(--s-7); align-content: start; min-width: 0; }
@container (min-width: 760px) { .split { grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr); gap: var(--s-6); } }

.aside { position: sticky; top: var(--s-4); height: calc(100dvh - var(--s-4) * 2); min-height: 520px; }

.ask-shortcut { display: none; }

.loading { display: grid; gap: var(--s-5); }
.sk-header { display: flex; gap: var(--s-5); align-items: center; }
.sk-badge { width: 72px; height: 80px; }
.sk-lines { flex: 1; display: grid; gap: var(--s-3); }
.sk-lines .skeleton:first-child { height: 14px; width: 30%; border-radius: var(--r-sm); }
.sk-lines .skeleton:last-child { height: 40px; width: 60%; border-radius: var(--r-sm); }
.sk-strip { height: 104px; }
.sk-rows { display: grid; gap: var(--s-2); }
.sk-rows .skeleton { height: 52px; border-radius: var(--r-sm); }

@media (max-width: 1200px) {
  .dashboard { grid-template-columns: minmax(0, 1fr); gap: var(--s-7); padding-bottom: var(--s-8); }
  .aside { position: static; height: min(720px, calc(100dvh - var(--s-6))); min-height: 480px; scroll-margin-top: var(--s-4); }
  .ask-shortcut {
    display: inline-flex; position: fixed; z-index: var(--z-sticky);
    right: max(var(--s-4), env(safe-area-inset-right)); bottom: calc(var(--s-4) + env(safe-area-inset-bottom));
    min-height: 40px; padding: 0 var(--s-4); gap: 6px; font-size: var(--fs-xs); border-radius: var(--r-full); box-shadow: 0 6px 20px -6px #000b;
    transition: transform var(--dur) var(--ease-out), opacity var(--dur-fast) ease, background-color var(--dur-fast) ease;
  }
  .ask-shortcut.hidden { opacity: 0; transform: translateY(16px); pointer-events: none; }
}
@media (max-width: 600px) {
  .dashboard { padding-top: var(--s-5); }
  .content, .split, .stack { gap: var(--s-6); }
  .content > :first-child { margin-bottom: calc(var(--s-6) * -1 + var(--s-4)); }
  .aside { height: calc(100dvh - var(--s-4) * 2); }
}
@media (prefers-reduced-motion: reduce) { .ask-shortcut.hidden { transform: none; } }
</style>
