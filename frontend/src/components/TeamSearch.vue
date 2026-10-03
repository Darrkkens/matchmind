<script setup lang="ts">
defineProps<{ loading: boolean; compact?: boolean }>()
const emit = defineEmits<{ search: [input: string] }>()
const input = defineModel<string>({ default: '' })
function submit() { if (input.value.trim()) emit('search', input.value.trim()) }
function example(name: string) { input.value = name; submit() }
</script>

<template>
  <div :class="['search-block', { compact }]">
    <form class="team-search" @submit.prevent="submit">
      <span class="search-symbol" aria-hidden="true">⌕</span>
      <label class="sr-only" for="team-input">Nome do clube ou URL do time</label>
      <input id="team-input" v-model="input" maxlength="2048" required autocomplete="off" placeholder="Digite o nome de um clube ou cole a URL do time" :disabled="loading" />
      <button class="primary" :disabled="loading || !input.trim()">{{ loading ? 'Analisando…' : 'Analisar' }} <span aria-hidden="true">↗</span></button>
    </form>
    <div class="examples"><span>EXPERIMENTE</span><button v-for="name in ['Palmeiras', 'Flamengo', 'Corinthians', 'Mirassol']" :key="name" :disabled="loading" @click="example(name)">{{ name }} <span aria-hidden="true">↗</span></button></div>
  </div>
</template>
