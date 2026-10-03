<script setup lang="ts">
defineProps<{ disabled: boolean }>()
defineEmits<{ ask: [question: string] }>()
const questions = [
  { label: 'Posição na tabela', text: 'Em que posição o clube está na tabela do Brasileirão e quantos pontos o separam do líder e da zona de rebaixamento?' },
  { label: 'Títulos e histórico', text: 'Quantos títulos brasileiros o clube tem no histórico disponível e qual é o seu retrospecto geral na Série A?' },
  { label: 'Confrontos', text: 'Contra quais adversários atuais o clube tem o melhor e o pior retrospecto na Série A?' },
  { label: 'Forma recente', text: 'Quantos jogos recentes o clube venceu, empatou e perdeu?' },
  { label: 'Últimos 3 jogos', text: 'Como foi o desempenho do clube nas três últimas partidas disponíveis?' },
  { label: 'Ataque', text: 'Analise o ataque com base nos gols marcados e, se disponíveis, nas finalizações. Separe fatos de interpretações.' },
  { label: 'Defesa', text: 'Analise a defesa com base nos gols sofridos e, se disponíveis, nas finalizações do adversário. Separe fatos de interpretações.' },
]
</script>
<template><div class="quick-questions"><button v-for="question in questions" :key="question.label" :disabled="disabled" @click="$emit('ask', question.text)">{{ question.label }} <span aria-hidden="true">↗</span></button></div></template>
