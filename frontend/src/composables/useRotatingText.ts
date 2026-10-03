import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

// Cycles through phrases while `active` is true and counts elapsed seconds.
// Restarts from the first phrase on every activation; timers stop when inactive or unmounted.
export function useRotatingText(phrases: readonly string[], active: Ref<boolean>, intervalMs = 4000) {
  const index = ref(0)
  const elapsed = ref(0)
  let phraseTimer: number | undefined
  let clockTimer: number | undefined

  const stop = () => { window.clearInterval(phraseTimer); window.clearInterval(clockTimer); phraseTimer = clockTimer = undefined }
  watch(active, (on) => {
    stop()
    if (!on) return
    index.value = 0
    elapsed.value = 0
    phraseTimer = window.setInterval(() => { index.value = (index.value + 1) % phrases.length }, intervalMs)
    clockTimer = window.setInterval(() => { elapsed.value++ }, 1000)
  }, { immediate: true })
  onBeforeUnmount(stop)

  return { index, elapsed }
}
