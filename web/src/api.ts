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
  // 结构化的牌组校验问题（开局/入座被拒时服务端带回），无则缺省。
  deckIssues?: DeckIssue[]
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
    let deckIssues: DeckIssue[] | undefined
    try {
      const data = await resp.json()
      if (data.error) msg = data.error
      if (Array.isArray(data.deckIssues)) deckIssues = data.deckIssues
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
    const err = new ApiError(resp.status, msg)
    err.deckIssues = deckIssues
    throw err
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
  // 副牌组（如奇异博士的召唤牌组）：数量对所有副牌组英雄公开；顶牌与
  // 弃牌堆顶只在明置副牌组（召唤牌组）暴露，海克力斯的 Labor/Gift 暗置。
  senseDeckCount?: number
  sideDeckTop?: CardRef
  sideDiscardCount?: number
  sideDiscardTop?: CardRef
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
  id: string
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

// 一条组牌规则违反（engine.ValidateDeck）：key 对应 deck.issue.* 文案键，
// card/title 指向违规卡，n/m 携带数量参数。
export interface DeckIssue {
  key: string
  card?: string
  title?: string
  aspect?: string
  n?: number
  m?: number
}

export interface Deck {
  id: string
  name: string
  investigatorCode: string
  slots: Record<string, number>
  // 组牌规则校验结果：不合规牌组仍可导入/查看，只是不能用于开局。
  valid?: boolean
  issues?: DeckIssue[]
}

// 身份卡上「所选派系之外也可加入……」骑手的结构化形式（服务端
// data.AspectException 的镜像）；匹配只读这些字段，绝不解析卡面文本。
export interface AspectException {
  trait?: string
  cardType?: string
  eventTraits?: string[]
  energyEvents?: boolean
  total?: number
  titles?: number
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
  cardSet?: string
  cost?: number | null
  unique: boolean
  traits?: string[]
  // English print traits — deckbuilding logic (aspect-exception matching)
  // must key on these: under the server-wide zh overlay (MC_ZH_DIR) the
  // `traits` field carries translated strings.
  etraits?: string[]
  resources?: string[]
  text?: string
  quantity?: number
  implemented?: boolean
  // Printed hero/ally stats (deck-detail hero panel); absent when unprinted.
  hp?: number
  attack?: number
  thwart?: number
  defense?: number
  recover?: number
  handSize?: number
  // 组牌骑手（印在化身面上）：组牌器据此驱动派系选择器、例外卡放行与
  // 复制上限，与 engine.ValidateDeck 同源。
  aspectMode?: string
  aspectException?: AspectException
  uniqueAll?: boolean
}

// GET /marvel/heroes 的一行：一个可选英雄身份。
export interface HeroInfo {
  base: string
  heroCode: string
  alterEgoCode: string
  name: string
  alterEgoName?: string
  packCode: string
  implemented: boolean
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
  id: string
  name: string
  scenarioId: string
  status: string
  updatedAt: string
}

// Lobby projection served by GET /games/{id}/lobby while the game waits for
// players; the endpoint 404s once started so pollers switch to the board.
export interface LobbyDeck {
  id: string
  name: string
  heroCode: string
  valid?: boolean
}

export interface LobbyPlayer {
  userId: string
  username: string
  slot: number
  host: boolean
  deck: LobbyDeck | null
}

export interface LobbyView {
  id: string
  name: string
  scenarioId: string
  difficulty: string
  status: string
  playerCount: number
  players: LobbyPlayer[]
  openSlots: number
}

// -------------------------------------------------------------- campaigns

export interface CampaignPlayerLog {
  slot: number
  userId?: number
  name: string
  heroBase: string
  deck: Record<string, number>
  hp?: number
  units?: number
  market?: string[]
  healNext?: boolean
  tech?: string
  condition?: string
  improved?: boolean
  allies?: string[]
  engagedEnemy?: boolean
  smTech?: string
  smTechOffer?: string[]
  smAspect?: string
  smPlanning?: string
  smEnhanced?: boolean
  setupHand?: string
  mgRole?: string
  // Contest campaigns.
  wiTrait?: string
  wiAllies?: string[]
  wiRewards?: string[]
  influence?: number
  awAlly?: string
  awIdentity?: string
  mojoRole?: string
  mojoMarket?: string
  mojoScheme?: string
  mojoEvent?: string
  bordObligations?: string[]
  bordGear?: string[]
}

export interface CampaignState {
  box: string
  difficulty: string
  index: number
  status: string
  players: CampaignPlayerLog[]
  experimental?: string[]
  delayCounters?: number
  removedAllies?: string[]
  collection?: string[]
  artifacts?: string[]
  headhunter?: boolean[]
  powerStone?: number
  evasion?: number
  pendingChoices?: Record<string, string>
  lastResult?: 'won' | 'lost' | ''
  pool?: string[]
  flags?: Record<string, boolean>
  counters?: Record<string, number>
  smOsborn?: string[]
  smCommunity?: string[]
  smWaking?: number
  smLastStanding?: string[]
  mgFuturePast?: string[]
  mgCaptives?: string[]
  mgRemovedAllies?: string[]
  nxEnvEarned?: string[]
  nxChosen?: string[]
  nxCurrent?: string
  aoMission?: string
  aoOverseer?: string
  aoMissionLog?: string[]
  aoOverseerLog?: string[]
  aoShieldEnvelope?: string[]
  aoEvidence?: string[]
  aoCounters?: Record<string, number>
  aoSurvivors?: string[]
  // Contest campaigns.
  selections?: Record<string, string>
  victims?: string[]
  cowlCaught?: string[]
}

export interface CampaignSeat {
  slot: number
  userId?: string
  username: string
  hero: string
  deckName: string
}

export interface CampaignChapter {
  id: string
  name: string
  /** Identity this chapter requires (The Watcher's Team). */
  requires?: string
}

export interface CampaignGameRef {
  id: string
  name: string
  scenarioId: string
  status: string
}

export interface MarketCard {
  code: string
  name: string
  cost: number
}

export interface CampaignDetail {
  id: string
  box: string
  name: string
  desc?: string
  difficulty: string
  status: string
  index: number
  playerCount: number
  host: boolean
  yourSlot: number
  state: CampaignState
  seats: CampaignSeat[]
  chapters: CampaignChapter[]
  games: CampaignGameRef[]
  market: MarketCard[]
  /** EN display names for every campaign card code in the log. */
  names: Record<string, string>
  /** Choice pools by pending-choice kind. */
  pools: {
    tech: string[]
    condition: string[]
    roles: string[]
    nx: string[]
    aosMembers: string[]
    smTech?: string[]
    community?: string[]
    traits?: string[]
    soe?: string[]
    viralNext?: string[]
    allNx?: string[]
  }
  /** Per-box choice tables (role cards, shop entries, path labels...). */
  tables?: Record<string, unknown>
}

export interface CampaignBoxInfo {
  key: string
  name: string
  desc: string
  scenarios: number
}

export interface CampaignSummary {
  id: string
  box: string
  name: string
  status: string
  index: number
  playerCount: number
  updatedAt: string
}

export const listCampaigns = () => get<CampaignSummary[]>('/marvel/campaigns')
export const listCampaignBoxes = () => get<CampaignBoxInfo[]>('/marvel/campaigns/boxes')
export const getCampaign = (id: string) => get<CampaignDetail>(`/marvel/campaigns/${id}`)
export const createCampaign = (body: { box: string; difficulty: string; playerCount: number; deckId?: string }) =>
  post<CampaignDetail>('/marvel/campaigns', body)
export const joinCampaign = (id: string, deckId: string) => post<CampaignDetail>(`/marvel/campaigns/${id}/join`, { deckId })
export const kickCampaign = (id: string, slot: number) => post<CampaignDetail>(`/marvel/campaigns/${id}/kick`, { slot })
export const startCampaign = (id: string) => post<CampaignDetail>(`/marvel/campaigns/${id}/start`)
export const playCampaign = (id: string) => post<{ gameId: string }>(`/marvel/campaigns/${id}/play`)
export const campaignChoice = (id: string, cardCode: string, kind?: string) =>
  post<CampaignDetail>(`/marvel/campaigns/${id}/choice`, kind ? { cardCode, kind } : { cardCode })
export const swapCampaignDeck = (id: string, deckId: string) =>
  post<CampaignDetail>(`/marvel/campaigns/${id}/deck`, { deckId })
export const buyMarket = (id: string, cardCode: string) => post<CampaignDetail>(`/marvel/campaigns/${id}/market`, { cardCode })
export const setCampaignHeal = (id: string, on: boolean) => post<CampaignDetail>(`/marvel/campaigns/${id}/heal`, { on })
