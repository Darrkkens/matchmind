<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref } from 'vue'
import { api } from '../services/api'
import QuickQuestions from './QuickQuestions.vue'

const props = defineProps<{ teamId: string; teamName: string; online: boolean | null }>()
interface Message { id: number; role: 'user' | 'assistant'; text: string; sources?: string[] }
const MAX_LENGTH = 1000
const TIMEOUT_MS = 260_000
const messages = ref<Message[]>([])
const question = ref('')
const busy = ref(false)
const error = ref('')
const lastQuestion = ref('')
const conversation = ref<HTMLElement>()
const input = ref<HTMLTextAreaElement>()
const sourceLabels: Record<string, string> = { team: 'clube', recent_matches: 'partidas recentes', recent_form: 'forma recente', next_match: 'próximo jogo', season_stats: 'temporada', simulation: 'simulação', standings: 'tabela', history: 'histórico', squad: 'elenco', trophies: 'títulos' }
let controller: AbortController | undefined
let disposed = false
let nextId = 0
onBeforeUnmount(() => { disposed = true; controller?.abort() })

async function scroll() {
  await nextTick()
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  conversation.value?.scrollTo({ top: conversation.value.scrollHeight, behavior: reduce ? 'auto' : 'smooth' })
}
async function ask(text: string, retry = false) {
  text = text.trim()
  if (!text || busy.value) return
  if (!retry) messages.value.push({ id: nextId++, role: 'user', text })
  lastQuestion.value = text; question.value = ''; busy.value = true; error.value = ''
  controller = new AbortController()
  const timer = window.setTimeout(() => controller?.abort('timeout'), TIMEOUT_MS)
  void scroll()
  try {
    const answer = await api.chat(props.teamId, text, controller.signal)
    if (!disposed) messages.value.push({ id: nextId++, role: 'assistant', text: answer.answer, sources: answer.sources_used })
  } catch (cause) {
    if (!disposed) error.value = controller.signal.aborted ? 'A requisição demorou demais. Tente novamente ou use um modelo menor.' : cause instanceof Error ? cause.message : 'Não foi possível responder. Tente novamente.'
  } finally { window.clearTimeout(timer); busy.value = false; if (!disposed) void scroll() }
}
const focus = () => input.value?.focus({ preventScroll: true })
defineExpose({ focus, ask })
</script>

<template>
  <section id="ask-matchmind" class="chat" aria-labelledby="chat-title">
    <header>
      <span class="mark" aria-hidden="true">✳</span>
      <div><h2 id="chat-title">Pergunte ao MatchMind</h2><p>Análise do {{ teamName }} com IA local</p></div>
      <span :class="['state', online ? 'on' : online === false ? 'off' : '']"><span :class="['status-dot', online ? 'online' : online === null ? 'checking' : '']" aria-hidden="true"></span>{{ online ? 'Gemma local' : online === false ? 'IA offline' : 'Verificando' }}</span>
    </header>

    <p v-if="online === false" class="banner banner-warn offline">A IA está offline ou o modelo não está instalado. Rode <code>ollama serve</code> e <code>ollama pull gemma3:4b</code>, e tente novamente.</p>

    <div ref="conversation" class="conversation" role="log" aria-label="Conversa com o MatchMind" :aria-busy="busy">
      <div v-if="!messages.length" class="welcome">
        <h3>Toda partida tem uma história.</h3>
        <p>Pergunte qualquer coisa sobre o {{ teamName }}. Vou usar só os dados disponíveis do clube e a tabela.</p>
        <p class="grounding">Fatos primeiro · interpretações claramente identificadas</p>
      </div>
      <article v-for="message in messages" :key="message.id" :class="['message', message.role]">
        <h3 class="label">{{ message.role === 'user' ? 'Você' : 'MatchMind' }}</h3>
        <p>{{ message.text }}</p>
        <div v-if="message.sources?.length" class="sources"><span>Contexto citado</span><ul><li v-for="source in message.sources" :key="source">{{ sourceLabels[source] ?? source }}</li></ul></div>
      </article>
      <div v-if="busy" class="thinking" role="status"><span class="dots" aria-hidden="true"><i></i><i></i><i></i></span>Lendo o jogo… a primeira resposta pode demorar um pouco.</div>
      <div v-if="error" class="banner banner-error" role="alert">{{ error }}<button type="button" class="btn btn-secondary" @click="ask(lastQuestion, true)">Tentar de novo</button></div>
    </div>

    <div class="composer">
      <QuickQuestions :disabled="busy" @ask="ask" />
      <form class="field" @submit.prevent="ask(question)">
        <label class="sr-only" for="question">Faça uma pergunta sobre este clube</label>
        <textarea id="question" ref="input" v-model="question" rows="2" :maxlength="MAX_LENGTH" :disabled="busy" enterkeyhint="send" placeholder="Pergunte qualquer coisa sobre este clube…" aria-describedby="chat-hint" @keydown.enter.exact.prevent="ask(question)"></textarea>
        <button class="btn btn-primary send" :disabled="busy || !question.trim()" aria-label="Enviar pergunta">
          <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true"><path d="M8 13V3m0 0L3.5 7.5M8 3l4.5 4.5" stroke="currentColor" stroke-width="1.8" fill="none" stroke-linecap="round" stroke-linejoin="round" /></svg>
        </button>
      </form>
      <p id="chat-hint" class="hint"><span>Enter envia · Shift+Enter quebra linha · sem IA na nuvem</span><span class="num" :class="{ near: question.length > MAX_LENGTH * .9 }">{{ question.length }}/{{ MAX_LENGTH }}</span></p>
    </div>
  </section>
</template>

<style scoped>
.chat { display: flex; flex-direction: column; min-height: 0; height: 100%; background: var(--surface-1); border: 1px solid var(--line); border-radius: var(--r-lg); overflow: hidden; }
header { display: flex; align-items: center; gap: var(--s-3); padding: var(--s-4) var(--s-5); border-bottom: 1px solid var(--line); }
.mark { font-size: 24px; color: var(--accent); line-height: 1; }
h2 { font-size: var(--fs-md); font-weight: 600; }
header p { font-size: var(--fs-xs); color: var(--text-3); }
.state { margin-left: auto; display: inline-flex; align-items: center; gap: 6px; font-size: var(--fs-2xs); color: var(--text-3); white-space: nowrap; }
.state.on { color: var(--accent); }
.state.off { color: var(--warn); }
.offline { margin: var(--s-4) var(--s-5) 0; display: block; font-size: var(--fs-xs); line-height: 1.6; }

.conversation { flex: 1; min-height: 240px; overflow-y: auto; overscroll-behavior: contain; padding: var(--s-4) var(--s-5); scrollbar-width: thin; scrollbar-color: var(--line-strong) transparent; display: flex; flex-direction: column; gap: var(--s-3); }
.welcome { margin: auto 0; padding: var(--s-6) var(--s-2); text-align: center; }
.welcome h3 { font-size: var(--fs-md); font-weight: 550; }
.welcome p { font-size: var(--fs-sm); color: var(--text-2); margin-top: var(--s-2); max-width: 36ch; margin-inline: auto; }
.welcome .grounding { font-size: var(--fs-2xs); color: var(--text-3); margin-top: var(--s-4); }

.message { padding: var(--s-3) var(--s-4); border-radius: var(--r-md); background: var(--surface-2); max-width: 100%; animation: rise var(--dur) var(--ease-out); }
.message.user { align-self: flex-end; max-width: 88%; background: var(--surface-3); border-bottom-right-radius: 4px; }
.message.assistant { border-bottom-left-radius: 4px; }
.label { font-size: var(--fs-2xs); font-weight: 600; letter-spacing: var(--tracking-label); text-transform: uppercase; color: var(--text-3); }
.assistant .label { color: var(--accent); }
.message p { font-size: var(--fs-sm); line-height: 1.7; white-space: pre-wrap; overflow-wrap: anywhere; margin-top: var(--s-1); }
.sources { display: flex; flex-wrap: wrap; align-items: center; gap: var(--s-2); margin-top: var(--s-3); font-size: var(--fs-2xs); color: var(--text-3); }
.sources ul { display: flex; flex-wrap: wrap; gap: var(--s-1); }
.sources li { background: var(--surface-3); color: var(--text-2); border-radius: var(--r-full); padding: 2px var(--s-2); }

.thinking { display: flex; align-items: center; gap: var(--s-2); font-size: var(--fs-xs); color: var(--text-3); animation: fade var(--dur) ease; }
.dots { display: inline-flex; gap: 3px; }
.dots i { width: 5px; height: 5px; border-radius: 50%; background: var(--accent); animation: pulse 1.2s ease-in-out infinite; }
.dots i:nth-child(2) { animation-delay: .15s; }
.dots i:nth-child(3) { animation-delay: .3s; }
@keyframes pulse { 0%, 100% { opacity: .25; } 50% { opacity: 1; } }

.composer { display: grid; gap: var(--s-3); padding: var(--s-3) var(--s-5) var(--s-4); border-top: 1px solid var(--line); }
.field { display: flex; align-items: flex-end; gap: var(--s-2); padding: 6px; background: var(--surface-2); border: 1px solid var(--line-strong); border-radius: var(--r-md); transition: border-color var(--dur-fast) ease, box-shadow var(--dur-fast) ease; }
.field:focus-within { border-color: var(--accent-line); box-shadow: 0 0 0 3px var(--accent-soft); }
textarea { flex: 1; resize: none; field-sizing: content; min-height: 44px; max-height: 160px; padding: 10px var(--s-2); border: 0; background: transparent; color: var(--text); font-size: var(--fs-sm); line-height: 1.5; }
textarea:focus { outline: none; }
.send { width: 40px; height: 40px; min-height: 40px; padding: 0; flex-shrink: 0; }
.hint { display: flex; justify-content: space-between; gap: var(--s-3); font-size: var(--fs-2xs); color: var(--text-3); }
.near { color: var(--warn); }
/* iOS zooms into fields under 16px. */
@media (pointer: coarse) { textarea { font-size: var(--fs-md); } .hint > span:first-child { visibility: hidden; } }
@media (max-width: 600px) {
  header, .conversation { padding-inline: var(--s-4); }
  .composer { padding-inline: var(--s-4); }
  .offline { margin-inline: var(--s-4); }
}
</style>
