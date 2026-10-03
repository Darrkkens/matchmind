import type { Snapshot, ChatAnswer, Health, LeagueTable, MatchLineups, Simulation } from '../types/football'

const base = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')
async function request<T>(path: string, body?: object, signal?: AbortSignal): Promise<T> {
  try {
    const res = await fetch(`${base}/api${path}`, {
      method: body ? 'POST' : 'GET',
      headers: body ? { 'Content-Type': 'application/json' } : undefined,
      body: body ? JSON.stringify(body) : undefined,
      signal: signal ?? AbortSignal.timeout(260_000),
    })
    const payload = await res.json()
    if (!res.ok) throw new Error(payload.error || `A requisição falhou (${res.status}).`)
    return payload as T
  } catch (error) {
    if (error instanceof TypeError) throw new Error('Não foi possível conectar ao MatchMind. Verifique se a API Go está rodando na porta 8080.')
    if (error instanceof DOMException && error.name === 'TimeoutError') throw new Error('A requisição demorou demais. Tente novamente ou use um modelo menor no Ollama.')
    throw error
  }
}
export const api = {
  resolve: (input: string) => request<Snapshot>('/team/resolve', { input }),
  chat: (team_id: string, question: string, signal: AbortSignal) => request<ChatAnswer>('/chat', { team_id, question }, signal),
  lineups: (ref: string) => request<MatchLineups>(`/lineups/${encodeURIComponent(ref)}`, undefined, AbortSignal.timeout(20_000)),
  standings: () => request<LeagueTable>('/standings', undefined, AbortSignal.timeout(15_000)),
  // The AI step runs on the local CPU model, so this can take as long as a chat answer.
  simulate: (team_id: string, runs: number) => request<Simulation>('/simulate', { team_id, runs, use_ai: true }, AbortSignal.timeout(260_000)),
  health: () => request<Health>('/health', undefined, AbortSignal.timeout(5000)),
}
