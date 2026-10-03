<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref } from 'vue'
import { api } from '../services/api'
import QuickQuestions from './QuickQuestions.vue'
const props = defineProps<{ teamId: string; teamName: string; online: boolean | null }>()
interface Message { role: 'user' | 'assistant'; text: string; sources?: string[] }
const messages = ref<Message[]>([])
const question = ref('')
const busy = ref(false)
const error = ref('')
const lastQuestion = ref('')
const conversation = ref<HTMLElement>()
const sourceLabels: Record<string, string> = { team: 'clube', recent_matches: 'partidas recentes', recent_form: 'forma recente', standings: 'tabela', history: 'histórico', squad: 'elenco', trophies: 'títulos' }
let controller: AbortController | undefined
let disposed = false
onBeforeUnmount(() => { disposed = true; controller?.abort() })
async function scroll() { await nextTick(); conversation.value?.scrollTo({ top: conversation.value.scrollHeight, behavior: 'smooth' }) }
async function ask(text: string, retry = false) {
  text = text.trim()
  if (!text || busy.value) return
  if (!retry) messages.value.push({ role: 'user', text })
  lastQuestion.value = text; question.value = ''; busy.value = true; error.value = ''
  controller = new AbortController()
  const timer = window.setTimeout(() => controller?.abort('timeout'), 260_000)
  void scroll()
  try {
    const answer = await api.chat(props.teamId, text, controller.signal)
    if (!disposed) messages.value.push({ role: 'assistant', text: answer.answer, sources: answer.sources_used })
  } catch (cause) {
    if (!disposed) error.value = controller.signal.aborted ? 'A requisição demorou demais. Tente novamente ou use um modelo menor.' : cause instanceof Error ? cause.message : 'Não foi possível responder. Tente novamente.'
  } finally { window.clearTimeout(timer); busy.value = false; if (!disposed) void scroll() }
}
</script>
<template>
  <section id="ask-matchmind" class="panel chat-panel">
    <div class="chat-heading"><div class="ai-mark" aria-hidden="true">✳</div><div><h2>Pergunte ao MatchMind</h2><p>Seu clube. Suas perguntas. Um pouco mais de análise.</p></div><span class="badge local">IA LOCAL</span></div>
    <p class="chat-offline" v-if="online === false">A IA está offline ou o modelo não está instalado. Rode <code>ollama serve</code> and <code>ollama pull gemma3:4b</code>, e tente novamente.</p>
    <QuickQuestions :disabled="busy" @ask="ask" />
    <div ref="conversation" class="conversation" role="log" aria-live="polite" aria-label="Conversa com o MatchMind" :aria-busy="busy">
      <div v-if="!messages.length" class="chat-welcome"><span class="welcome-orbit" aria-hidden="true">✳</span><h3>Toda partida tem uma história.</h3><p>Pergunte qualquer coisa sobre o {{ teamName }}.<br />Vou usar os dados disponíveis do clube e a tabela.</p><div class="grounding-note"><span class="live-dot"></span> Fatos primeiro. Interpretações claramente identificadas.</div></div>
      <article v-for="(message, i) in messages" :key="i" :class="['message', message.role]"><span class="message-label">{{ message.role === 'user' ? 'VOCÊ' : '✳ MATCHMIND' }}</span><p>{{ message.text }}</p><div v-if="message.sources?.length" class="sources"><span>Contexto citado</span><span v-for="source in message.sources" :key="source">{{ sourceLabels[source] ?? source }}</span></div></article>
      <div v-if="busy" class="thinking" role="status"><span></span><span></span><span></span> Lendo o jogo… a primeira resposta pode demorar um pouco.</div>
      <div v-if="error" class="error chat-error" role="alert">{{ error }} <button @click="ask(lastQuestion, true)">Tentar de novo ↗</button></div>
    </div>
    <form class="chat-form" @submit.prevent="ask(question)"><label class="sr-only" for="question">Faça uma pergunta sobre este clube</label><textarea id="question" v-model="question" rows="2" maxlength="1000" :disabled="busy" placeholder="Pergunte qualquer coisa sobre este clube…" @keydown.enter.exact.prevent="ask(question)"></textarea><button class="primary" :disabled="busy || !question.trim()" aria-label="Enviar pergunta">Enviar <span aria-hidden="true">↑</span></button></form>
    <div class="chat-footnote"><span>Com Ollama + Gemma · Sem IA na nuvem</span><span>{{ question.length }}/1000</span></div>
  </section>
</template>
