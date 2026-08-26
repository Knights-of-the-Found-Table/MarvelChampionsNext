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

// Structured, locale-neutral message argument (see internal/engine/i18n.go):
// k "card" resolves to the localized card name by code, "i" is an int,
// "msg" nests another localizable message, "" is plain text.
export interface ArgWire {
  k?: string
  s?: string
  i?: number
  code?: string
  msg?: MsgWire
}

// A localizable message: key + structured args (client renders in the
// viewer's language) plus text (canonical English fallback).
export interface MsgWire {
  key?: string
  args?: ArgWire[]
  text: string
}

export interface Question {
  type: 'choose_one' | 'choose_n' | 'choose_player_order'
  prompt?: string
  promptKey?: string
  promptArgs?: ArgWire[]
  choices: Choice[]
  n?: number
  // Structured icon requirements for payment questions (mirror of the
  // server's abilityIcons spec); drives the "x/y paid" tracker.
  payIcons?: Array<{ icon: string; n: number }>
}

export interface Choice {
  id: string
  // Structured label in current builds; a bare string in old saves.
  label: MsgWire | string
  kind: string
  cardCode?: string
  sourceId?: string
  disabled?: boolean
  then?: Question
  // Resource icons this choice contributes toward a payment (server data;
  // never parsed back out of the rendered label).
  icons?: string[]
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
  confused?: boolean
  tough?: boolean
  engagedWith?: string
  faceDown?: boolean
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
  confused?: boolean
  tough?: boolean
  counters?: number
}

export interface EntityLite {
  id: string
  code: string
  name: string
  exhausted: boolean
  counters?: number
  attachTo?: string
}

// Card attached to (or associated with) a host entity.
export interface AttachmentView {
  id: string
  code: string
  name: string
  host?: string
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
  alterEgoCode: string
  heroName?: string
  alterEgoName?: string
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
  discardCount?: number
  discardTop?: CardRef
  allies: AllyView[] | null
  supports: EntityLite[] | null
  upgrades: EntityLite[] | null
  encounterDown: number
}

// One line of the in-game event journal: level is minor | info | major; seq
// is the server's monotonic entry number (absent on legacy saves) used to
// diff snapshots.
export interface LogEntry {
  level: string
  text: string
  key?: string
  args?: ArgWire[]
  seq?: number
}

// Public table talk; separate from LogEntry and never part of game state.
export interface ChatMessage {
  id: number
  at: number
  userId: string
  name: string
  text: string
  spectator: boolean
}

export interface GameView {
  id: number
  name: string
  scenario: string
  round: number
  over: boolean
  won: boolean
  reason?: MsgWire | string
  villains: VillainView[] | null
  mainScheme: SchemeView | null
  sideSchemes: SchemeView[] | null
  minions: MinionView[] | null
  attachments?: AttachmentView[] | null
  treacheries?: AttachmentView[] | null
  environments?: EntityLite[] | null
  players: PlayerView[]
  log: LogEntry[] | null
  question?: Question
  waitingFor?: string
  encounterCount?: number
  encounterDiscardCount?: number
  encounterDiscardTop?: CardRef
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
  text?: string
}

let cardsPromise: Promise<CardInfo[]> | null = null

// Full card catalog, fetched once per session and shared by consumers.
export function allCards(): Promise<CardInfo[]> {
  if (!cardsPromise) cardsPromise = get<CardInfo[]>('/marvel/cards')
  return cardsPromise
}

// Simplified-Chinese card faces (name/text/traits by code) for the Ctrl
// hover text overlay. Served as a static asset (~1MB) and fetched lazily on
// first use; en mode never loads it.
export interface ZhCardDetail {
  name: string
  subname?: string
  text?: string
  traits?: string
}

let zhDetailsPromise: Promise<Record<string, ZhCardDetail>> | null = null

export function zhCardDetails(): Promise<Record<string, ZhCardDetail>> {
  if (!zhDetailsPromise) {
    zhDetailsPromise = fetch('/zh-cards-full.json')
      .then((r) => {
        if (!r.ok) throw new Error(String(r.status))
        return r.json() as Promise<Record<string, ZhCardDetail>>
      })
      .catch((e) => {
        zhDetailsPromise = null
        throw e
      })
  }
  return zhDetailsPromise
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
