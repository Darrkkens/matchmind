<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Player } from '../types/football'
const props = defineProps<{ players: Player[] }>()
const order = ['Goleiro', 'Defensor', 'Meio-campo', 'Atacante']
const showAll = ref(false)
// Most-used players first, then by position; numbers are the totals the source reports.
const sorted = computed(() => [...props.players].sort((a, b) => (b.appearances ?? 0) - (a.appearances ?? 0) || order.indexOf(a.position) - order.indexOf(b.position)))
const visible = computed(() => showAll.value ? sorted.value : sorted.value.slice(0, 12))
const hasStats = computed(() => props.players.some((p) => p.appearances))
</script>
<template>
  <section class="panel squad-panel">
    <div class="section-title"><h2>Elenco</h2><span class="muted">{{ players.length }} jogadores disponíveis</span></div>
    <div class="squad-grid">
      <div v-for="player in visible" :key="player.id" class="player-row">
        <span class="shirt-number" :title="hasStats ? 'Jogos' : 'Número'">{{ hasStats ? player.appearances ?? 0 : player.number }}</span>
        <div><h3>{{ player.name }}</h3><p>{{ player.position }}<template v-if="hasStats"> · {{ player.goals ?? 0 }} G · {{ player.assists ?? 0 }} A<template v-if="player.rating"> · nota {{ player.rating.toFixed(2) }}</template></template></p></div>
        <span class="player-dot" aria-hidden="true"></span>
      </div>
    </div>
    <button v-if="players.length > 12" class="squad-more" type="button" @click="showAll = !showAll">{{ showAll ? 'Mostrar menos' : `Mostrar todos (${players.length})` }}</button>
    <p v-if="!players.length" class="empty">Elenco indisponível nesta fonte de dados.</p>
    <p v-else-if="hasStats" class="squad-note">Quadrado = jogos · G gols · A assistências · totais informados pela <a href="https://almanacstats.com" target="_blank" rel="noopener noreferrer">AlmanacStats ↗</a></p>
  </section>
</template>
