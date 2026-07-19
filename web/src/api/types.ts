export interface Hole {
  hole: number
  custom: boolean
  top_strokes: number | null
}

export interface ScoreRecord {
  share_link: string
  hole: number
  strokes: number
  user_id: string
  username: string
  guild_id: string
  channel_id: string
  message_id: string
  recorded_at: string
}

export interface ErrorResponse {
  error: string
}
