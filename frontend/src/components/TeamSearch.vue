<script setup lang="ts">
defineProps<{ loading: boolean; compact?: boolean }>()
const emit = defineEmits<{ search: [input: string] }>()
const input = defineModel<string>({ default: '' })
const examples = ['Palmeiras', 'Flamengo', 'Corinthians', 'Mirassol']
function submit() { if (input.value.trim()) emit('search', input.value.trim()) }
function example(name: string) { input.value = name; submit() }
</script>

<template>
  <div :class="['search', { compact }]">
    <form class="search-field" role="search" @submit.prevent="submit">
      <svg class="search-icon" viewBox="0 0 20 20" aria-hidden="true"><circle cx="8.5" cy="8.5" r="5.75" fill="none" stroke="currentColor" stroke-width="1.6" /><path d="m13 13 4 4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" /></svg>
      <label class="sr-only" for="team-input">Nome do clube ou URL do time</label>
      <input id="team-input" v-model="input" type="search" maxlength="2048" required autocomplete="off" autocapitalize="words" enterkeyhint="search" :placeholder="compact ? 'Buscar outro clube' : 'Digite o nome de um clube ou cole a URL do time'" :disabled="loading" />
      <button :class="['btn btn-primary', { 'btn-lg': !compact }]" :disabled="loading || !input.trim()" :aria-busy="loading">
        <span v-if="loading" class="spinner" aria-hidden="true"></span>
        {{ loading ? 'Analisando' : 'Analisar' }}
      </button>
    </form>
    <div v-if="!compact" class="examples">
      <span class="eyebrow" id="examples-label">Experimente</span>
      <ul aria-labelledby="examples-label">
        <li v-for="name in examples" :key="name"><button type="button" class="btn btn-chip" :disabled="loading" @click="example(name)">{{ name }}</button></li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.search-field { display: flex; align-items: center; gap: var(--s-3); padding: 6px 6px 6px var(--s-4); background: var(--surface-2); border: 1px solid var(--line-strong); border-radius: var(--r-md); transition: border-color var(--dur-fast) ease, box-shadow var(--dur-fast) ease; }
.search-field:focus-within { border-color: var(--accent-line); box-shadow: 0 0 0 3px var(--accent-soft); }
.search-icon { width: 18px; height: 18px; color: var(--text-3); flex-shrink: 0; }
input { flex: 1; min-height: 40px; border: 0; background: transparent; color: var(--text); font-size: var(--fs-md); }
input:focus { outline: none; }
input::-webkit-search-cancel-button { display: none; }
.compact .search-field { padding: 4px 4px 4px var(--s-3); border-color: var(--line); background: var(--surface-1); }
.compact input { min-height: 32px; font-size: var(--fs-base); }
.spinner { width: 14px; height: 14px; border-radius: 50%; border: 2px solid currentColor; border-right-color: transparent; animation: spin .6s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.examples { display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: var(--s-2) var(--s-3); margin-top: var(--s-4); }
.examples ul { display: flex; flex-wrap: wrap; justify-content: center; gap: var(--s-2); }
/* iOS zooms into inputs under 16px; keep the compact field at 16px on touch devices. */
@media (pointer: coarse) { .compact input { font-size: var(--fs-md); } }
@media (max-width: 600px) {
  .search-field { padding-left: var(--s-3); gap: var(--s-2); }
  .search-icon { display: none; }
}
</style>
