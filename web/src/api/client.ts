import type { ErrorResponse, Hole, ScoreRecord } from './types'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`/api${path}`)
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ErrorResponse | null
    throw new Error(body?.error ?? `request to ${path} failed with ${res.status}`)
  }
  return res.json() as Promise<T>
}

// The default limit covers the full archive of daily holes in one request —
// see maxHolesLimit in internal/api/holes.go.
export function listHoles(limit = 1000): Promise<Hole[]> {
  return get<Hole[]>(`/holes?limit=${limit}`)
}

export function topScores(hole: number, limit = 25): Promise<ScoreRecord[]> {
  return get<ScoreRecord[]>(`/holes/${hole}/top?limit=${limit}`)
}

/** Each matching user's best play per hole, holes descending. */
export function searchScores(username: string, limit = 50): Promise<ScoreRecord[]> {
  return get<ScoreRecord[]>(`/scores?username=${encodeURIComponent(username)}&limit=${limit}`)
}
