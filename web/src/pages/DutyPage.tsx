import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Home, Plane, Calendar, UserCheck, History, Users } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import EventTypeFilter, { type EventTypeFilterEntry } from '../components/EventTypeFilter'
import EventSearchInput from '../components/EventSearchInput'
import FilterEmptyState from '../components/FilterEmptyState'
import { useDebouncedQueryParam } from '../hooks/useDebouncedQueryParam'
import { parseQuery, matchesQuery } from '../lib/eventFilter'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'
import { getEventColors } from '../lib/eventColors'
import { buildTeamShortNames } from '../lib/teamName'
import { HEADER_CTRL, HEADER_CTRL_ICON, HEADER_FIELD, HEADER_NEUTRAL, HEADER_PRIMARY } from '../lib/buttonStyles'
import DutySlotList, { BoardSlot } from '../components/DutySlotList'

interface BoardGroup {
  game_id: number | null
  team_ids: number[]
  team_names: string[]
  date: string | null
  event_time: string | null
  opponent: string | null
  event_type: string | null
  venue?: string
  label: string | null
  past: boolean
  slots: BoardSlot[]
}

interface Team {
  id: number
  name: string
  age_class: string
  gender: string
  team_number: number
  group_count: number
  is_active: boolean
}

export interface ProxyChild {
  user_id: number
  member_id: number
  name: string
}

const WEEKDAYS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']

// Adapter für den Textfilter: welche Felder einer Board-Gruppe durchsucht
// werden. Diensttyp und zugewiesene Person sind die beiden Felder, die es nur
// hier gibt — deshalb keine gemeinsame Feldmenge über die drei Seiten
// (openspec/changes/termin-textfilter/design.md §6).
function groupFilterFields(g: BoardGroup): (string | null | undefined)[] {
  const fields: (string | null | undefined)[] = [g.opponent, g.venue, g.label, ...g.team_names]
  for (const s of g.slots) {
    fields.push(s.duty_type, s.role_desc)
    for (const a of s.assignees ?? []) fields.push(a.name)
  }
  return fields
}

function formatDate(iso: string): string {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]} ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.`
}

const ALL_TYPES = new Set(['heim', 'auswärts', 'generisch'])
const AUDIENCE_FILTER_FUNCTIONS = ['vorstand', 'vorstand_beisitzer', 'trainer', 'sportliche_leitung']

function parseFilters(sp: URLSearchParams) {
  const team = parseInt(sp.get('team') ?? '') || null
  const typesRaw = sp.get('types')
  const types = typesRaw
    ? (() => {
        const parsed = new Set(typesRaw.split(',').filter(t => ALL_TYPES.has(t)))
        return parsed.size > 0 ? parsed : new Set(ALL_TYPES)
      })()
    : new Set(ALL_TYPES)
  const mine = sp.get('mine') === '1'
  const past = sp.get('past') === '1'
  const audienceAll = sp.get('audience') === 'all'
  const focusRaw = sp.get('focus')
  const focusMatch = focusRaw?.match(/^(slot|game)-(\d+)$/)
  const focus = focusMatch ? { kind: focusMatch[1] as 'slot' | 'game', id: parseInt(focusMatch[2]) } : null
  return { team, types, mine, past, audienceAll, focus }
}

export default function DutyPage() {
  const { user, hasCapability } = useAuth()
  // Slot-Verwaltung (Bearbeiten/Löschen) = manage_duties (admin/vorstand/trainer/
  // sportliche_leitung) — deckungsgleich mit dem Backend-Gate der duty-slots-Routen.
  // Vorstand ist hier bewusst eingeschlossen (Dienste wie Kasse/Einkauf).
  const canManageDuties = hasCapability('manage_duties')

  const [searchParams, setSearchParams] = useSearchParams()
  const { team: filterTeamId, types: filterTypes, mine: viewMine, past: showPast, audienceAll, focus } = parseFilters(searchParams)
  const showAudiencePill = AUDIENCE_FILTER_FUNCTIONS.some(f => user?.clubFunctions?.includes(f))

  // q lebt getrennt von parseFilters: die Filterung wirkt sofort, die URL zieht
  // verzögert nach (design.md §9).
  const [query, setQuery] = useDebouncedQueryParam('q')
  const queryTokens = useMemo(() => parseQuery(query), [query])

  const [groups, setGroups] = useState<BoardGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [teams, setTeams] = useState<Team[]>([])
  const teamShortNames = useMemo(() => buildTeamShortNames(teams), [teams])
  const [proxyChildren, setProxyChildren] = useState<ProxyChild[]>([])
  const compact = useCompactHeader(950)
  const DUTY_TYPES: EventTypeFilterEntry[] = [
    ['heim',      'Heim',      <Home className="w-3.5 h-3.5" />],
    ['auswärts',  'Auswärts',  <Plane className="w-3.5 h-3.5" />],
    ['generisch', 'Sonstiges', <Calendar className="w-3.5 h-3.5" />],
  ]

  const updateFilter = (patch: { team?: number | null; types?: Set<string>; mine?: boolean; past?: boolean; audienceAll?: boolean; focus?: { kind: 'slot' | 'game'; id: number } | null }) => {
    const next = new URLSearchParams(searchParams)
    if ('team' in patch) {
      if (patch.team === null) next.delete('team')
      else next.set('team', String(patch.team))
    }
    if ('types' in patch && patch.types) {
      const isDefault = patch.types.size === ALL_TYPES.size && [...ALL_TYPES].every(t => patch.types!.has(t))
      if (isDefault) next.delete('types')
      else next.set('types', [...patch.types].join(','))
    }
    if ('mine' in patch) {
      if (patch.mine) next.set('mine', '1')
      else next.delete('mine')
    }
    if ('past' in patch) {
      if (patch.past) next.set('past', '1')
      else next.delete('past')
    }
    if ('audienceAll' in patch) {
      if (patch.audienceAll) next.set('audience', 'all')
      else next.delete('audience')
    }
    if ('focus' in patch) {
      if (patch.focus) next.set('focus', `${patch.focus.kind}-${patch.focus.id}`)
      else next.delete('focus')
    }
    setSearchParams(next, { replace: true })
  }

  const toggleType = (type: string) => {
    const next = new Set(filterTypes)
    if (next.has(type)) next.delete(type); else next.add(type)
    updateFilter({ types: next })
  }

  // Wird von DutySlotList vor der Navigation zur Anleitungsseite aufgerufen: hinterlegt
  // den Fokus-Marker auf der aktuellen /dienste-URL, damit „Zurück" später zu dieser
  // Zeile scrollt (siehe openspec/changes/zurueck-position-wiederherstellen).
  const handleFocusSlot = (slotId: number) => {
    updateFilter({ focus: { kind: 'slot', id: slotId } })
  }

  const load = () => {
    setLoading(true)
    const params = new URLSearchParams()
    if (viewMine) params.set('view', 'mine')
    if (audienceAll) params.set('audience', 'all')
    // Ohne „Vergangene": Datumsfenster serverseitig ab heute — spart die
    // komplette Historie in der Payload. Mit Toggle wird alles geladen.
    if (!showPast) {
      const d = new Date()
      params.set('from', `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`)
    }
    const qs = params.toString()
    const url = qs ? `/duty-board?${qs}` : '/duty-board'
    api.get(url).then(r => setGroups(r.data ?? [])).finally(() => setLoading(false))
  }

  // load kapselt viewMine/audienceAll/showPast, soll nur bei deren Änderung neu laufen
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [viewMine, audienceAll, showPast])
  useLiveUpdates((event) => { if (event === 'duties') load() })

  useEffect(() => {
    api.get('/teams')
      .then(r => setTeams(Array.isArray(r.data) ? r.data : (r.data?.teams ?? [])))
      .catch(() => {})
    api.get('/family/proxy-accounts')
      .then(r => setProxyChildren(r.data ?? []))
      .catch(() => setProxyChildren([]))
  }, [])

  // focus kann entweder einen einzelnen Slot (Anleitungs-Link) oder eine ganze
  // Spiel-Gruppe (z. B. „In Diensten öffnen" aus dem Kalender-Modal) markieren.
  const groupMatchesFocus = (g: BoardGroup) => {
    if (!focus) return false
    if (focus.kind === 'slot') return g.slots.some(s => s.id === focus.id)
    return g.game_id === focus.id
  }

  const visibleGroups = groups.filter(g => {
    // Die fokussierte Gruppe bleibt sichtbar, auch wenn sie sonst durch Typ-/
    // Team-Filter herausfiele — analog zu TerminePage.tsx, das die fokussierte
    // Karte ebenfalls unconditional durchlässt. Das serverseitige Datumsfenster
    // (past) kann clientseitig nicht nachgeholt werden — eine fokussierte Gruppe
    // außerhalb des geladenen Zeitraums bleibt entsprechend über `focusNotFound`
    // sichtbar statt hier stillschweigend zu fehlen.
    if (groupMatchesFocus(g)) return true
    if (!showPast && g.past) return false
    const eventType = g.event_type ?? 'generisch'
    if (!filterTypes.has(eventType)) return false
    if (filterTeamId !== null && !g.team_ids.includes(filterTeamId)) return false
    // Textfilter zuletzt: er ist das teuerste Prädikat (String-Normalisierung
    // über mehrere Felder), die billigen Set-Lookups schneiden vorher weg.
    if (!matchesQuery(queryTokens, groupFilterFields(g), [g.date])) return false
    return true
  })

  // Wie viele Gruppen der Textfilter treffen würde, wenn kein anderer Filter
  // aktiv wäre. Nur nötig, wenn nichts sichtbar ist — sonst gar nicht gerechnet.
  // `showPast` zählt bewusst nicht als „anderer Filter": ohne den Toggle lädt
  // der Server die Vergangenheit gar nicht erst, ein Treffer dort wäre auch in
  // diesem zweiten Durchlauf nicht auffindbar.
  const otherFiltersActive =
    filterTeamId !== null || filterTypes.size !== ALL_TYPES.size || viewMine
  const hiddenByOtherFilters =
    visibleGroups.length === 0 && queryTokens.length > 0 && otherFiltersActive
      ? groups.filter(g => matchesQuery(queryTokens, groupFilterFields(g), [g.date])).length
      : 0

  const resetFilters = () => updateFilter({ team: null, types: new Set(ALL_TYPES), mine: false })

  const noTypesActive = filterTypes.size === 0

  // Ziel existiert nicht (mehr) in den geladenen Daten — z. B. gelöscht, oder
  // außerhalb des serverseitig geladenen Zeitraums (past-Toggle aus). Anders als
  // TerminePage.tsx gibt es hier bewusst KEINE automatische Filter-Erweiterung:
  // der Fokus wird ausschließlich durch einen Klick auf eine bereits sichtbare
  // Zeile/ein bereits sichtbares Spiel gesetzt (kein Push-Notification-Deep-Link
  // wie bei Termine), die Filter (inkl. past/mine/audience) sind beim
  // Zurücknavigieren also unverändert dieselben, unter denen das Ziel zuvor
  // sichtbar war — ein Mismatch ist damit nur durch zwischenzeitliche
  // Löschung/Live-Update zu erwarten, nicht durch die Filter selbst. Ein
  // einfacher Hinweis genügt dafür.
  const focusNotFound = !loading && focus !== null && !groups.some(groupMatchesFocus)

  useEffect(() => {
    if (!focus || loading) return
    const elId = focus.kind === 'slot' ? `duty-slot-${focus.id}` : `duty-game-${focus.id}`
    const el = document.getElementById(elId)
    if (!el) return
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    el.classList.add('ring-2', 'ring-brand-yellow', 'transition-all')
    const t = setTimeout(() => el.classList.remove('ring-2', 'ring-brand-yellow'), 2000)
    return () => clearTimeout(t)
    // focus.kind/id als Primitives sind die minimale Dependency (focus selbst ist
    // ein bei jedem Render neu erzeugtes Objekt); die Slot-Gesamtzahl steht
    // stellvertretend dafür, dass neue Zeilen ins DOM kamen (z. B. nach dem
    // initialen Laden), analog zu visibleTermine.length in TerminePage.tsx.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focus?.kind, focus?.id, loading, groups.reduce((acc, g) => acc + g.slots.length, 0)])

  return (
    <div>
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold text-brand-text shrink-0">Dienste</h1>
        <div className="flex items-center gap-1.5 flex-1 flex-nowrap min-w-0">
          {/* `hidden sm:block`: siehe TerminePage — auf Mobile fehlt neben Typ-Filter
              und Suchfeld der Platz. */}
          {teams.length > 1 && (
            <select
              value={filterTeamId ?? ''}
              onChange={e => updateFilter({ team: e.target.value === '' ? null : Number(e.target.value) })}
              className={`${HEADER_FIELD} hidden sm:block w-24 shrink-0`}
            >
              <option value="">Teams</option>
              {teams.map(t => (
                <option key={t.id} value={t.id}>{teamShortNames.get(t.id) ?? t.name}</option>
              ))}
            </select>
          )}
          <EventTypeFilter
            types={DUTY_TYPES}
            active={filterTypes}
            onToggle={toggleType}
            compact={compact}
            ariaLabel="Dienst-Typ-Filter"
          />
          <EventSearchInput
            value={query}
            onChange={setQuery}
            compact={compact}
            placeholder="Gegner, Ort, Dienst, Person…"
            ariaLabel="Dienste filtern"
          />
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <button
            onClick={() => updateFilter({ mine: !viewMine })}
            aria-label="Meine"
            className={`${compact ? HEADER_CTRL_ICON : HEADER_CTRL} ${
              viewMine ? HEADER_PRIMARY : HEADER_NEUTRAL
            }`}
          >
            <UserCheck className="w-3.5 h-3.5" />
            {!compact && <span>Meine</span>}
          </button>
          <button
            onClick={() => updateFilter({ past: !showPast })}
            aria-label="Vergangene anzeigen"
            className={`${compact ? HEADER_CTRL_ICON : HEADER_CTRL} ${
              showPast ? HEADER_PRIMARY : HEADER_NEUTRAL
            }`}
          >
            <History className="w-3.5 h-3.5" />
            {!compact && <span>Vergangene</span>}
          </button>
          {showAudiencePill && (
            <button
              onClick={() => updateFilter({ audienceAll: !audienceAll })}
              aria-label="Nur meine Audience"
              title={audienceAll ? 'Alle Audiences sichtbar — klicken für Filter auf meine Audience' : 'Nur meine Audience — klicken für alle Audiences'}
              className={`${compact ? HEADER_CTRL_ICON : HEADER_CTRL} ${
                !audienceAll ? HEADER_PRIMARY : HEADER_NEUTRAL
              }`}
            >
              <Users className="w-3.5 h-3.5" />
              {!compact && <span>Nur Audience</span>}
            </button>
          )}
        </div>
      </div>

      {focusNotFound && (
        <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
          Dieser Dienst ist nicht verfügbar.
        </div>
      )}

      {visibleGroups.length === 0 && (
        hiddenByOtherFilters > 0 ? (
          <FilterEmptyState hiddenByOtherFilters={hiddenByOtherFilters} onResetFilters={resetFilters} />
        ) : (
          <p className="text-brand-text-muted">
            {noTypesActive
              ? 'Kein Event-Typ ausgewählt — bitte mindestens eine Pill aktivieren.'
              : groups.length === 0
                ? 'Keine Dienste für deine Mannschaften.'
                : query !== ''
                  ? 'Keine Dienste passen zum Filter.'
                  : viewMine
                    ? 'Du hast keine Dienste übernommen.'
                    : 'Keine Dienste passen zum aktuellen Filter.'}
          </p>
        )
      )}

      <div className="space-y-4">
        {visibleGroups.map((g, i) => {
          const colors = getEventColors(g.event_type ?? 'generisch')
          const cardClass = g.past
            ? 'bg-brand-surface-card border-brand-border opacity-60'
            : `${colors.card.bg} ${colors.card.border}`
          const EventIcon = g.event_type === 'heim' ? Home : g.event_type === 'auswärts' ? Plane : Calendar
          return (
            <div
              key={i}
              id={g.game_id ? `duty-game-${g.game_id}` : undefined}
              className={`rounded-xl shadow border-t-4 overflow-hidden ${cardClass}`}
            >
              <div className="px-4 py-3 border-b border-brand-border-subtle flex items-center justify-between">
                <div className="flex items-center gap-3">
                  {g.game_id && (
                    <EventIcon className={`w-5 h-5 shrink-0 ${g.past ? 'text-brand-text-muted' : colors.card.icon}`} />
                  )}
                  <div>
                  {g.game_id ? (
                    <span className="font-semibold text-sm text-brand-text">
                      {g.date ? formatDate(g.date) : ''}
                      {g.event_time ? ` · ${g.event_time} Uhr` : ''}
                      {g.opponent ? ` · ${g.event_type === 'generisch' ? g.opponent : `Team vs ${g.opponent}`}` : ''}
                    </span>
                  ) : (
                    <span className="font-semibold text-sm text-brand-text">
                      {g.date ? formatDate(g.date) : ''}{g.label ? ` · ${g.label}` : ''}
                    </span>
                  )}
                  </div>
                </div>
                <span className="text-xs text-brand-text-muted font-medium">{g.team_names.join(', ')}</span>
              </div>

              <DutySlotList
                slots={g.slots}
                isPast={g.past}
                canEdit={canManageDuties}
                onReload={load}
                proxyChildren={proxyChildren}
                onFocusSlot={handleFocusSlot}
              />
            </div>
          )
        })}
      </div>
    </div>
  )
}
