import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ChevronDown, ChevronRight, Plus, Trash2, X } from 'lucide-react'
import { api } from '../lib/api'
import PersonChip from '../components/PersonChip'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

interface TrainerEntry { userId: number; name: string }
interface Responsibility { id: number; label: string }
interface PlayerEntry {
  userId: number
  name: string
  jerseyNumber: number | null
  memberId: number
  responsibilities?: Responsibility[]
}
interface ParentEntry { userId: number; name: string; children: string[] }

interface TeamRoster {
  team: { id: number; name: string; display_short?: string; display_long?: string }
  trainers: TrainerEntry[]
  players: PlayerEntry[]
  parents: ParentEntry[]
  extended_players: PlayerEntry[]
  extended_parents: ParentEntry[]
  canManage?: boolean
}

interface MyTeam { id: number; name: string }

// Strafen-Datenmodell (Beträge in Cent)
interface Penalty { id: number; memberId: number; memberName: string; amountCent: number; reason: string; createdAt: string }
interface PenaltyTotal { memberId: number; memberName: string; totalCent: number }
interface PenaltiesData { penalties: Penalty[]; totals: PenaltyTotal[]; canLevy: boolean }

// Verwaltungs-Kataloge
interface RespType { id: number; label: string }
interface PenaltyType { id: number; reason: string; defaultAmountCent: number }
interface Strafenwart { memberId: number; name: string }

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_SMALL = 'bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'

const fmtEur = (cent: number) =>
  (cent / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })

const eurToCent = (s: string): number => {
  const n = parseFloat(s.replace(',', '.'))
  return Number.isFinite(n) ? Math.round(n * 100) : NaN
}

const fmtDate = (iso: string): string => {
  const d = iso.slice(0, 10)
  const parsed = new Date(d + 'T12:00:00')
  return Number.isNaN(parsed.getTime()) ? d : parsed.toLocaleDateString('de-DE')
}

type RosterTab = 'team' | 'trainer' | 'eltern' | 'verwalten' | 'strafen'

const BASE_TABS: { id: RosterTab; label: string }[] = [
  { id: 'team', label: 'Team' },
  { id: 'trainer', label: 'Trainer' },
  { id: 'eltern', label: 'Eltern' },
]

interface RosterSectionProps {
  roster: TeamRoster
  teamId: number
  penalties?: PenaltiesData
  penaltyHidden: boolean
  reloadRoster: () => void
  reloadPenalties: () => void
  bump: number
}

function RosterSection({ roster, teamId, penalties, penaltyHidden, reloadRoster, reloadPenalties, bump }: RosterSectionProps) {
  const [activeTab, setActiveTab] = useState<RosterTab>('team')

  const allPlayers = [...roster.players, ...roster.extended_players]

  // --- Verwaltungs-Kataloge (lazy) ---
  const [respTypes, setRespTypes] = useState<RespType[] | null>(null)
  const [penaltyTypes, setPenaltyTypes] = useState<PenaltyType[] | null>(null)
  const [strafenwarte, setStrafenwarte] = useState<Strafenwart[] | null>(null)

  const loadRespTypes = useCallback(async () => {
    try { const r = await api.get(`/teams/${teamId}/responsibility-types`); setRespTypes(r.data ?? []) } catch { /* still */ }
  }, [teamId])
  const loadPenaltyTypes = useCallback(async () => {
    try { const r = await api.get(`/teams/${teamId}/penalty-types`); setPenaltyTypes(r.data ?? []) } catch { /* still */ }
  }, [teamId])
  const loadStrafenwarte = useCallback(async () => {
    try { const r = await api.get(`/teams/${teamId}/strafenwarte`); setStrafenwarte(r.data ?? []) } catch { /* still */ }
  }, [teamId])

  // Kataloge on-demand laden (und bei Live-Event via bump neu ziehen).
  useEffect(() => {
    if (activeTab === 'verwalten' && roster.canManage) {
      loadRespTypes(); loadPenaltyTypes(); loadStrafenwarte()
    } else if (activeTab === 'strafen' && penalties?.canLevy) {
      loadPenaltyTypes()
    }
  }, [activeTab, bump, roster.canManage, penalties?.canLevy, loadRespTypes, loadPenaltyTypes, loadStrafenwarte])

  // --- Formularzustand ---
  const [newRespLabel, setNewRespLabel] = useState('')
  const [assignMember, setAssignMember] = useState('')
  const [assignLabel, setAssignLabel] = useState('')
  const [newPenaltyReason, setNewPenaltyReason] = useState('')
  const [newPenaltyEur, setNewPenaltyEur] = useState('')
  const [newStrafenwart, setNewStrafenwart] = useState('')
  const [levyMember, setLevyMember] = useState('')
  const [levyTypeId, setLevyTypeId] = useState('')
  const [levyEur, setLevyEur] = useState('')

  // --- Aufgaben-Katalog ---
  async function addRespType() {
    const label = newRespLabel.trim()
    if (!label) return
    try { await api.post(`/teams/${teamId}/responsibility-types`, { label }); setNewRespLabel(''); loadRespTypes() } catch { /* still */ }
  }
  async function delRespType(id: number) {
    try { await api.delete(`/teams/${teamId}/responsibility-types/${id}`); loadRespTypes() } catch { /* still */ }
  }

  // --- Aufgabe zuweisen / entfernen ---
  async function assignResp() {
    const label = assignLabel.trim()
    if (!assignMember || !label) return
    try {
      await api.post(`/teams/${teamId}/responsibilities`, { memberId: Number(assignMember), label })
      setAssignLabel('')
      reloadRoster()
    } catch { /* still */ }
  }
  async function removeResp(respId: number) {
    try {
      await api.delete(`/teams/${teamId}/responsibilities/${respId}`)
      reloadRoster()
    } catch { /* still */ }
  }

  // --- Strafen-Katalog ---
  async function addPenaltyType() {
    const reason = newPenaltyReason.trim()
    const cent = eurToCent(newPenaltyEur)
    if (!reason || !Number.isFinite(cent) || cent <= 0) return
    try { await api.post(`/teams/${teamId}/penalty-types`, { reason, defaultAmountCent: cent }); setNewPenaltyReason(''); setNewPenaltyEur(''); loadPenaltyTypes() } catch { /* still */ }
  }
  async function delPenaltyType(id: number) {
    try { await api.delete(`/teams/${teamId}/penalty-types/${id}`); loadPenaltyTypes() } catch { /* still */ }
  }

  // --- Strafenwart ---
  async function addStrafenwart() {
    if (!newStrafenwart) return
    try { await api.post(`/teams/${teamId}/strafenwarte`, { memberId: Number(newStrafenwart) }); setNewStrafenwart(''); loadStrafenwarte() } catch { /* still */ }
  }
  async function delStrafenwart(memberId: number) {
    try { await api.delete(`/teams/${teamId}/strafenwarte/${memberId}`); loadStrafenwarte() } catch { /* still */ }
  }

  // --- Strafe verhängen / stornieren / zurücksetzen ---
  function onLevyTypeChange(v: string) {
    setLevyTypeId(v)
    const t = (penaltyTypes ?? []).find(pt => String(pt.id) === v)
    if (t) setLevyEur((t.defaultAmountCent / 100).toFixed(2))
  }
  async function levy() {
    const t = (penaltyTypes ?? []).find(pt => String(pt.id) === levyTypeId)
    const cent = eurToCent(levyEur)
    if (!levyMember || !t || !Number.isFinite(cent) || cent <= 0) return
    try {
      await api.post(`/teams/${teamId}/penalties`, { memberId: Number(levyMember), amountCent: cent, reason: t.reason })
      setLevyMember(''); setLevyTypeId(''); setLevyEur('')
      reloadPenalties()
    } catch { /* still */ }
  }
  async function stornoPenalty(id: number) {
    try { await api.delete(`/teams/${teamId}/penalties/${id}`); reloadPenalties() } catch { /* still */ }
  }
  async function resetMember(memberId: number, name: string) {
    if (!window.confirm(`Alle Strafen von ${name} zurücksetzen?`)) return
    try { await api.delete(`/teams/${teamId}/penalties?member=${memberId}`); reloadPenalties() } catch { /* still */ }
  }

  const tabs = [...BASE_TABS]
  if (roster.canManage) tabs.push({ id: 'verwalten', label: 'Verwalten' })
  if (!penaltyHidden) tabs.push({ id: 'strafen', label: 'Strafen' })

  return (
    <>
      <div className="flex flex-wrap gap-1 mb-3">
        {tabs.map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-3 py-1 rounded-md text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'bg-brand-yellow text-brand-black'
                : 'text-brand-text-muted hover:text-brand-text'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'team' && (
        <>
          {roster.players.length === 0 ? (
            <p className="text-sm text-brand-text-muted">— keine Einträge —</p>
          ) : (
            <div className="overflow-x-auto -mx-5 px-5">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left">
                    <th className="pb-2 pr-4 text-xs text-brand-text-muted font-medium">#</th>
                    <th className="pb-2 text-xs text-brand-text-muted font-medium">Name</th>
                  </tr>
                </thead>
                <tbody>
                  {roster.players.map((p, i) => (
                    <tr key={i} className="border-t border-brand-border-subtle">
                      <td className="py-2 pr-4 text-brand-text-muted w-8">
                        {p.jerseyNumber != null ? p.jerseyNumber : '–'}
                      </td>
                      <td className="py-2">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <PersonChip userId={p.userId || undefined} name={p.name} />
                          {(p.responsibilities ?? []).map(r => (
                            <span key={r.id} className="inline-flex items-center rounded-full border border-brand-yellow px-2 py-0.5 text-xs text-brand-text-muted">
                              {r.label}
                            </span>
                          ))}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {roster.extended_players?.length > 0 && (
            <div className="mt-5">
              <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">
                Erweiterter Kader
              </p>
              <div className="overflow-x-auto -mx-5 px-5">
                <table className="w-full text-sm">
                  <tbody>
                    {roster.extended_players.map((p, i) => (
                      <tr key={i} className="border-t border-brand-border-subtle">
                        <td className="py-2 pr-4 text-brand-text-muted w-8">
                          {p.jerseyNumber != null ? p.jerseyNumber : '–'}
                        </td>
                        <td className="py-2">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <PersonChip userId={p.userId || undefined} name={p.name} />
                            {(p.responsibilities ?? []).map(r => (
                              <span key={r.id} className="inline-flex items-center rounded-full border border-brand-yellow px-2 py-0.5 text-xs text-brand-text-muted">
                                {r.label}
                              </span>
                            ))}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {activeTab === 'trainer' && (
        roster.trainers.length === 0 ? (
          <p className="text-sm text-brand-text-muted">— keine Einträge —</p>
        ) : (
          <table className="w-full text-sm">
            <tbody>
              {roster.trainers.map((t, i) => (
                <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                  <td className="py-2">
                    <PersonChip userId={t.userId || undefined} name={t.name} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )
      )}

      {activeTab === 'eltern' && (
        roster.parents.length === 0 && (roster.extended_parents?.length ?? 0) === 0 ? (
          <p className="text-sm text-brand-text-muted">— keine Einträge —</p>
        ) : (
          <>
            {roster.parents.length > 0 && (
              <table className="w-full text-sm">
                <tbody>
                  {roster.parents.map((p, i) => (
                    <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                      <td className="py-2">
                        <PersonChip userId={p.userId || undefined} name={p.name} />
                        {p.children.length > 0 && (
                          <p className="text-xs text-brand-text-muted mt-0.5">{p.children.join(', ')}</p>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {roster.extended_parents?.length > 0 && (
              <div className="mt-5">
                <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">
                  Erweiterter Kader
                </p>
                <table className="w-full text-sm">
                  <tbody>
                    {roster.extended_parents.map((p, i) => (
                      <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                        <td className="py-2">
                          <PersonChip userId={p.userId || undefined} name={p.name} />
                          {p.children.length > 0 && (
                            <p className="text-xs text-brand-text-muted mt-0.5">{p.children.join(', ')}</p>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )
      )}

      {activeTab === 'verwalten' && roster.canManage && (
        <div className="space-y-6">
          {/* Aufgaben-Katalog */}
          <div>
            <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">Aufgaben-Katalog</p>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {(respTypes ?? []).map(rt => (
                <span key={rt.id} className="inline-flex items-center gap-1 rounded-full border border-brand-border px-2 py-0.5 text-xs text-brand-text">
                  {rt.label}
                  <button onClick={() => delRespType(rt.id)} aria-label={`Aufgabe ${rt.label} löschen`} className="text-brand-text-muted hover:text-brand-danger transition-colors">
                    <X className="w-3 h-3" />
                  </button>
                </span>
              ))}
              {respTypes && respTypes.length === 0 && (
                <span className="text-xs text-brand-text-muted">— leer —</span>
              )}
            </div>
            <div className="flex gap-2">
              <input value={newRespLabel} onChange={e => setNewRespLabel(e.target.value)} placeholder="Neue Aufgabe…" className={INPUT} />
              <button onClick={addRespType} disabled={!newRespLabel.trim()} className={BTN_SMALL} aria-label="Aufgabe hinzufügen">
                <Plus className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Aufgabe zuweisen */}
          <div>
            <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">Aufgabe zuweisen</p>
            <div className="flex flex-col gap-2 sm:flex-row">
              <select value={assignMember} onChange={e => setAssignMember(e.target.value)} className={INPUT}>
                <option value="">Spieler wählen…</option>
                {allPlayers.map(p => (
                  <option key={p.memberId} value={p.memberId}>{p.name}</option>
                ))}
              </select>
              <input
                value={assignLabel}
                onChange={e => setAssignLabel(e.target.value)}
                list={`resp-catalog-${teamId}`}
                placeholder="Aufgabe (Katalog oder frei)…"
                className={INPUT}
              />
              <datalist id={`resp-catalog-${teamId}`}>
                {(respTypes ?? []).map(rt => <option key={rt.id} value={rt.label} />)}
              </datalist>
              <button onClick={assignResp} disabled={!assignMember || !assignLabel.trim()} className={BTN_SMALL} aria-label="Aufgabe zuweisen">
                <Plus className="w-4 h-4" />
              </button>
            </div>
            <div className="mt-3 space-y-1.5">
              {allPlayers.filter(p => (p.responsibilities ?? []).length > 0).map(p => (
                <div key={p.memberId} className="flex flex-wrap items-center gap-1.5 text-sm">
                  <span className="text-brand-text-muted">{p.name}:</span>
                  {(p.responsibilities ?? []).map(r => (
                    <span key={r.id} className="inline-flex items-center gap-1 rounded-full border border-brand-yellow px-2 py-0.5 text-xs text-brand-text">
                      {r.label}
                      <button onClick={() => removeResp(r.id)} aria-label={`Aufgabe ${r.label} entfernen`} className="text-brand-text-muted hover:text-brand-danger transition-colors">
                        <X className="w-3 h-3" />
                      </button>
                    </span>
                  ))}
                </div>
              ))}
            </div>
          </div>

          {/* Strafen-Katalog */}
          <div>
            <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">Strafen-Katalog</p>
            <div className="space-y-1.5 mb-2">
              {(penaltyTypes ?? []).map(pt => (
                <div key={pt.id} className="flex items-center justify-between text-sm">
                  <span className="text-brand-text">{pt.reason} <span className="text-brand-text-muted">· {fmtEur(pt.defaultAmountCent)}</span></span>
                  <button onClick={() => delPenaltyType(pt.id)} aria-label={`Strafe ${pt.reason} löschen`} className="text-brand-text-muted hover:text-brand-danger transition-colors">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))}
              {penaltyTypes && penaltyTypes.length === 0 && (
                <span className="text-xs text-brand-text-muted">— leer —</span>
              )}
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input value={newPenaltyReason} onChange={e => setNewPenaltyReason(e.target.value)} placeholder="Grund…" className={INPUT} />
              <input value={newPenaltyEur} onChange={e => setNewPenaltyEur(e.target.value)} inputMode="decimal" placeholder="Betrag €" className={INPUT} />
              <button onClick={addPenaltyType} disabled={!newPenaltyReason.trim() || !(eurToCent(newPenaltyEur) > 0)} className={BTN_SMALL} aria-label="Strafe hinzufügen">
                <Plus className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Strafenwart */}
          <div>
            <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">Strafenwart</p>
            <div className="flex flex-wrap gap-1.5 mb-2">
              {(strafenwarte ?? []).map(sw => (
                <span key={sw.memberId} className="inline-flex items-center gap-1 rounded-full border border-brand-border px-2 py-0.5 text-xs text-brand-text">
                  {sw.name}
                  <button onClick={() => delStrafenwart(sw.memberId)} aria-label={`Strafenwart ${sw.name} entfernen`} className="text-brand-text-muted hover:text-brand-danger transition-colors">
                    <X className="w-3 h-3" />
                  </button>
                </span>
              ))}
              {strafenwarte && strafenwarte.length === 0 && (
                <span className="text-xs text-brand-text-muted">— keiner —</span>
              )}
            </div>
            <div className="flex gap-2">
              <select value={newStrafenwart} onChange={e => setNewStrafenwart(e.target.value)} className={INPUT}>
                <option value="">Spieler wählen…</option>
                {allPlayers.map(p => (
                  <option key={p.memberId} value={p.memberId}>{p.name}</option>
                ))}
              </select>
              <button onClick={addStrafenwart} disabled={!newStrafenwart} className={BTN_SMALL} aria-label="Strafenwart ernennen">
                <Plus className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'strafen' && !penaltyHidden && (
        !penalties ? (
          <div className="h-16 bg-brand-border-subtle rounded-lg animate-pulse" />
        ) : (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-semibold text-brand-text">Mannschaftskasse</span>
              <span className="text-sm font-semibold text-brand-text">
                {fmtEur(penalties.totals.reduce((s, t) => s + t.totalCent, 0))}
              </span>
            </div>

            {penalties.totals.length === 0 ? (
              <p className="text-sm text-brand-text-muted">— keine Strafen —</p>
            ) : (
              penalties.totals.map(t => (
                <div key={t.memberId} className="border-t border-brand-border-subtle pt-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-brand-text">{t.memberName}</span>
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-brand-text-muted">{fmtEur(t.totalCent)}</span>
                      {penalties.canLevy && (
                        <button onClick={() => resetMember(t.memberId, t.memberName)} className="text-xs text-brand-text-muted hover:text-brand-danger transition-colors">
                          Zurücksetzen
                        </button>
                      )}
                    </div>
                  </div>
                  <ul className="mt-1.5 space-y-1">
                    {penalties.penalties.filter(p => p.memberId === t.memberId).map(p => (
                      <li key={p.id} className="flex items-center justify-between text-sm">
                        <span className="text-brand-text">
                          {p.reason}
                          <span className="text-brand-text-muted"> · {fmtEur(p.amountCent)} · {fmtDate(p.createdAt)}</span>
                        </span>
                        {penalties.canLevy && (
                          <button onClick={() => stornoPenalty(p.id)} aria-label={`Strafe stornieren: ${p.reason}`} className="text-brand-text-muted hover:text-brand-danger transition-colors">
                            <Trash2 className="w-4 h-4" />
                          </button>
                        )}
                      </li>
                    ))}
                  </ul>
                </div>
              ))
            )}

            {penalties.canLevy && (
              <div className="border-t border-brand-border-subtle pt-4">
                <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">Strafe verhängen</p>
                {penaltyTypes && penaltyTypes.length === 0 ? (
                  <p className="text-xs text-brand-text-muted">Kein Strafen-Katalog angelegt.</p>
                ) : (
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <select value={levyMember} onChange={e => setLevyMember(e.target.value)} className={INPUT}>
                      <option value="">Spieler wählen…</option>
                      {allPlayers.map(p => (
                        <option key={p.memberId} value={p.memberId}>{p.name}</option>
                      ))}
                    </select>
                    <select value={levyTypeId} onChange={e => onLevyTypeChange(e.target.value)} className={INPUT}>
                      <option value="">Grund wählen…</option>
                      {(penaltyTypes ?? []).map(pt => (
                        <option key={pt.id} value={pt.id}>{pt.reason}</option>
                      ))}
                    </select>
                    <input value={levyEur} onChange={e => setLevyEur(e.target.value)} inputMode="decimal" placeholder="Betrag €" className={INPUT} />
                    <button onClick={levy} disabled={!levyMember || !levyTypeId || !(eurToCent(levyEur) > 0)} className={BTN_SMALL} aria-label="Strafe verhängen">
                      <Plus className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        )
      )}
    </>
  )
}

export default function MeinTeamPage() {
  const [searchParams] = useSearchParams()
  const focusTeamId = searchParams.get('team') ? Number(searchParams.get('team')) : null

  const [myTeams, setMyTeams] = useState<MyTeam[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // On-Demand-Rosters: erst beim Aufklappen/Fokus geladen, dann in der Session
  // behalten (kein Re-Fetch beim erneuten Aufklappen).
  const [rosters, setRosters] = useState<Record<number, TeamRoster>>({})
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [rosterErrors, setRosterErrors] = useState<Record<number, string>>({})

  // Strafen: on-demand beim Aufklappen. 403 (Eltern/Externe) → Sektion verstecken.
  const [penalties, setPenalties] = useState<Record<number, PenaltiesData>>({})
  const [penaltyHidden, setPenaltyHidden] = useState<Set<number>>(new Set())

  // Zähler, der bei relevanten Live-Events hochzählt → RosterSection zieht seine
  // bereits geladenen Kataloge neu.
  const [liveBump, setLiveBump] = useState(0)

  const loadRoster = useCallback(async (teamId: number) => {
    try {
      const r = await api.get(`/teams/${teamId}/roster`)
      setRosters(prev => ({ ...prev, [teamId]: r.data as TeamRoster }))
    } catch (err) {
      setRosterErrors(prev => ({ ...prev, [teamId]: err instanceof Error ? err.message : 'Fehler beim Laden' }))
    }
  }, [])

  const loadPenalties = useCallback(async (teamId: number) => {
    try {
      const r = await api.get(`/teams/${teamId}/penalties`)
      setPenalties(prev => ({ ...prev, [teamId]: r.data as PenaltiesData }))
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 403) setPenaltyHidden(prev => new Set(prev).add(teamId))
      // andere Fehler: still, Sektion bleibt einfach ladend/leer
    }
  }, [])

  const toggleTeam = useCallback((teamId: number) => {
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(teamId)) next.delete(teamId)
      else next.add(teamId)
      return next
    })
  }, [])

  // On-Demand-Laden: sobald ein Team aufgeklappt ist und sein Roster weder im
  // Session-Cache noch als Fehler vorliegt, wird es geladen. Bereits geladene
  // Rosters bleiben erhalten (kein Re-Fetch beim erneuten Aufklappen).
  useEffect(() => {
    for (const teamId of expanded) {
      if (!rosters[teamId] && !rosterErrors[teamId]) loadRoster(teamId)
      if (!penalties[teamId] && !penaltyHidden.has(teamId)) loadPenalties(teamId)
    }
  }, [expanded, rosters, rosterErrors, penalties, penaltyHidden, loadRoster, loadPenalties])

  const loadTeams = useCallback(() => {
    api.get('/teams/my')
      .then(res => {
        const teams: MyTeam[] = res.data ?? []
        setMyTeams(teams)
        setLoading(false)
        // Fokus-/Einzelteam wird automatisch aufgeklappt (→ Roster lädt on-demand
        // via Effekt); alle anderen bleiben eingeklappt und laden erst bei Fokus.
        const autoOpen = focusTeamId != null
          ? teams.filter(t => t.id === focusTeamId)
          : teams.length === 1 ? teams : []
        if (autoOpen.length > 0) setExpanded(new Set(autoOpen.map(t => t.id)))
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [focusTeamId])

  // Nur die Team-Liste eager laden; Rosters folgen on-demand.
  // focusTeamId/loadTeams stabil; bewusst nur bei Fokuswechsel neu laufen.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { loadTeams() }, [focusTeamId])

  // Bei Mitglieds-/Kader-/Aufgaben-/Strafen-Änderungen: geladene Daten auffrischen.
  useLiveUpdates(event => {
    if (event === 'members' || event === 'kader') {
      for (const idStr of Object.keys(rosters)) loadRoster(Number(idStr))
    }
    if (event === 'responsibilities') {
      for (const idStr of Object.keys(rosters)) loadRoster(Number(idStr))
      setLiveBump(b => b + 1)
    }
    if (event === 'penalties') {
      for (const idStr of Object.keys(penalties)) loadPenalties(Number(idStr))
      setLiveBump(b => b + 1)
    }
  })

  if (loading) {
    return (
      <div className="max-w-3xl mx-auto space-y-3">
        {[1, 2].map(i => <div key={i} className="h-32 bg-brand-border-subtle rounded-xl animate-pulse" />)}
      </div>
    )
  }

  if (error) {
    return (
      <div className="max-w-3xl mx-auto py-8 text-center">
        <p className="text-sm text-brand-text-muted">{error}</p>
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-text">Mein Team</h1>
        {myTeams.length > 1 && (
          <p className="text-sm text-brand-text-muted mt-0.5">{myTeams.length} Teams</p>
        )}
      </div>

      {myTeams.length === 0 ? (
        <p className="text-sm text-brand-text-muted">Kein Team zugeordnet.</p>
      ) : (
        <div className="space-y-4">
          {myTeams.map(team => {
            const isOpen = expanded.has(team.id)
            const roster = rosters[team.id]
            const rosterError = rosterErrors[team.id]
            return (
              <div key={team.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
                <button
                  onClick={() => toggleTeam(team.id)}
                  aria-expanded={isOpen}
                  className="w-full flex items-center justify-between px-5 py-4 hover:bg-brand-border-subtle transition-colors min-h-[44px]"
                >
                  <h2 className="text-lg font-bold text-brand-text text-left">{roster?.team.display_long || team.name}</h2>
                  {isOpen
                    ? <ChevronDown className="w-5 h-5 text-brand-text-muted shrink-0" />
                    : <ChevronRight className="w-5 h-5 text-brand-text-muted shrink-0" />
                  }
                </button>
                {isOpen && (
                  <div className="px-5 py-4 border-t border-brand-border-subtle">
                    {rosterError ? (
                      <p className="text-sm text-brand-danger">{rosterError}</p>
                    ) : roster ? (
                      <RosterSection
                        roster={roster}
                        teamId={team.id}
                        penalties={penalties[team.id]}
                        penaltyHidden={penaltyHidden.has(team.id)}
                        reloadRoster={() => loadRoster(team.id)}
                        reloadPenalties={() => loadPenalties(team.id)}
                        bump={liveBump}
                      />
                    ) : (
                      <div className="h-20 bg-brand-border-subtle rounded-lg animate-pulse" />
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
