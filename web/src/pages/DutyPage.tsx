import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Home, Plane, Calendar, UserCheck, History, Filter } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import EventTypeFilter, { type EventTypeFilterEntry } from '../components/EventTypeFilter'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'
import { getEventColors } from '../lib/eventColors'
import { buildTeamShortNames } from '../lib/teamName'
import DutySlotList, { BoardSlot } from '../components/DutySlotList'

interface BoardGroup {
  game_id: number | null
  team_id: number | null
  date: string | null
  event_time: string | null
  opponent: string | null
  event_type: string | null
  team_name: string
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
  return { team, types, mine, past, audienceAll }
}

export default function DutyPage() {
  const { user, hasCapability } = useAuth()
  // Slot-Verwaltung (Bearbeiten/Löschen) = manage_duties (admin/vorstand/trainer/
  // sportliche_leitung) — deckungsgleich mit dem Backend-Gate der duty-slots-Routen.
  // Vorstand ist hier bewusst eingeschlossen (Dienste wie Kasse/Einkauf).
  const canManageDuties = hasCapability('manage_duties')

  const [searchParams, setSearchParams] = useSearchParams()
  const { team: filterTeamId, types: filterTypes, mine: viewMine, past: showPast, audienceAll } = parseFilters(searchParams)
  const showAudiencePill = AUDIENCE_FILTER_FUNCTIONS.some(f => user?.clubFunctions?.includes(f))

  const [groups, setGroups] = useState<BoardGroup[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const teamShortNames = useMemo(() => buildTeamShortNames(teams), [teams])
  const [proxyChildren, setProxyChildren] = useState<ProxyChild[]>([])
  const compact = useCompactHeader(950)
  const DUTY_TYPES: EventTypeFilterEntry[] = [
    ['heim',      'Heim',      <Home className="w-3.5 h-3.5" />],
    ['auswärts',  'Auswärts',  <Plane className="w-3.5 h-3.5" />],
    ['generisch', 'Sonstiges', <Calendar className="w-3.5 h-3.5" />],
  ]

  const updateFilter = (patch: { team?: number | null; types?: Set<string>; mine?: boolean; past?: boolean; audienceAll?: boolean }) => {
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
    setSearchParams(next, { replace: true })
  }

  const toggleType = (type: string) => {
    const next = new Set(filterTypes)
    if (next.has(type)) next.delete(type); else next.add(type)
    updateFilter({ types: next })
  }

  const load = () => {
    const params = new URLSearchParams()
    if (viewMine) params.set('view', 'mine')
    if (audienceAll) params.set('audience', 'all')
    const qs = params.toString()
    const url = qs ? `/duty-board?${qs}` : '/duty-board'
    api.get(url).then(r => setGroups(r.data ?? []))
  }

  // load kapselt viewMine/audienceAll, soll nur bei deren Änderung neu laufen
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [viewMine, audienceAll])
  useLiveUpdates((event) => { if (event === 'duties') load() })

  useEffect(() => {
    api.get('/teams')
      .then(r => setTeams(Array.isArray(r.data) ? r.data : (r.data?.teams ?? [])))
      .catch(() => {})
    api.get('/family/proxy-accounts')
      .then(r => setProxyChildren(r.data ?? []))
      .catch(() => setProxyChildren([]))
  }, [])

  const visibleGroups = groups.filter(g => {
    if (!showPast && g.past) return false
    const eventType = g.event_type ?? 'generisch'
    if (!filterTypes.has(eventType)) return false
    if (filterTeamId !== null && g.team_id !== filterTeamId) return false
    return true
  })

  const noTypesActive = filterTypes.size === 0

  return (
    <div>
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold text-brand-text shrink-0">Dienste</h1>
        <div className="flex items-center gap-1.5 flex-1 flex-nowrap min-w-0">
          {teams.length > 1 && (
            <select
              value={filterTeamId ?? ''}
              onChange={e => updateFilter({ team: e.target.value === '' ? null : Number(e.target.value) })}
              className="border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-24 shrink-0"
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
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <button
            onClick={() => updateFilter({ mine: !viewMine })}
            aria-label="Meine"
            className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors ${compact ? 'px-2' : 'px-3'} ${
              viewMine
                ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
            }`}
          >
            <UserCheck className="w-3.5 h-3.5" />
            {!compact && <span>Meine</span>}
          </button>
          <button
            onClick={() => updateFilter({ past: !showPast })}
            aria-label="Vergangene anzeigen"
            className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors ${compact ? 'px-2' : 'px-3'} ${
              showPast
                ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
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
              className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors ${compact ? 'px-2' : 'px-3'} ${
                !audienceAll
                  ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                  : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
              }`}
            >
              <Filter className="w-3.5 h-3.5" />
              {!compact && <span>Nur Audience</span>}
            </button>
          )}
        </div>
      </div>

      {visibleGroups.length === 0 && (
        <p className="text-brand-text-muted">
          {noTypesActive
            ? 'Kein Event-Typ ausgewählt — bitte mindestens eine Pill aktivieren.'
            : groups.length === 0
              ? 'Keine Dienste für deine Mannschaften.'
              : viewMine
                ? 'Du hast keine Dienste übernommen.'
                : 'Keine Dienste passen zum aktuellen Filter.'}
        </p>
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
                <span className="text-xs text-brand-text-muted font-medium">{g.team_name}</span>
              </div>

              <DutySlotList
                slots={g.slots}
                isPast={g.past}
                canEdit={canManageDuties}
                onReload={load}
                proxyChildren={proxyChildren}
              />
            </div>
          )
        })}
      </div>
    </div>
  )
}
