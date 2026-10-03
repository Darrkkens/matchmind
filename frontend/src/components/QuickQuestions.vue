<script setup lang="ts">
defineProps<{ disabled: boolean }>()
defineEmits<{ ask: [question: string] }>()
const questions = [
  { label: 'Posição na tabela', text: 'Em que posição o clube está na tabela do Brasileirão e quantos pontos o separam do líder e da zona de rebaixamento?' },
  { label: 'Títulos e histórico', text: 'Quantos títulos brasileiros o clube tem no histórico disponível e qual é o seu retrospecto geral na Série A?' },
  { label: 'Confrontos', text: 'Contra quais adversários atuais o clube tem o melhor e o pior retrospecto na Série A?' },
  { label: 'Próximo jogo', text: 'Qual é o próximo jogo do clube, contra qual adversário, quando e onde (em casa ou fora)?' },
  { label: 'Prévia do próximo jogo', text: 'Compare o clube e o próximo adversário nesta temporada: campanha em casa e fora, ataque e defesa. Separe fatos de interpretações e não preveja o resultado.' },
  { label: 'Forma recente', text: 'Quantos jogos recentes o clube venceu, empatou e perdeu?' },
  { label: 'Últimos 5 jogos', text: 'Como foi o desempenho do clube nas cinco últimas partidas disponíveis?' },
  { label: 'Ataque', text: 'Analise o ataque na temporada (gols, finalizações, precisão e posição na liga), se disponíveis, e nos jogos recentes. Separe fatos de interpretações.' },
  { label: 'Defesa', text: 'Analise a defesa na temporada (gols sofridos, finalizações cedidas, defesas do goleiro e jogos sem sofrer gol), se disponíveis, e nos jogos recentes. Separe fatos de interpretações.' },
]
</script>

<template>
  <ul class="quick-questions" aria-label="Perguntas rápidas">
    <li v-for="question in questions" :key="question.label"><button type="button" class="btn btn-chip" :disabled="disabled" :title="question.text" @click="$emit('ask', question.text)">{{ question.label }}</button></li>
  </ul>
</template>

<style scoped>
/* One scrollable row keeps the conversation tall; fades hint that more chips exist. */
.quick-questions { display: flex; gap: var(--s-2); overflow-x: auto; scrollbar-width: none; padding: 2px; margin: -2px; scroll-snap-type: x proximity; overscroll-behavior-x: contain; mask-image: linear-gradient(90deg, transparent, #000 12px, #000 calc(100% - 24px), transparent); padding-left: 12px; margin-left: -12px; }
.quick-questions::-webkit-scrollbar { display: none; }
li { scroll-snap-align: start; flex-shrink: 0; }
</style>
