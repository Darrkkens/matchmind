import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

// True while any part of the element is on screen; the observer follows the element across re-renders.
export function useInView(target: Ref<HTMLElement | undefined>, threshold = 0.15) {
  const visible = ref(false)
  const observer = new IntersectionObserver(([entry]) => { visible.value = entry.isIntersecting }, { threshold })
  watch(target, (el, previous) => {
    if (previous) observer.unobserve(previous)
    if (el) observer.observe(el)
    else visible.value = false
  }, { immediate: true, flush: 'post' })
  onBeforeUnmount(() => observer.disconnect())
  return visible
}
