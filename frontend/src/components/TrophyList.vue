<script setup lang="ts">
import type { Trophy } from '../types/football'
defineProps<{ trophies: Trophy[]; covered?: boolean }>()
</script>
<template>
  <section class="panel trophy-panel"><div class="section-title"><h2>Sala de troféus</h2><span class="eyebrow">NESTE CONJUNTO DE DADOS</span></div><div v-for="trophy in trophies" :key="trophy.competition" class="trophy-row"><div class="trophy-icon" aria-hidden="true">♜</div><div><h3>{{ trophy.competition }}</h3><p>{{ trophy.seasons?.join(' · ') || 'Temporadas indisponíveis' }}</p></div><strong>×{{ trophy.count }}</strong></div><p v-if="!trophies.length" class="empty">{{ covered ? 'Nenhum título brasileiro no período coberto; outras competições não estão disponíveis.' : 'Títulos indisponíveis nesta fonte de dados.' }}</p></section>
</template>
