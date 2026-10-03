import type { SeasonMetric, SeasonStats, SplitRecord } from '../types/football'

// Labels and units for the metric keys sent by the backend (see internal/football/fbref.go).
export const metricInfo: Record<string, { label: string; short: string; unit?: '%'; integer?: boolean }> = {
  goals: { label: 'Gols por jogo', short: 'Gols/jogo' },
  shots: { label: 'Finalizações por jogo', short: 'Finalizações/jogo' },
  shots_on_target: { label: 'No alvo por jogo', short: 'No alvo/jogo' },
  shot_accuracy: { label: 'Precisão das finalizações', short: 'Precisão', unit: '%' },
  goals_per_shot: { label: 'Gols por finalização', short: 'Gols/finalização' },
  possession: { label: 'Posse de bola média', short: 'Posse', unit: '%' },
  goals_against: { label: 'Gols sofridos por jogo', short: 'Sofridos/jogo' },
  shots_against: { label: 'Finalizações cedidas por jogo', short: 'Cedidas/jogo' },
  shots_on_target_against: { label: 'No alvo cedidas por jogo', short: 'No alvo cedidas' },
  save_pct: { label: 'Defesas do goleiro', short: 'Defesas', unit: '%' },
  clean_sheets: { label: 'Jogos sem sofrer gol', short: 'Sem sofrer gol', integer: true },
  fouls: { label: 'Faltas por jogo', short: 'Faltas/jogo' },
  yellow_cards: { label: 'Cartões amarelos', short: 'Amarelos', integer: true },
  red_cards: { label: 'Cartões vermelhos', short: 'Vermelhos', integer: true },
}

export const groups = [
  { id: 'attack', label: 'Ataque' },
  { id: 'defense', label: 'Defesa' },
  { id: 'discipline', label: 'Disciplina' },
] as const

const decimal = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 2 })
const whole = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 0 })
export function formatMetric(key: string, value: number) {
  const info = metricInfo[key]
  return `${info?.integer ? whole.format(value) : decimal.format(value)}${info?.unit ?? ''}`
}
export const formatNumber = (value: number) => whole.format(value)

// Rank tiers: top quarter is good, bottom quarter is bad, the rest neutral.
export function rankTier(m: Pick<SeasonMetric, 'rank' | 'clubs'>) {
  const quarter = Math.max(1, Math.round(m.clubs / 4))
  return m.rank <= quarter ? 'good' : m.rank > m.clubs - quarter ? 'bad' : ''
}

export const metric = (stats: SeasonStats | undefined, key: string) => stats?.metrics.find((m) => m.key === key)

export const recordLine = (r: SplitRecord) => `${r.wins}V ${r.draws}E ${r.losses}D`
