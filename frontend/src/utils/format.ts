import type { Goal, Match } from '../types/football'

export type Outcome = 'W' | 'D' | 'L'

// Dates from the API are calendar days in UTC; formatting them in local time would shift them a day back in Brazil.
const dayFormat = new Intl.DateTimeFormat('pt-BR', { timeZone: 'UTC' })
const longDayFormat = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC' })
const weekdayFormat = new Intl.DateTimeFormat('pt-BR', { weekday: 'short', day: 'numeric', month: 'short', timeZone: 'UTC' })
const dateTimeFormat = new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short', timeStyle: 'short' })

export const formatDay = (value: string) => dayFormat.format(new Date(value))
export const formatLongDay = (value: string) => longDayFormat.format(new Date(value))
export const formatWeekday = (value: string) => weekdayFormat.format(new Date(value))
export const formatDateTime = (value: string) => dateTimeFormat.format(new Date(value))

export const outcomeLetter: Record<string, string> = { W: 'V', D: 'E', L: 'D' }
export const outcomeLabel: Record<string, string> = { W: 'Vitória', D: 'Empate', L: 'Derrota' }

export function outcome(match: Match, teamId: string): Outcome {
  const difference = (match.home_score - match.away_score) * (match.home_team.id === teamId ? 1 : -1)
  return difference > 0 ? 'W' : difference < 0 ? 'L' : 'D'
}

export const goalNote = (goal: Goal) => goal.kind === 'own_goal' ? ' (contra)' : goal.kind === 'penalty' ? ' (pên.)' : ''

export const scorerList = (goals: Goal[] | undefined, side: Goal['side']) =>
  (goals ?? []).filter((g) => g.side === side).map((g) => `${g.player} ${g.minute}'${goalNote(g)}`).join(', ')

// Share of available points: 3 per win, 1 per draw, one decimal place.
export const pointsPercentage = (r: { played: number; wins: number; draws: number }) =>
  r.played ? Math.round((r.wins * 3 + r.draws) / (r.played * 3) * 1000) / 10 : 0

export const initials = (name: string) => name.split(/\s+/).map((w) => w[0]).join('').slice(0, 3).toUpperCase()

export const errorMessage = (cause: unknown, fallback: string) => cause instanceof Error ? cause.message : fallback

// OpenFootball rounds come as "Matchday 28".
export const roundLabel = (round?: string) => round ? round.replace(/^matchday\s*/i, 'Rodada ') : ''

// Whole days from today (local calendar) to a YYYY-MM-DD date.
export function daysUntil(date: string, now = new Date()) {
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())
  return Math.round((new Date(date).getTime() - today) / 86_400_000)
}
export function relativeDay(date: string) {
  const days = daysUntil(date)
  return days <= 0 ? 'Hoje' : days === 1 ? 'Amanhã' : `Em ${days} dias`
}
