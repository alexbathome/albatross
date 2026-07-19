import type { ErrorResponse, Hole, ScoreRecord } from './types'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`/api${path}`)
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ErrorResponse | null
    throw new Error(body?.error ?? `request to ${path} failed with ${res.status}`)
  }
  return res.json() as Promise<T>
}

export function listHoles(limit = 20): Promise<Hole[]> {
  return get<Hole[]>(`/holes?limit=${limit}`)
}

export function topScores(hole: number, limit = 25): Promise<ScoreRecord[]> {
  return get<ScoreRecord[]>(`/holes/${hole}/top?limit=${limit}`)
}
