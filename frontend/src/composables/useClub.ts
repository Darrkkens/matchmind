import { ref } from 'vue'
import { api } from '../services/api'
import type { LeagueTable, Snapshot } from '../types/football'
import { errorMessage } from '../utils/format'

export function useClub() {
  const data = ref<Snapshot | null>(null)
  const loading = ref(false)
  const error = ref('')
  const query = ref('')

  async function resolve(input: string) {
    loading.value = true; error.value = ''; query.value = input
    try { data.value = await api.resolve(input) } catch (cause) { error.value = errorMessage(cause, 'Não foi possível carregar o clube.') } finally { loading.value = false }
  }

  return { data, loading, error, query, resolve }
}

export function useStandings() {
  const table = ref<LeagueTable | null>(null)
  const error = ref('')
  const loading = ref(false)

  async function load() {
    loading.value = true; error.value = ''
    try { table.value = await api.standings() } catch (cause) { error.value = errorMessage(cause, 'Não foi possível carregar a tabela.') } finally { loading.value = false }
  }

  return { table, error, loading, load }
}
