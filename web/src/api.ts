export const API = '/api/v1'

let token: string | null = localStorage.getItem('token')

export function getToken(): string | null {
  return token
}

export function setToken(t: string | null) {
  token = t
  if (t) localStorage.setItem('token', t)
  else localStorage.removeItem('token')
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers['Authorization'] = `Bearer ${token}`
  const resp = await fetch(API + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!resp.ok) {
    let msg = resp.statusText
    try {
      const data = await resp.json()
      if (data.error) msg = data.error
    } catch {
      /* ignore */
    }
    // A stored token the server rejects (expired or signed with a rotated
    // secret) is unrecoverable: drop it and return to the login screen.
    // Login/register calls carry no token, so they are unaffected.
    if (resp.status === 401 && token) {
      setToken(null)
      window.location.assign('/login')
    }
    throw new ApiError(resp.status, msg)
  }
  if (resp.status === 204) return undefined as T
  return (await resp.json()) as T
}

export const get = <T>(path: string) => request<T>('GET', path)
export const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body)
export const del = <T>(path: string) => request<T>('DELETE', path)

// ---------------------------------------------------------------- types

export interface Question {
  type: 'choose_one' | 'choose_n' | 'choose_player_order'
  prompt?: string
  choices: Choice[]
  n?: number
}

export interface Choice {
  id: string
  label: string
  kind: string
  cardCode?: string
  sourceId?: string
  disabled?: boolean
  then?: Question
}

export interface VillainView {
  id: string
  code: string
  name: string
  stage: number
  stageLabel: string
  hp: number
  maxHp: number
  scheme: number
  attack: number
  stunned: boolean
  confused: boolean
  tough: boolean
  boosts: number
}

export interface SchemeView {
  id: string
  code: string
  name: string
  threat: number
  maxThreat: number
  stage?: number
  crisis?: boolean
  hazard?: number
  acceleration?: number
}

export interface MinionView {
  id: string
  code: string
  name: string
  hp: number
  maxHp: number
  attack: number
  scheme: number
  guard: boolean
  stunned: boolean
  tough: boolean
}

export interface AllyView {
  id: string
  code: string
  name: string
  hp: number
  maxHp: number
  attack: number
  thwart: number
  exhausted: boolean
  stunned: boolean
}

export interface EntityLite {
  id: string
  code: string
  name: string
  exhausted: boolean
}

export interface CardRef {
  id: string
  code: string
  name: string
}

export interface PlayerView {
  id: string
  name: string
  userId?: string
  side: string
  heroCode: string
  alterEgo: string
  hp: number
  maxHp: number
  exhausted: boolean
  stunned: boolean
  confused: boolean
  tough: boolean
  firstPlayer: boolean
  koed: boolean
  formChanged: boolean
  hand?: CardRef[]
  handSize: number
  deckCount: number
  discardTop?: CardRef
  allies: AllyView[] | null
  supports: EntityLite[] | null
  upgrades: EntityLite[] | null
  encounterDown: number
}

export interface GameView {
  id: number
  name: string
  scenario: string
  round: number
  over: boolean
  won: boolean
  reason?: string
  villains: VillainView[] | null
  mainScheme: SchemeView | null
  sideSchemes: SchemeView[] | null
  minions: MinionView[] | null
  players: PlayerView[]
  log: string[] | null
  question?: Question
  waitingFor?: string
}

export interface Deck {
  id: number
  name: string
  investigatorCode: string
  slots: Record<string, number>
}

export interface CardInfo {
  code: string
  name: string
  subname?: string
  packCode: string
  packName?: string
  type: string
  category: string
  aspect?: string
  cost?: number | null
  unique: boolean
  traits?: string[]
  resources?: string[]
}

let cardsPromise: Promise<CardInfo[]> | null = null

// Full card catalog, fetched once per session and shared by consumers.
export function allCards(): Promise<CardInfo[]> {
  if (!cardsPromise) cardsPromise = get<CardInfo[]>('/marvel/cards')
  return cardsPromise
}

export interface ScenarioInfo {
  id: string
  name: string
}

export interface GameListItem {
  id: number
  name: string
  scenarioId: string
  status: string
  updatedAt: string
}
