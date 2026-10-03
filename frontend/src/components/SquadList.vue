<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Player } from '../types/football'

const props = defineProps<{ players: Player[] }>()
const PREVIEW = 12
const order = ['Goleiro', 'Defensor', 'Meio-campo', 'Atacante']
const showAll = ref(false)
// Most-used players first, then by position; numbers are the totals the source reports.
const sorted = computed(() => [...props.players].sort((a, b) => (b.appearances ?? 0) - (a.appearances ?? 0) || order.indexOf(a.position) - order.indexOf(b.position)))
const visible = computed(() => showAll.value ? sorted.value : sorted.value.slice(0, PREVIEW))
const hasStats = computed(() => props.players.some((p) => p.appearances))
</script>

<template>
  <section aria-labelledby="squad-title">
    <div class="section-head"><h2 id="squad-title">Elenco</h2><span class="muted">{{ players.length }} jogadores disponíveis</span></div>
    <template v-if="players.length">
      <div v-if="hasStats" class="columns eyebrow" aria-hidden="true"><span>Jogador</span><span>Jogos</span></div>
      <ul id="squad-list" class="list">
        <li v-for="player in visible" :key="player.id">
          <span v-if="!hasStats" class="shirt num" title="Número">{{ player.number }}</span>
          <div class="who">
            <h3>{{ player.name }}</h3>
            <p class="num">{{ player.position }}<template v-if="hasStats"> · {{ player.goals ?? 0 }} G · {{ player.assists ?? 0 }} A<template v-if="player.rating"> · nota {{ player.rating.toFixed(2) }}</template></template></p>
          </div>
          <span v-if="hasStats" class="apps num"><span class="sr-only">Jogos: </span>{{ player.appearances ?? 0 }}</span>
        </li>
      </ul>
      <button v-if="players.length > PREVIEW" class="btn btn-secondary more" type="button" aria-controls="squad-list" :aria-expanded="showAll" @click="showAll = !showAll">{{ showAll ? 'Mostrar menos' : `Mostrar todos (${players.length})` }}</button>
      <p v-if="hasStats" class="note">G gols · A assistências · totais informados pela <a class="link" href="https://almanacstats.com" target="_blank" rel="noopener noreferrer">AlmanacStats</a></p>
    </template>
    <p v-else class="empty">Elenco indisponível nesta fonte de dados.</p>
  </section>
</template>

<style scoped>
.columns { display: flex; justify-content: space-between; padding-bottom: var(--s-2); }
.list li { display: flex; align-items: center; gap: var(--s-3); min-height: 48px; padding: 6px 0; border-top: 1px solid var(--line); min-width: 0; animation: rise var(--dur) var(--ease-out) both; }
.shirt { width: 32px; height: 32px; display: grid; place-items: center; border-radius: var(--r-sm); background: var(--surface-2); color: var(--text-2); font-size: var(--fs-xs); flex-shrink: 0; }
.who { min-width: 0; }
h3 { font-size: var(--fs-sm); font-weight: 550; }
p { font-size: var(--fs-xs); color: var(--text-3); }
.apps { margin-left: auto; font-size: var(--fs-md); font-weight: 600; color: var(--text); }
.more { margin-top: var(--s-3); }
.note { font-size: var(--fs-2xs); margin-top: var(--s-3); }
</style>
