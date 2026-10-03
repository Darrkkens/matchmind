<script setup lang="ts">
import type { Trophy } from '../types/football'
defineProps<{ trophies: Trophy[]; covered?: boolean }>()
</script>

<template>
  <section aria-labelledby="trophies-title">
    <div class="section-head"><h2 id="trophies-title">Sala de troféus</h2><span class="muted">Neste conjunto de dados</span></div>
    <ul v-if="trophies.length">
      <li v-for="trophy in trophies" :key="trophy.competition">
        <div><h3>{{ trophy.competition }}</h3><p class="num">{{ trophy.seasons?.join(' · ') || 'Temporadas indisponíveis' }}</p></div>
        <strong class="num"><span aria-hidden="true">×</span>{{ trophy.count }}<span class="sr-only"> {{ trophy.count === 1 ? 'título' : 'títulos' }}</span></strong>
      </li>
    </ul>
    <p v-else class="empty">{{ covered ? 'Nenhum título brasileiro no período coberto; outras competições não estão disponíveis.' : 'Títulos indisponíveis nesta fonte de dados.' }}</p>
  </section>
</template>

<style scoped>
li { display: flex; align-items: center; gap: var(--s-4); padding: var(--s-3) 0; border-top: 1px solid var(--line); }
li:first-child { border-top: 0; padding-top: 0; }
h3 { font-size: var(--fs-sm); font-weight: 600; }
p { font-size: var(--fs-xs); color: var(--text-3); margin-top: 2px; }
strong { margin-left: auto; font-size: var(--fs-xl); font-weight: 500; color: var(--warn); letter-spacing: -.02em; }
strong > span[aria-hidden] { font-size: .6em; margin-right: 2px; color: var(--text-3); }
</style>
