import { useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Check, ChevronDown, ChevronRight, Copy, Home, Lightbulb, MapPin, X } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'

// Import des H4A-Spielplans in zwei Schritten:
//   1. Zugangsdaten + Periode  → POST /games/import/h4a/preview  (liest Handball4All)
//   2. Diff bestätigen         → POST /games/import/h4a/apply    (schreibt, ohne H4A)
// Die Zugangsdaten leben ausschließlich im Komponenten-State bis der Plan geladen
// ist und werden danach sofort verworfen — sie gehen nie in einen Store, ins
// sessionStorage oder in einen zweiten Request (siehe design.md §2/§3).
//
// Die aufrufende Seite mountet das Modal nur im geöffneten Zustand; beim Schließen
// räumt React den kompletten State ab. Das ist bewusst kein Reset-Effect: die
// Zugangsdaten sollen mit dem Unmount verschwinden, nicht durch eine Aufräumroutine,
// die man beim Erweitern des States vergessen kann.

export interface H4APlanGame {
  game_no: string
  staffel: string
  club_alias: string
  opponent: string
  date: string
  time: string
  is_home: boolean
  event_type: string
  hall_number: string
  team_id: number | null
  team_name: string
  /** 'gelernt' = bestätigte Zuordnung aus früherem Import, 'vorschlag' = aus dem
   *  Staffelcode abgeleitet und noch ungeprüft. */
  team_source?: 'gelernt' | 'vorschlag'
  venue_id: number | null
  venue_name: string
  status: string
  changes?: { field: string; old: string; new: string }[]
  existing_game_id?: number
  duplicate_of_game_id?: number
  warnings?: string[]
}

interface PreviewResponse {
  needs_period?: boolean
  periods?: { ID: string; Name: string }[]
  new?: H4APlanGame[]
  changed?: H4APlanGame[]
  unchanged?: H4APlanGame[]
  warnings?: string[]
}

export interface H4AApplyResult {
  imported: number
  updated: number
  skipped: number
  regen_summary?: unknown
}

interface Team { id: number; name: string; is_active?: boolean }
interface Template { id: number; name: string; template_type: string }

interface Props {
  isOpen: boolean
  onClose: () => void
  onImported: (result: H4AApplyResult) => void
}

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const SELECT_SM = 'border border-brand-border rounded-md px-2 py-1 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_PRIMARY = 'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_SECONDARY = 'px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text hover:bg-brand-surface-card disabled:opacity-40 disabled:cursor-not-allowed transition-colors'

const FIELD_LABELS: Record<string, string> = {
  date: 'Datum',
  time: 'Uhrzeit',
  opponent: 'Gegner',
  is_home: 'Heim/Auswärts',
  venue: 'Spielort',
}

function rowKey(g: H4APlanGame): string {
  return g.game_no
}

export default function H4AImportModal({ isOpen, onClose, onImported }: Props) {
  const [step, setStep] = useState<'credentials' | 'diff'>('credentials')
  const [user, setUser] = useState('')
  const [pw, setPw] = useState('')
  const [periods, setPeriods] = useState<{ ID: string; Name: string }[]>([])
  const [periodId, setPeriodId] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [plan, setPlan] = useState<PreviewResponse | null>(null)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [teamOverrides, setTeamOverrides] = useState<Record<string, number>>({})
  const [templateOverrides, setTemplateOverrides] = useState<Record<string, number>>({})
  const [batchTemplate, setBatchTemplate] = useState<Record<string, number>>({})
  const [showUnchanged, setShowUnchanged] = useState(false)

  const [teams, setTeams] = useState<Team[]>([])
  const [templates, setTemplates] = useState<Template[]>([])

  useEscapeKey(isOpen && !busy ? onClose : null)

  useEffect(() => {
    if (!isOpen) return
    api.get<Team[]>('/teams').then(r => setTeams(r.data ?? [])).catch(() => setTeams([]))
    api.get<Template[]>('/duty-templates').then(r => setTemplates(r.data ?? [])).catch(() => setTemplates([]))
  }, [isOpen])

  const allRows = useMemo(
    () => [...(plan?.new ?? []), ...(plan?.changed ?? [])],
    [plan],
  )

  function describeError(err: unknown): string {
    const status = (err as { response?: { status?: number; data?: { error?: string } } })?.response?.status
    const code = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
    if (status === 502 && code === 'h4a_login_failed') {
      return 'Anmeldung bei Handball4All fehlgeschlagen. Bitte Zugangsdaten prüfen.'
    }
    if (status === 502 && code === 'h4a_parse_failed') {
      return 'Handball4All hat den Spielplan in einem unbekannten Format geliefert. Der Import-Adapter muss angepasst werden.'
    }
    if (status === 502) return 'Handball4All ist derzeit nicht erreichbar. Bitte später erneut versuchen.'
    if (status === 403) return 'Keine Berechtigung für den Spielplan-Import.'
    if (status === 400) return 'Bitte Benutzername und Passwort ausfüllen.'
    return 'Der Import ist fehlgeschlagen.'
  }

  async function loadPeriods() {
    setBusy(true)
    setError(null)
    try {
      const r = await api.post<PreviewResponse>('/games/import/h4a/preview', { user, pw })
      setPeriods(r.data.periods ?? [])
      if ((r.data.periods ?? []).length > 0) setPeriodId(r.data.periods![0].ID)
    } catch (err) {
      setError(describeError(err))
    } finally {
      setBusy(false)
    }
  }

  async function loadPlan() {
    setBusy(true)
    setError(null)
    try {
      const r = await api.post<PreviewResponse>('/games/import/h4a/preview', {
        user, pw, period_id: periodId,
      })
      setPlan(r.data)
      // Vorauswahl: alles Importierbare (Mannschaft aufgelöst) ist angehakt.
      const preselected = new Set<string>()
      for (const g of [...(r.data.new ?? []), ...(r.data.changed ?? [])]) {
        if (g.team_id != null) preselected.add(rowKey(g))
      }
      setSelected(preselected)
      setStep('diff')
    } catch (err) {
      setError(describeError(err))
    } finally {
      // Zugangsdaten werden nicht mehr gebraucht — apply läuft ohne H4A.
      setPw('')
      setBusy(false)
    }
  }

  function effectiveTeamId(g: H4APlanGame): number | null {
    return teamOverrides[rowKey(g)] ?? g.team_id ?? null
  }

  function toggleRow(key: string) {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  // Eine Mannschaft von Hand zuzuordnen ist die bewusste Entscheidung, die Zeile
  // zu übernehmen — sonst müsste man zusätzlich die Checkbox anhaken, die vorher
  // gesperrt war und deshalb nie angehakt sein kann.
  function changeTeam(key: string, id: number) {
    setTeamOverrides(prev => ({ ...prev, [key]: id }))
    setSelected(prev => new Set(prev).add(key))
  }

  function changeTemplate(key: string, id: number | null) {
    setTemplateOverrides(prev => {
      const next = { ...prev }
      if (id == null) delete next[key]
      else next[key] = id
      return next
    })
  }

  function effectiveTemplateId(g: H4APlanGame): number | null {
    const own = templateOverrides[rowKey(g)]
    if (own != null) return own
    return batchTemplate[g.event_type] ?? null
  }

  async function apply() {
    setBusy(true)
    setError(null)
    try {
      const decisions = allRows
        .filter(g => selected.has(rowKey(g)) && effectiveTeamId(g) != null)
        .map(g => ({
          game_no: g.game_no,
          staffel: g.staffel,
          club_alias: g.club_alias,
          team_id: effectiveTeamId(g),
          venue_id: g.venue_id,
          opponent: g.opponent,
          date: g.date,
          time: g.time,
          event_type: g.event_type,
          template_id: effectiveTemplateId(g),
        }))
      const r = await api.post<H4AApplyResult>('/games/import/h4a/apply', { decisions })
      onImported(r.data)
      onClose()
    } catch (err) {
      setError(describeError(err))
    } finally {
      setBusy(false)
    }
  }

  if (!isOpen) return null

  const importable = allRows.filter(g => selected.has(rowKey(g)) && effectiveTeamId(g) != null).length

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={busy ? undefined : onClose} />
      <div className="relative bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow max-w-4xl mx-4 w-full max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
          <h2 className="text-lg font-bold text-brand-text">Spielplan aus Handball4All importieren</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4 overflow-y-auto">
          {error && (
            <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </div>
          )}

          {step === 'credentials' && (
            <>
              <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                Die Zugangsdaten werden nur für diesen Abruf verwendet und weder gespeichert
                noch protokolliert.
              </div>
              <div>
                <label htmlFor="h4a-user" className="block text-sm font-medium text-brand-text mb-1">
                  H4A-Benutzername
                </label>
                <input
                  id="h4a-user"
                  className={INPUT}
                  value={user}
                  autoComplete="off"
                  onChange={e => setUser(e.target.value)}
                  placeholder="v_109"
                />
              </div>
              <div>
                <label htmlFor="h4a-pw" className="block text-sm font-medium text-brand-text mb-1">
                  H4A-Passwort
                </label>
                <input
                  id="h4a-pw"
                  type="password"
                  className={INPUT}
                  value={pw}
                  autoComplete="off"
                  onChange={e => setPw(e.target.value)}
                />
              </div>

              {periods.length > 0 && (
                <div>
                  <label htmlFor="h4a-period" className="block text-sm font-medium text-brand-text mb-1">
                    Spielperiode
                  </label>
                  <select
                    id="h4a-period"
                    className={INPUT}
                    value={periodId}
                    onChange={e => setPeriodId(e.target.value)}
                  >
                    {periods.map(p => (
                      <option key={p.ID} value={p.ID}>{p.Name}</option>
                    ))}
                  </select>
                </div>
              )}
            </>
          )}

          {step === 'diff' && plan && (
            <>
              <div className="flex flex-wrap items-center gap-3 text-sm text-brand-text-muted">
                <span>{plan.new?.length ?? 0} neu</span>
                <span>{plan.changed?.length ?? 0} geändert</span>
                <span>{plan.unchanged?.length ?? 0} unverändert</span>
              </div>

              <div className="flex flex-wrap items-end gap-3">
                {(['heim', 'auswärts'] as const).map(type => (
                  <div key={type}>
                    <label htmlFor={`h4a-batch-${type}`} className="block text-xs text-brand-text-muted mb-1">
                      Dienst-Vorlage für alle {type === 'heim' ? 'Heimspiele' : 'Auswärtsspiele'}
                    </label>
                    <select
                      id={`h4a-batch-${type}`}
                      className={SELECT_SM}
                      value={batchTemplate[type] ?? ''}
                      onChange={e => setBatchTemplate(prev => {
                        const next = { ...prev }
                        if (e.target.value === '') delete next[type]
                        else next[type] = Number(e.target.value)
                        return next
                      })}
                    >
                      <option value="">Keine</option>
                      {templates
                        .filter(tpl => tpl.template_type === type || tpl.template_type === 'generisch')
                        .map(tpl => (
                          <option key={tpl.id} value={tpl.id}>{tpl.name}</option>
                        ))}
                    </select>
                  </div>
                ))}
              </div>

              {([['Neu', plan.new ?? []], ['Geändert', plan.changed ?? []]] as const).map(([title, games]) => (
                <PlanSection
                  key={title}
                  title={title}
                  games={games}
                  selected={selected}
                  onToggle={toggleRow}
                  teams={teams}
                  templates={templates}
                  teamOverrides={teamOverrides}
                  onTeamChange={changeTeam}
                  templateOverrides={templateOverrides}
                  batchTemplate={batchTemplate}
                  onTemplateChange={changeTemplate}
                />
              ))}

              {(plan.unchanged?.length ?? 0) > 0 && (
                <div>
                  <button
                    onClick={() => setShowUnchanged(v => !v)}
                    className="flex items-center gap-1 text-sm font-medium text-brand-text-muted hover:text-brand-text transition-colors"
                  >
                    {showUnchanged ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                    Unverändert ({plan.unchanged?.length ?? 0})
                  </button>
                  {showUnchanged && (
                    <ul className="mt-2 space-y-1">
                      {plan.unchanged?.map(g => (
                        <li key={rowKey(g)} className="text-xs text-brand-text-muted">
                          Nr. {g.game_no} · {g.date} {g.time} · {g.staffel} · {g.opponent}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </>
          )}
        </div>

        <div className="flex gap-2 justify-end px-6 py-4 border-t border-brand-border-subtle">
          <button onClick={onClose} disabled={busy} className={BTN_SECONDARY}>Abbrechen</button>
          {step === 'credentials' && periods.length === 0 && (
            <button onClick={loadPeriods} disabled={busy || !user || !pw} className={BTN_PRIMARY}>
              {busy ? 'Verbindet…' : 'Verbinden'}
            </button>
          )}
          {step === 'credentials' && periods.length > 0 && (
            <button onClick={loadPlan} disabled={busy || !periodId} className={BTN_PRIMARY}>
              {busy ? 'Lädt Spielplan…' : 'Spielplan laden'}
            </button>
          )}
          {step === 'diff' && (
            <button onClick={apply} disabled={busy || importable === 0} className={BTN_PRIMARY}>
              {busy ? 'Importiert…' : `${importable} Spiele importieren`}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

interface SectionProps {
  title: string
  games: H4APlanGame[]
  selected: Set<string>
  onToggle: (key: string) => void
  teams: Team[]
  templates: Template[]
  teamOverrides: Record<string, number>
  onTeamChange: (key: string, id: number) => void
  templateOverrides: Record<string, number>
  batchTemplate: Record<string, number>
  onTemplateChange: (key: string, id: number | null) => void
}

function PlanSection({
  title, games, selected, onToggle, teams, templates,
  teamOverrides, onTeamChange, templateOverrides, batchTemplate, onTemplateChange,
}: SectionProps) {
  if (games.length === 0) return null

  return (
    <section>
      <h3 className="text-sm font-bold text-brand-text mb-2">{title} ({games.length})</h3>
      <ul className="space-y-2">
        {games.map(g => {
          const key = rowKey(g)
          const teamId = teamOverrides[key] ?? g.team_id ?? null
          const unresolved = teamId == null
          const templateId = templateOverrides[key] ?? batchTemplate[g.event_type] ?? null
          return (
            <li
              key={key}
              className={`rounded-lg border p-3 ${unresolved ? 'border-brand-danger/30 bg-brand-danger-light' : 'border-brand-border-subtle bg-brand-surface-card'}`}
            >
              <div className="flex items-start gap-3">
                <input
                  type="checkbox"
                  aria-label={`Spiel ${g.game_no} übernehmen`}
                  checked={selected.has(key)}
                  disabled={unresolved}
                  onChange={() => onToggle(key)}
                  className="mt-1 accent-brand-yellow disabled:opacity-40"
                />
                <div className="flex-1 min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2 text-sm text-brand-text">
                    {g.is_home ? <Home className="w-4 h-4" /> : <MapPin className="w-4 h-4" />}
                    <span className="font-medium">{g.date} {g.time || '—'}</span>
                    <span>{g.opponent}</span>
                    <span className="text-xs text-brand-text-muted">Nr. {g.game_no} · {g.staffel}</span>
                  </div>
                  <div className="text-xs text-brand-text-muted">
                    {g.venue_id != null
                      ? g.venue_name
                      : g.hall_number
                        ? `Halle ${g.hall_number} — kein Spielort zugeordnet`
                        : 'kein Spielort'}
                  </div>

                  {g.changes && g.changes.length > 0 && (
                    <ul className="text-xs text-brand-text">
                      {g.changes.map(c => (
                        <li key={c.field}>
                          {FIELD_LABELS[c.field] ?? c.field}: <span className="line-through text-brand-text-muted">{c.old}</span>
                          {' → '}
                          <span className="font-medium">{c.new}</span>
                        </li>
                      ))}
                    </ul>
                  )}

                  {g.duplicate_of_game_id != null && (
                    <div className="flex items-center gap-1 text-xs text-brand-danger">
                      <Copy className="w-3.5 h-3.5" />
                      Mögliche Dublette zu einem bereits angelegten Termin
                    </div>
                  )}

                  {g.warnings?.filter(wn => !wn.startsWith('mögliche Dublette') && !wn.startsWith('Mannschaft vorgeschlagen')).map(wn => (
                    <div key={wn} className="flex items-center gap-1 text-xs text-brand-danger">
                      <AlertTriangle className="w-3.5 h-3.5" />
                      {wn}
                    </div>
                  ))}

                  <div className="flex flex-wrap items-center gap-2 pt-1">
                    <label className="text-xs text-brand-text-muted" htmlFor={`h4a-team-${key}`}>Mannschaft</label>
                    <select
                      id={`h4a-team-${key}`}
                      className={SELECT_SM}
                      value={teamId ?? ''}
                      onChange={e => onTeamChange(key, Number(e.target.value))}
                    >
                      <option value="">— zuordnen —</option>
                      {teams.map(tm => (
                        <option key={tm.id} value={tm.id}>{tm.name}</option>
                      ))}
                    </select>

                    <label className="text-xs text-brand-text-muted" htmlFor={`h4a-tpl-${key}`}>Dienst-Vorlage</label>
                    <select
                      id={`h4a-tpl-${key}`}
                      className={SELECT_SM}
                      value={templateId ?? ''}
                      onChange={e => onTemplateChange(key, e.target.value === '' ? null : Number(e.target.value))}
                    >
                      <option value="">Keine</option>
                      {templates.map(tpl => (
                        <option key={tpl.id} value={tpl.id}>{tpl.name}</option>
                      ))}
                    </select>

                    {!unresolved && (
                      teamOverrides[key] === undefined && g.team_source === 'vorschlag' ? (
                        <span
                          className="flex items-center gap-1 text-xs text-brand-text"
                          title={`Aus Staffel ${g.staffel} abgeleitet — bitte prüfen`}
                        >
                          <Lightbulb className="w-3.5 h-3.5" />
                          Vorschlag: {teams.find(tm => tm.id === teamId)?.name ?? g.team_name}
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-xs text-brand-text-muted">
                          <Check className="w-3.5 h-3.5" />
                          {teams.find(tm => tm.id === teamId)?.name ?? g.team_name}
                        </span>
                      )
                    )}
                  </div>
                </div>
              </div>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
