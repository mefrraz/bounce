const DEFAULT_URL = 'http://localhost:3001'

export interface Game {
  id: string
  data: string
  hora: string
  equipa_casa: string
  equipa_fora: string
  resultado_casa: number | null
  resultado_fora: number | null
  local?: string
  competicao?: string
  estado: string
  logo_casa?: string
  logo_fora?: string
}

export interface Standing {
  posicao: number
  equipa: string
  j: number; v: number; d: number
  pm?: number; ps?: number; dif?: number
  pts: number
  logo?: string
}

export interface WSEvent {
  type: 'score_update' | 'game_started' | 'game_finished'
  data: Game
}

export class BounceClient {
  private baseUrl: string
  constructor(baseUrl = DEFAULT_URL) { this.baseUrl = baseUrl.replace(/\/$/, '') }

  async health() { const r = await fetch(`${this.baseUrl}/health`); return r.json() }
  async games(p: { date?: string; competition?: string } = {}) {
    const qs = new URLSearchParams()
    if (p.date) qs.set('date', p.date)
    if (p.competition) qs.set('competition', p.competition)
    const r = await fetch(`${this.baseUrl}/api/games?${qs}`)
    if (!r.ok) throw new Error(`Bounce error: ${r.status}`)
    return r.json() as Promise<Game[]>
  }
  async standings(id: string) {
    const r = await fetch(`${this.baseUrl}/api/standings/${id}`)
    if (!r.ok) throw new Error(`Bounce error: ${r.status}`)
    return r.json() as Promise<Standing[]>
  }
  async game(id: string) {
    const r = await fetch(`${this.baseUrl}/api/game/${id}`)
    if (!r.ok) throw new Error(`Bounce error: ${r.status}`)
    return r.json() as Promise<Game>
  }
  watchGame(gameID: string, onEvent: (e: WSEvent) => void): () => void {
    const wsUrl = this.baseUrl.replace(/^http/, 'ws')
    const ws = new WebSocket(`${wsUrl}/ws/game/${gameID}`)
    ws.onmessage = (msg) => { try { onEvent(JSON.parse(msg.data)) } catch {} }
    return () => ws.close()
  }
}

export default BounceClient
