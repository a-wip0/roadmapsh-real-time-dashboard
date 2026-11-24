// Typed client for the Go backend. All paths go through the /api proxy prefix.

export interface Entry {
  rank: number
  username: string
  score: number
}

export interface SubmitResult {
  game: string
  score: number
  personal_best: number
  new_personal_best: boolean
  game_rank: number
  global_rank: number
}

export interface HistoryItem {
  game: string
  score: number
  created_at: string
}

export interface Report {
  from: string
  to: string
  game: string
  players: Entry[]
}

export type Period = 'day' | 'week' | 'month'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

const BASE = '/api'

async function request<T>(path: string, opts: { method?: string; token?: string | null; body?: unknown } = {}): Promise<T> {
  const headers: Record<string, string> = {}
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'
  if (opts.token) headers.Authorization = `Bearer ${opts.token}`
  const res = await fetch(BASE + path, {
    method: opts.method ?? 'GET',
    headers,
    body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
  })
  const data = await res.json().catch(() => null)
  if (!res.ok) throw new ApiError(res.status, data?.error ?? res.statusText)
  return data as T
}

export const api = {
  register: (username: string, password: string) =>
    request<{ username: string }>('/auth/register', { method: 'POST', body: { username, password } }),
  login: (username: string, password: string) =>
    request<{ token: string }>('/auth/login', { method: 'POST', body: { username, password } }),
  submitScore: (token: string, game: string, score: number) =>
    request<SubmitResult>('/scores', { method: 'POST', token, body: { game, score } }),
  history: (token: string, game?: string) =>
    request<HistoryItem[]>(`/scores/history${game ? `?game=${encodeURIComponent(game)}` : ''}`, { token }),
  leaderboard: (game?: string, limit = 10) =>
    request<Entry[]>(`/leaderboard${game ? `/${encodeURIComponent(game)}` : ''}?limit=${limit}`),
  myRank: (token: string, game?: string) =>
    request<Entry>(`/rankings/me${game ? `?game=${encodeURIComponent(game)}` : ''}`, { token }),
  report: (period: Period, game?: string) =>
    request<Report>(`/reports/top-players?period=${period}${game ? `&game=${encodeURIComponent(game)}` : ''}`),
}
