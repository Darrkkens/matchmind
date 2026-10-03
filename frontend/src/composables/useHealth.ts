import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '../services/api'
import type { Health } from '../types/football'

const POLL_MS = 30_000

// Polls the API health while the tab is visible; a hidden tab has nobody to show the status to.
export function useHealth() {
  const health = ref<Health | null>(null)
  const apiOffline = ref(false)
  const checking = ref(false)
  let timer: number | undefined

  async function refresh() {
    checking.value = true
    try { health.value = await api.health(); apiOffline.value = false } catch { health.value = null; apiOffline.value = true } finally { checking.value = false }
  }
  const start = () => { stop(); timer = window.setInterval(refresh, POLL_MS) }
  const stop = () => { window.clearInterval(timer); timer = undefined }
  function onVisibility() {
    if (document.hidden) return stop()
    void refresh(); start()
  }

  onMounted(() => { void refresh(); start(); document.addEventListener('visibilitychange', onVisibility) })
  onBeforeUnmount(() => { stop(); document.removeEventListener('visibilitychange', onVisibility) })

  const status = computed<'checking' | 'online' | 'offline'>(() => health.value?.ollama ? 'online' : health.value || apiOffline.value ? 'offline' : 'checking')
  const label = computed(() => apiOffline.value ? 'API offline' : health.value?.ollama ? 'IA local conectada' : health.value ? 'IA local offline' : 'Verificando conexão')

  return { health, apiOffline, checking, status, label, refresh }
}
