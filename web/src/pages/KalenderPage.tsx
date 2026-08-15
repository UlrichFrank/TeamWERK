import { useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Home, Plane, Calendar, CalendarClock, ChevronDown, ChevronLeft, ChevronRight, Plus, Dumbbell, RefreshCw, Check, X, AlertTriangle, Download, UserX } from 'lucide-react'
import { api } from '../lib/api'
import { buildPreviewUrl } from '../lib/dutyPreview'
import { getEventColors } from '../lib/eventColors'
import { buildTeamShortNames, formatTeamList, TeamForName } from '../lib/teamName'
import { errorStatus } from '../lib/errors'
import { useAuth } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'
import { useDebouncedQueryParam } from '../hooks/useDebouncedQueryParam'
import EventSearchInput from '../components/EventSearchInput'
import { parseQuery, matchesQuery } from '../lib/eventFilter'

import TrainingEditModal from '../components/TrainingEditModal'
import GameEditModal from '../components/GameEditModal'
import EventInfoModal from '../components/EventInfoModal'
import SpieltagDetailModal from '../components/SpieltagDetailModal'
import VenuePicker, { Venue as VenueType } from '../components/VenuePicker'
import RsvpDefaultsEditor, { type RsvpDefault } from '../components/RsvpDefaultsEditor'
import RegenSummaryCard, { RegenSummary } from '../components/RegenSummaryCard'
import H4AImportModal from '../components/H4AImportModal'
import DutyBulkRegenModal, { BulkRegenResult } from '../components/DutyBulkRegenModal'
import {
  GameDayHostSelect,
  GameDayHostPreviewDialog,
  useAusrichterOptions,
  fetchGameDayHost,
  previewGameDayHost,
  applyGameDayHost,
  describeHostError,
  type GameDayHost,
} from '../components/GameDayHostPicker'

interface VenueRef {
  id: number
  name: string
  street: string
  city: string
  postal_code: string
  note: string
}

interface Training {
  id: number
  title: string
  date: string
  start_time: string
  end_time: string
  team_name?: string
  venue?: VenueRef | null
  status: 'active' | 'cancelled'
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
  my_rsvp_locked?: boolean
  series_id?: number
  team_id: number
  season_id: number
  note: string
  cancel_reason?: string
  rsvp_default_players?: RsvpDefault
  rsvp_default_extended?: RsvpDefault
  rsvp_require_reason?: number
}

interface Game {
  id: number
  date: string
  time: string
  end_time?: string | null
  end_date?: string | null
  opponent: string
  teams: Array<{ id: number; name: string; display_short?: string; display_long?: string }>
  event_type: string
  template_id?: number | null
  slot_count: number
  filled_count: number
  total_count: number
  confirmed_count: number
  declined_count: number
  maybe_count: number
  venue?: VenueRef | null
  rsvp_default_players?: RsvpDefault
  rsvp_default_extended?: RsvpDefault
  rsvp_require_reason?: number
  note?: string
}

// Adapter für den Textfilter (openspec/changes/termin-textfilter/design.md §6).
// Die drei Termin-Arten des Kalenders tragen verschiedene Felder.
function gameFilterFields(g: Game): (string | null | undefined)[] {
  return [g.opponent, g.venue?.name, g.venue?.city, g.note, ...g.teams.flatMap(t => [t.name, t.display_short, t.display_long])]
}

function trainingFilterFields(t: Training): (string | null | undefined)[] {
  return [t.title, t.venue?.name, t.venue?.city, t.team_name, t.note]
}

const ABSENCE_TYPE_LABELS: Record<string, string> = { vacation: 'Urlaub', injury: 'Verletzung' }

function absenceFilterFields(a: Absence): (string | null | undefined)[] {
  return [a.member_name, a.note, ABSENCE_TYPE_LABELS[a.type]]
}

interface Absence {
  id: number
  member_id: number
  member_name: string
  can_edit: boolean
  type: 'vacation' | 'injury'
  start_date: string
  end_date: string
  note: string
  created_by: number
  is_own: boolean
}

interface SlotPreview {
  duty_type_id: number
  duty_type_name: string
  event_time: string
  slots_count: number
  role_desc: string
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

const WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']
const MONTHS = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember']

function padDate(year: number, month: number, day: number): string {
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

const BTN_SECONDARY = 'border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors'
const INPUT_WIZ = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function KalenderPage() {
  const { user, hasCapability } = useAuth()
  // Server-computed permissions (see policy.Capabilities). Event management = manage_games
  // (admin/vorstand/trainer/sportliche_leitung); training creation = manage_trainings
  // (admin/trainer/sportliche_leitung, no pure vorstand).
  const canEdit = hasCapability('manage_games')
  // Enger als canEdit: der H4A-Import nimmt fremde Zugangsdaten entgegen und bleibt
  // beim Vorstand, obwohl Trainer/sportliche Leitung Spiele pflegen dürfen.
  const canImportGames = hasCapability('import_games')
  // Ebenfalls enger als canEdit: ein Massenlauf kann hunderte Dienst-Slots
  // löschen/neu anlegen — bleibt deshalb wie der H4A-Import beim Vorstand.
  const canBulkRegenDuties = hasCapability('bulk_regen_duties')
  const canSeeTeamAbsences = canEdit
  const canManageTrainings = hasCapability('manage_trainings')
  const [searchParams, setSearchParams] = useSearchParams()
  const now = new Date()
  const initDate = () => {
    const param = searchParams.get('date')
    if (param) {
      const d = new Date(param + 'T12:00:00')
      if (!isNaN(d.getTime())) return d
    }
    return now
  }
  const startDate = initDate()
  const [year, setYear] = useState(startDate.getFullYear())
  const [month, setMonth] = useState(startDate.getMonth())
  const [games, setGames] = useState<Game[]>([])
  const [trainings, setTrainings] = useState<Training[]>([])
  const [absences, setAbsences] = useState<Absence[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const [allTeamNames, setAllTeamNames] = useState<TeamForName[]>([])
  // q lebt — anders als die übrigen Kalender-Filter — in der URL, damit ein
  // gefilterter Kalender teilbar ist. Die bestehende Inkonsistenz der anderen
  // Filter (lokaler State) wird hier bewusst nicht mitrepariert.
  //
  // Bewusst OHNE den Ausgeblendet-Zähler der Listenseiten (design.md §7): das
  // Monatsgitter hat keine Leermeldung, in die er passte, und ein leeres Gitter
  // ist die normale Anzeige eines Monats ohne Termine — kein Signal, das eine
  // Erklärung bräuchte. Das ist eine Auslassung, kein Vergessen.
  const [query, setQuery] = useDebouncedQueryParam('q')
  const queryTokens = useMemo(() => parseQuery(query), [query])
  const [filterTeamId, setFilterTeamId] = useState<number | null>(null)
  const [filterTypes, setFilterTypes] = useState<Set<string>>(new Set(['heim', 'auswärts', 'generisch', 'training']))
  const [showTeamAbsences, setShowTeamAbsences] = useState<boolean>(
    () => sessionStorage.getItem('kalender_show_team_absences') === 'true'
  )
  const compact = useCompactHeader(950)

  const [regenSummary, setRegenSummary] = useState<RegenSummary | null>(null)
  const [showH4AImport, setShowH4AImport] = useState(false)
  const [showBulkRegen, setShowBulkRegen] = useState(false)
  const [showEventMenu, setShowEventMenu] = useState(false)
  const [importResult, setImportResult] = useState<{ imported: number; updated: number; skipped: number } | null>(null)
  const [bulkRegenResult, setBulkRegenResult] = useState<BulkRegenResult | null>(null)

  // Wizard dialog
  const [showCreate, setShowCreate] = useState(false)
  const [wizardStep, setWizardStep] = useState(1)
  const [eventType, setEventType] = useState<'heim' | 'auswärts' | 'generisch' | 'training' | 'serie' | 'abwesenheit' | ''>('')
  const [selectedDate, setSelectedDate] = useState('')
  const [selectedTime, setSelectedTime] = useState('15:00')
  const [selectedOpponent, setSelectedOpponent] = useState('')
  const [selectedTeamIds, setSelectedTeamIds] = useState<number[]>([])
  const [selectedEndTime, setSelectedEndTime] = useState('16:00')
  const [selectedEndDate, setSelectedEndDate] = useState('')
  const [selectedTemplate, setSelectedTemplate] = useState<number | null>(null)
  const [templates, setTemplates] = useState<{ id: number; name: string; template_type: string; duration_minutes?: number }[]>([])
  const [preview, setPreview] = useState<SlotPreview[]>([])
  const [selectedSlotIndices, setSelectedSlotIndices] = useState<Set<number>>(new Set())
  const [previewLoading, setPreviewLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  // Training / Serie wizard states
  const [activeSeasonId, setActiveSeasonId] = useState(0)
  const [trainingTitle, setTrainingTitle] = useState('')
  const [trainingStartTime, setTrainingStartTime] = useState('18:00')
  const [trainingEndTime, setTrainingEndTime] = useState('19:30')
  const [trainingVenueId, setTrainingVenueId] = useState<number | null>(null)
  const [selectedVenueId, setSelectedVenueId] = useState<number | null>(null)
  const [allVenues, setAllVenues] = useState<VenueType[]>([])
  const [seriesWeekday, setSeriesWeekday] = useState(1)
  const [seriesValidFrom, setSeriesValidFrom] = useState('')
  const [seriesValidUntil, setSeriesValidUntil] = useState('')
  const [gameDefaultPlayers, setGameDefaultPlayers] = useState<RsvpDefault>('none')
  const [gameDefaultExtended, setGameDefaultExtended] = useState<RsvpDefault>('none')
  const [gameRsvpRequireReason, setGameRsvpRequireReason] = useState(1)
  // Absence wizard states
  const [absenceForm, setAbsenceForm] = useState<{ member_ids: number[]; type: string; start_date: string; end_date: string; note: string }>({ member_ids: [], type: 'vacation', start_date: '', end_date: '', note: '' })
  const [absencePreviewEvents, setAbsencePreviewEvents] = useState<Array<{ event_type: string; event_id: number; name: string; date: string; pending: boolean }> | null>(null)
  const [absencePreviewLoading, setAbsencePreviewLoading] = useState(false)
  const [absenceChildren, setAbsenceChildren] = useState<Array<{ id: number; name: string }>>([])
  const [absenceSaving, setAbsenceSaving] = useState(false)
  const [absenceError, setAbsenceError] = useState('')
  // Ausrichter des gewählten Spieltags im Wizard (heimspieltag-ausrichter,
  // design.md Decision 9). Der Wert ist tagesbezogen und wird deshalb NICHT mit
  // dem Termin angelegt, sondern — falls er vom geltenden abweicht — vor dem
  // Anlegen über dieselbe Vorschau geschrieben wie im Detail-Modal.
  const ausrichterOptions = useAusrichterOptions(canEdit)
  const [wizardHost, setWizardHost] = useState<GameDayHost | null>(null)
  const [wizardHostId, setWizardHostId] = useState<number | null>(null)
  const [hostPreview, setHostPreview] = useState<GameDayHost | null>(null)
  const [hostBusy, setHostBusy] = useState(false)
  const [hostError, setHostError] = useState<string | null>(null)
  // Inline edit modal
  const [editingTraining, setEditingTraining] = useState<Training | null>(null)
  const [editingGame, setEditingGame] = useState<Game | null>(null)
  const [detailGameId, setDetailGameId] = useState<number | null>(null)
  const [infoItem, setInfoItem] = useState<{ type: 'game' | 'training' | 'absence'; game?: Game; training?: Training; absence?: Absence } | null>(null)


  const loadGames = async () => {
    try {
      // Kalender zeigt einen ganzen Monat → großzügiges Limit, damit keine
      // Spiele fehlen. Antwort ist {items,total}.
      const r = await api.get('/games?limit=500')
      const data = r.data
      const payload = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
      setGames(payload)
      return payload
    } catch {
      setGames([])
      return []
    }
  }

  const loadTrainings = async () => {
    try {
      const from = `${year}-${String(month + 1).padStart(2, '0')}-01`
      const lastDay = new Date(year, month + 1, 0).getDate()
      const to = `${year}-${String(month + 1).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`
      const r = await api.get(`/training-sessions?from=${from}&to=${to}&limit=500`)
      // Antwort ist {items,total}; Kalender braucht alle Termine des Monats.
      const items = Array.isArray(r.data?.items) ? r.data.items : (Array.isArray(r.data) ? r.data : [])
      setTrainings(items)
    } catch {
      setTrainings([])
    }
  }

  const loadAbsences = async (overrideShowTeam?: boolean, overrideTeamId?: number | null) => {
    try {
      const from = `${year}-${String(month + 1).padStart(2, '0')}-01`
      const lastDay = new Date(year, month + 1, 0).getDate()
      const to = `${year}-${String(month + 1).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`
      const show = overrideShowTeam !== undefined ? overrideShowTeam : showTeamAbsences
      const tid = overrideTeamId !== undefined ? overrideTeamId : filterTeamId
      let url = `/absences/calendar?from=${from}&to=${to}`
      if (show && canSeeTeamAbsences) {
        url += '&show_team=true'
        if (tid !== null) url += `&team_id=${tid}`
      }
      const r = await api.get(url)
      setAbsences(Array.isArray(r.data) ? r.data : [])
    } catch {
      setAbsences([])
    }
  }

  const loadAbsenceChildren = async () => {
    try {
      const r = await api.get('/profile/me')
      const kinder: Array<{ id: number; first_name: string; last_name: string }> = r.data?.children ?? []
      setAbsenceChildren(kinder.map(k => ({ id: k.id, name: `${k.first_name} ${k.last_name}` })))
    } catch {}
  }

  useEffect(() => {
    const loadInitialData = async () => {
      await Promise.all([
        loadGames(),
        loadTrainings(),
        loadAbsences(),
        api.get('/teams')
          .then(r => setTeams(Array.isArray(r.data) ? r.data : (r.data?.teams ?? [])))
          .catch(() => setTeams([])),
        api.get('/teams/names')
          .then(r => setAllTeamNames(Array.isArray(r.data) ? r.data : []))
          .catch(() => setAllTeamNames([])),
        api.get('/seasons')
          .then(r => {
            const seasons = Array.isArray(r.data) ? r.data : []
            const active = seasons.find((s: { id: number; is_active: boolean }) => s.is_active)
            if (active) setActiveSeasonId(active.id)
          })
          .catch(() => {}),
        api.get<VenueType[]>('/venues')
          .then(r => setAllVenues(r.data ?? []))
          .catch(() => {}),
      ])
      if (user?.isParent) {
        loadAbsenceChildren()
      }
    }
    loadInitialData()
    // Initial-Load nur beim Mount; load*-Funktionen kapseln year/month/Filter, soll nicht erneut feuern
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
  useEffect(() => { loadTrainings(); loadAbsences() }, [year, month]) // eslint-disable-line react-hooks/exhaustive-deps
  // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
  useEffect(() => { loadAbsences() }, [filterTeamId, showTeamAbsences]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-select the only child once children have loaded — keeps the parent
  // with exactly one linked kid from being forced through a useless selector.
  useEffect(() => {
    if (eventType === 'abwesenheit' && absenceChildren.length === 1 && absenceForm.member_ids.length === 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
      setAbsenceForm(f => ({ ...f, member_ids: [absenceChildren[0].id] }))
    }
  }, [eventType, absenceChildren, absenceForm.member_ids.length])

  // Geltenden Tages-Ausrichter laden, sobald im Wizard ein Heimspiel-Datum
  // feststeht — die Auswahl startet damit auf dem Wert, der ohne Zutun gälte.
  useEffect(() => {
    if (!showCreate || eventType !== 'heim' || !selectedDate || !canEdit) return
    let cancelled = false
    fetchGameDayHost(selectedDate)
      .then(h => {
        if (cancelled) return
        setWizardHost(h)
        setWizardHostId(h.ausrichter_id)
      })
      .catch(() => {
        if (cancelled) return
        setWizardHost(null)
        setWizardHostId(null)
      })
    return () => { cancelled = true }
  }, [showCreate, eventType, selectedDate, canEdit])

  useLiveUpdates((event) => {
    if (event === 'games') loadGames()
    // 'duties' kommt u.a. vom Massenlauf (ApplyBulkRegen) — betrifft nur Dienst-Slots,
    // nicht games-Felder, aber der Kalender zeigt Dienst-Badges pro Termin.
    if (event === 'duties') loadGames()
    if (event === 'absences') loadAbsences()
    if (event === 'trainings') loadTrainings()
    if (event === 'event-note') { loadGames(); loadTrainings() }
  })

  const prevMonth = () => {
    let newMonth = month
    let newYear = year
    if (month === 0) {
      newMonth = 11
      newYear = year - 1
    } else {
      newMonth = month - 1
    }
    setMonth(newMonth)
    setYear(newYear)
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.set('date', padDate(newYear, newMonth, 1))
      return next
    }, { replace: true })
  }

  const nextMonth = () => {
    let newMonth = month
    let newYear = year
    if (month === 11) {
      newMonth = 0
      newYear = year + 1
    } else {
      newMonth = month + 1
    }
    setMonth(newMonth)
    setYear(newYear)
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.set('date', padDate(newYear, newMonth, 1))
      return next
    }, { replace: true })
  }

  const goToToday = () => {
    const today = new Date()
    const newYear = today.getFullYear()
    const newMonth = today.getMonth()
    setYear(newYear)
    setMonth(newMonth)
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      next.set('date', padDate(newYear, newMonth, 1))
      return next
    }, { replace: true })
  }

  const calendarRef = useRef<HTMLDivElement>(null)
  const eventMenuRef = useRef<HTMLDivElement>(null)
  const pointerStart = useRef<{ x: number; y: number; committed: boolean } | null>(null)
  const SWIPE_THRESHOLD = 50

  const setCalendarTransform = (x: number, animated: boolean) => {
    const el = calendarRef.current
    if (!el) return
    el.style.transition = animated ? 'transform 220ms ease-out' : 'none'
    el.style.transform = x === 0 ? '' : `translateX(${x}px)`
  }

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    pointerStart.current = { x: e.clientX, y: e.clientY, committed: false }
  }

  const handlePointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!pointerStart.current) return
    const dx = e.clientX - pointerStart.current.x
    const dy = e.clientY - pointerStart.current.y
    if (!pointerStart.current.committed) {
      if (Math.abs(dx) < 8 && Math.abs(dy) < 8) return
      if (Math.abs(dy) > Math.abs(dx)) { pointerStart.current = null; return }
      pointerStart.current.committed = true
      e.currentTarget.setPointerCapture(e.pointerId)
    }
    setCalendarTransform(dx, false)
  }

  const handlePointerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!pointerStart.current?.committed) { pointerStart.current = null; return }
    const delta = e.clientX - pointerStart.current.x
    pointerStart.current = null
    const width = calendarRef.current?.offsetWidth ?? 400
    if (Math.abs(delta) < SWIPE_THRESHOLD) { setCalendarTransform(0, true); return }
    const isNext = delta < 0
    setCalendarTransform(isNext ? -width : width, true)
    setTimeout(() => {
      setCalendarTransform(isNext ? width : -width, false)
      if (isNext) nextMonth(); else prevMonth()
      requestAnimationFrame(() => requestAnimationFrame(() => setCalendarTransform(0, true)))
    }, 220)
  }

  const handlePointerCancel = () => {
    pointerStart.current = null
    setCalendarTransform(0, true)
  }

  const openWizardWithDate = (dateStr: string) => {
    setSelectedDate(dateStr)
    setShowCreate(true)
    if (!canEdit && canCreateAbsence) {
      setEventType('abwesenheit')
      setAbsenceForm(f => ({ ...f, start_date: dateStr, end_date: dateStr }))
      setWizardStep(2)
      if (user?.isParent && absenceChildren.length === 0) loadAbsenceChildren()
    } else {
      setWizardStep(1)
      loadTemplates()
    }
  }

  const toggleType = (type: string) => {
    setFilterTypes(prev => {
      const next = new Set(prev)
      if (next.has(type)) next.delete(type); else next.add(type)
      return next
    })
  }

  const shortNames = useMemo(() => buildTeamShortNames(allTeamNames), [allTeamNames])

  const safeGames = Array.isArray(games) ? games : []
  const monthStart = `${year}-${String(month + 1).padStart(2, '0')}-01`
  const lastDay = new Date(year, month + 1, 0).getDate()
  const monthEnd = `${year}-${String(month + 1).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`
  const monthGames = safeGames.filter(g => {
    const effectiveEnd = g.end_date ? g.end_date.slice(0, 10) : g.date.slice(0, 10)
    const start = g.date.slice(0, 10)
    if (start > monthEnd || effectiveEnd < monthStart) return false
    if (!filterTypes.has(g.event_type)) return false
    if (filterTeamId !== null && !g.teams.some(t => t.id === filterTeamId)) return false
    if (!matchesQuery(queryTokens, gameFilterFields(g), [g.date])) return false
    return true
  })

  const gamesByDate: Record<string, Game[]> = {}
  for (const g of monthGames) {
    const start = g.date.slice(0, 10)
    const end = g.end_date ? g.end_date.slice(0, 10) : start
    const cur = new Date(start + 'T12:00:00')
    const endDate = new Date(end + 'T12:00:00')
    while (cur <= endDate) {
      const key = cur.toISOString().slice(0, 10)
      if (key >= monthStart && key <= monthEnd) {
        if (!gamesByDate[key]) gamesByDate[key] = []
        gamesByDate[key].push(g)
      }
      cur.setDate(cur.getDate() + 1)
    }
  }

  const filteredTrainings = trainings.filter(t => {
    if (!filterTypes.has('training')) return false
    if (filterTeamId !== null && t.team_id !== filterTeamId) return false
    if (!matchesQuery(queryTokens, trainingFilterFields(t), [t.date])) return false
    return true
  })

  const trainingsByDate: Record<string, Training[]> = {}
  for (const t of filteredTrainings) {
    const key = t.date.slice(0, 10)
    if (!trainingsByDate[key]) trainingsByDate[key] = []
    trainingsByDate[key].push(t)
  }

  const firstDayOfWeek = (new Date(year, month, 1).getDay() + 6) % 7
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const todayStr = padDate(now.getFullYear(), now.getMonth(), now.getDate())

  // Compute which absences cover each day, and whether they start/end on that day or continue
  const absencesForDay = (dateStr: string): Array<{ absence: Absence; isFirst: boolean; isLast: boolean }> => {
    return absences
      .filter(a => a.start_date <= dateStr && a.end_date >= dateStr)
      // Abwesenheiten werden bisher clientseitig gar nicht gefiltert (Team-Filter
      // läuft serverseitig). Ohne diese Zeile blieben bei aktivem q die
      // Abwesenheits-Balken in Zellen stehen, deren Termine weggefiltert sind —
      // das Gitter sähe schlicht kaputt aus.
      .filter(a => matchesQuery(queryTokens, absenceFilterFields(a), [a.start_date, a.end_date]))
      .map(a => {
        const d = new Date(dateStr + 'T12:00:00')
        const isMonday = d.getDay() === 1
        const isSunday = d.getDay() === 0
        return {
          absence: a,
          isFirst: a.start_date === dateStr || isMonday,
          isLast: a.end_date === dateStr || isSunday,
        }
      })
  }

  // Abweichender Tages-Ausrichter im Wizard: erst die Vorschau (Decision 9), dann
  // schreiben, dann den Termin anlegen. Die Reihenfolge ist wesentlich — der
  // Auto-Regen beim Anlegen liest den Tageswert und würde bei umgekehrter Folge
  // noch mit dem alten Ausrichter rechnen.
  const hostDiffersInWizard = eventType === 'heim' && wizardHost != null
    && wizardHostId != null && wizardHostId !== wizardHost.ausrichter_id

  const confirmCreateGame = async (slots: SlotPreview[]) => {
    if (!hostDiffersInWizard || wizardHostId == null) {
      await doCreateGame(slots)
      return
    }
    setHostBusy(true)
    setHostError(null)
    try {
      setHostPreview(await previewGameDayHost(selectedDate, wizardHostId))
    } catch (e) {
      setCreateError(describeHostError(e))
    } finally {
      setHostBusy(false)
    }
  }

  const applyHostThenCreate = async (slots: SlotPreview[]) => {
    if (wizardHostId == null) return
    setHostBusy(true)
    setHostError(null)
    try {
      const applied = await applyGameDayHost(selectedDate, wizardHostId)
      setWizardHost(applied)
      setHostPreview(null)
      await doCreateGame(slots)
    } catch (e) {
      setHostError(describeHostError(e))
    } finally {
      setHostBusy(false)
    }
  }

  const doCreateGame = async (slots: SlotPreview[]) => {
    setCreating(true)
    setCreateError(null)
    try {
      // For heim/auswärts the backend derives slots from template + adjacency.
      // For generisch the wizard's custom slots are persisted as-is (is_custom=1).
      const slotsPayload = eventType === 'generisch'
        ? slots.map(s => ({
            duty_type_id: s.duty_type_id,
            event_time: s.event_time,
            slots_count: s.slots_count,
            role_desc: s.role_desc,
          }))
        : undefined
      const r = await api.post('/games', {
        date: selectedDate,
        time: selectedTime,
        end_time: eventType === 'generisch' ? selectedEndTime : undefined,
        end_date: eventType === 'generisch' && selectedEndDate ? selectedEndDate : undefined,
        opponent: selectedOpponent,
        team_ids: selectedTeamIds,
        event_type: eventType,
        template_id: selectedTemplate,
        venue_id: selectedVenueId,
        rsvp_default_players: gameDefaultPlayers,
        rsvp_default_extended: gameDefaultExtended,
        rsvp_require_reason: gameRsvpRequireReason,
        slots: slotsPayload,
      })
      if (r.data?.regen_summary) {
        setRegenSummary(r.data.regen_summary)
      }
      await loadGames()
      closeDialog()
    } catch (e) {
      // 403 = Team-Scope (game-mutation-team-scope): bei generischen Events muss
      // eine eigene Mannschaft dabei sein, bei Spielen müssen alle eigene sein.
      setCreateError(
        errorStatus(e) === 403
          ? 'Mindestens eine deiner eigenen Mannschaften muss am Event beteiligt sein.'
          : 'Event konnte nicht angelegt werden. Ist eine aktive Saison vorhanden?'
      )
    } finally {
      setCreating(false)
    }
  }

  const doCreateTraining = async () => {
    if (!selectedDate || selectedTeamIds.length === 0 || !trainingStartTime || !trainingEndTime || !activeSeasonId) {
      setCreateError('Bitte alle Pflichtfelder ausfüllen. Ist eine aktive Saison vorhanden?')
      return
    }
    setCreating(true)
    setCreateError(null)
    try {
      await api.post('/training-sessions', {
        team_id: selectedTeamIds[0],
        season_id: activeSeasonId,
        title: trainingTitle,
        date: selectedDate,
        start_time: trainingStartTime,
        end_time: trainingEndTime,
        venue_id: trainingVenueId,
        rsvp_default_players: gameDefaultPlayers,
        rsvp_default_extended: gameDefaultExtended,
        rsvp_require_reason: gameRsvpRequireReason,
      })
      await loadTrainings()
      closeDialog()
    } catch {
      setCreateError('Training konnte nicht angelegt werden.')
    } finally {
      setCreating(false)
    }
  }

  const doCreateSerie = async () => {
    if (selectedTeamIds.length === 0 || !seriesValidFrom || !seriesValidUntil || !trainingStartTime || !trainingEndTime || !activeSeasonId) {
      setCreateError('Bitte alle Pflichtfelder ausfüllen. Ist eine aktive Saison vorhanden?')
      return
    }
    const teamName = teams.find(t => t.id === selectedTeamIds[0])?.name ?? 'Training'
    setCreating(true)
    setCreateError(null)
    try {
      await api.post('/training-series', {
        team_id: selectedTeamIds[0],
        season_id: activeSeasonId,
        name: `Training ${teamName}`,
        venue_id: trainingVenueId,
        day_of_week: seriesWeekday,
        start_time: trainingStartTime,
        end_time: trainingEndTime,
        valid_from: seriesValidFrom,
        valid_until: seriesValidUntil,
        rsvp_default_players: gameDefaultPlayers,
        rsvp_default_extended: gameDefaultExtended,
        rsvp_require_reason: gameRsvpRequireReason,
      })
      await loadTrainings()
      closeDialog()
    } catch {
      setCreateError('Trainingsserie konnte nicht angelegt werden.')
    } finally {
      setCreating(false)
    }
  }

  const loadTemplates = async () => {
    try {
      const r = await api.get('/duty-templates')
      setTemplates(r.data ?? [])
    } catch {
      setTemplates([])
    }
  }

  const handleFetchPreview = async () => {
    if (!selectedTemplate || !selectedDate || selectedTeamIds.length === 0) return
    setPreviewLoading(true)
    try {
      const r = await api.get(buildPreviewUrl({
        templateId: selectedTemplate,
        eventType,
        time: selectedTime,
        date: selectedDate,
        endTime: selectedEndTime,
        teamIds: selectedTeamIds,
      }))
      const slots: SlotPreview[] = r.data ?? []
      setPreview(slots)
      setSelectedSlotIndices(new Set(slots.map((_, i) => i)))
      setWizardStep(4)
    } catch {
      setPreview([])
      setSelectedSlotIndices(new Set())
      setWizardStep(4)
    } finally {
      setPreviewLoading(false)
    }
  }

  const toggleSlot = (i: number) => {
    setSelectedSlotIndices(prev => {
      const next = new Set(prev)
      if (next.has(i)) next.delete(i); else next.add(i)
      return next
    })
  }

  const handleAbsencePreview = async () => {
    setAbsenceError('')
    if (!absenceForm.start_date || !absenceForm.end_date) {
      setAbsenceError('Bitte Start- und Enddatum angeben.')
      return
    }
    if (absenceForm.start_date > absenceForm.end_date) {
      setAbsenceError('Startdatum muss vor dem Enddatum liegen.')
      return
    }
    if (user?.isParent && absenceChildren.length > 0 && absenceForm.member_ids.length === 0) {
      setAbsenceError(absenceChildren.length === 1 ? 'Bitte ein Kind auswählen.' : 'Bitte mindestens ein Kind auswählen.')
      return
    }
    setAbsencePreviewLoading(true)
    try {
      const params = new URLSearchParams({
        from: absenceForm.start_date,
        to: absenceForm.end_date,
        ...(absenceForm.member_ids.length > 0 ? { member_ids: absenceForm.member_ids.join(',') } : {}),
      })
      const r = await api.get(`/absences/preview?${params}`)
      const events = r.data ?? []
      if (events.length === 0) {
        await doSaveAbsence()
      } else {
        setAbsencePreviewEvents(events)
      }
    } catch {
      setAbsenceError('Fehler beim Laden der Vorschau.')
    } finally {
      setAbsencePreviewLoading(false)
    }
  }

  const doSaveAbsence = async () => {
    setAbsenceSaving(true)
    setAbsenceError('')
    try {
      const body: Record<string, unknown> = {
        type: absenceForm.type,
        start_date: absenceForm.start_date,
        end_date: absenceForm.end_date,
        note: absenceForm.note,
      }
      if (absenceForm.member_ids.length > 0) {
        body.member_ids = absenceForm.member_ids
      }
      await api.post('/absences', body)
      closeDialog()
      loadAbsences()
      loadTrainings()
    } catch (err: unknown) {
      const resp = (err as { response?: { status?: number; data?: { conflicts?: Array<{ member_name: string }> } } })?.response
      if (resp?.status === 409) {
        const conflicts = resp.data?.conflicts ?? []
        if (conflicts.length > 0) {
          const names = conflicts.map(c => c.member_name).filter(Boolean).join(', ')
          setAbsenceError(`Eintragung abgebrochen — ${names} ${conflicts.length === 1 ? 'hat' : 'haben'} in diesem Zeitraum bereits eine Abwesenheit.`)
        } else {
          setAbsenceError('Eine Abwesenheit dieses Typs überschneidet sich bereits mit diesem Zeitraum.')
        }
      } else {
        setAbsenceError('Fehler beim Speichern.')
      }
      setAbsencePreviewEvents(null)
      setAbsenceSaving(false)
    }
  }

  const closeDialog = () => {
    setShowCreate(false)
    setWizardStep(1)
    setEventType('')
    setSelectedDate('')
    setSelectedTime('15:00')
    setSelectedEndTime('16:00')
    setSelectedEndDate('')
    setSelectedOpponent('')
    setSelectedTeamIds([])
    setSelectedTemplate(null)
    setPreview([])
    setSelectedSlotIndices(new Set())
    setCreateError(null)
    setTrainingTitle('')
    setTrainingStartTime('18:00')
    setTrainingEndTime('19:30')
    setTrainingVenueId(null)
    setSelectedVenueId(null)
    setSeriesWeekday(1)
    setSeriesValidFrom('')
    setSeriesValidUntil('')
    setGameDefaultPlayers('none')
    setGameDefaultExtended('none')
    setGameRsvpRequireReason(1)
    setWizardHost(null)
    setWizardHostId(null)
    setHostPreview(null)
    setHostError(null)
    setAbsenceForm({ member_ids: [], type: 'vacation', start_date: '', end_date: '', note: '' })
    setAbsencePreviewEvents(null)
    setAbsencePreviewLoading(false)
    setAbsenceSaving(false)
    setAbsenceError('')
  }

  useEscapeKey(
    showCreate ? closeDialog :
    editingGame ? () => setEditingGame(null) :
    editingTraining ? () => setEditingTraining(null) :
    infoItem ? () => setInfoItem(null) :
    showEventMenu ? () => setShowEventMenu(false) :
    null
  )

  // Klick außerhalb schließt das Aktionsmenü (gleiches Muster wie MembersPage).
  useEffect(() => {
    if (!showEventMenu) return
    const handler = (e: MouseEvent) => {
      if (eventMenuRef.current && !eventMenuRef.current.contains(e.target as Node)) {
        setShowEventMenu(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showEventMenu])

  const canCreateAbsence = Boolean(user && (user.clubFunctions?.includes('spieler') || user.isParent))

  return (
    <div>
      {regenSummary && (
        <RegenSummaryCard summary={regenSummary} onDismiss={() => setRegenSummary(null)} />
      )}
      {importResult && (
        <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text flex items-start gap-2">
          <Check className="w-4 h-4 mt-0.5 shrink-0" />
          <span className="flex-1">
            H4A-Import: {importResult.imported} neu angelegt, {importResult.updated} aktualisiert
            {importResult.skipped > 0 && `, ${importResult.skipped} übersprungen`}.
          </span>
          <button onClick={() => setImportResult(null)} aria-label="Hinweis schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}
      {bulkRegenResult && (
        <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text flex items-start gap-2">
          <Check className="w-4 h-4 mt-0.5 shrink-0" />
          <span className="flex-1">
            Dienste aktualisiert: {bulkRegenResult.totals.games} Termine, +{bulkRegenResult.totals.created} / −{bulkRegenResult.totals.deleted}
            {bulkRegenResult.totals.assignments_lost > 0 && `, ${bulkRegenResult.totals.assignments_lost} Zuweisungen verloren`}.
          </span>
          <button onClick={() => setBulkRegenResult(null)} aria-label="Hinweis schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold shrink-0">Kalender</h1>
        {/* `hidden sm:block`: siehe TerminePage — auf Mobile fehlt neben Typ-Filter
            und Suchfeld der Platz. */}
        <select
          value={filterTeamId ?? ''}
          onChange={e => setFilterTeamId(e.target.value === '' ? null : Number(e.target.value))}
          className="hidden sm:block border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow shrink-0 max-w-[6rem]"
        >
          <option value="">Alle</option>
          {teams.filter(t => t.is_active).map(t => (
            <option key={t.id} value={t.id}>{shortNames.get(t.id) ?? t.name}</option>
          ))}
        </select>
        <div className="flex items-center gap-1.5 flex-1 flex-nowrap min-w-0">
          {([
            ['heim',      'Heim',       <Home className="w-3.5 h-3.5" />],
            ['auswärts',  'Auswärts',   <Plane className="w-3.5 h-3.5" />],
            ['generisch', 'Sonstiges',  <Calendar className="w-3.5 h-3.5" />],
            ['training',  'Training',   <Dumbbell className="w-3.5 h-3.5" />],
          ] as [string, string, React.ReactNode][]).map(([type, label, icon]) => (
            <button
              key={type}
              onClick={() => toggleType(type)}
              aria-label={label}
              className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors shrink-0 ${compact ? 'px-2' : 'px-3'} ${
                filterTypes.has(type)
                  ? getEventColors(type).filter
                  : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
              }`}
            >
              {icon}
              {!compact && <span>{label}</span>}
            </button>
          ))}
          <EventSearchInput
            value={query}
            onChange={setQuery}
            compact={compact}
            placeholder="Gegner, Ort, Notiz…"
            ariaLabel="Kalender filtern"
          />
        </div>
        {canSeeTeamAbsences && (
          <button
            onClick={() => {
              const next = !showTeamAbsences
              setShowTeamAbsences(next)
              sessionStorage.setItem('kalender_show_team_absences', String(next))
              loadAbsences(next, filterTeamId)
            }}
            aria-label="Mannschaftsabwesenheiten"
            title="Mannschaftsabwesenheiten"
            className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors shrink-0 ${compact ? 'px-2' : 'px-3'} ${
              showTeamAbsences
                ? 'bg-brand-blue text-white border-brand-blue'
                : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
            }`}
          >
            <UserX className="w-3.5 h-3.5" />
            {!compact && <span>Abwesenheit</span>}
          </button>
        )}
        {(canEdit || canCreateAbsence || canImportGames || canBulkRegenDuties) && (
          <div ref={eventMenuRef} className="relative shrink-0">
            <div className="flex">
              {(canEdit || canCreateAbsence) && (
                <button
                  onClick={() => {
                    if (!canEdit && canCreateAbsence) {
                      setEventType('abwesenheit')
                      setWizardStep(2)
                      if (user?.isParent && absenceChildren.length === 0) loadAbsenceChildren()
                    }
                    setShowCreate(true)
                  }}
                  aria-label={canEdit ? 'Event' : 'Abwesenheit'}
                  className={`flex items-center gap-1 py-1.5 text-xs font-medium bg-brand-yellow text-brand-black border border-brand-yellow hover:bg-brand-black hover:text-brand-yellow transition-colors ${compact ? 'px-2' : 'px-3'} ${(canImportGames || canBulkRegenDuties) ? 'rounded-l-md' : 'rounded-md'}`}
                >
                  <Plus className="w-3.5 h-3.5" />
                  {!compact && <span>{canEdit ? 'Event' : 'Abwesenheit'}</span>}
                </button>
              )}
              {(canImportGames || canBulkRegenDuties) && (
                <button
                  onClick={() => setShowEventMenu(v => !v)}
                  aria-label="Weitere Aktionen"
                  aria-expanded={showEventMenu}
                  aria-haspopup="menu"
                  className={`flex items-center py-1.5 px-2 text-xs font-medium bg-brand-yellow text-brand-black border border-brand-yellow hover:bg-brand-black hover:text-brand-yellow transition-colors ${
                    canEdit || canCreateAbsence ? 'border-l border-l-brand-black/20 rounded-r-md' : 'rounded-md'
                  }`}
                >
                  <ChevronDown className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
            {showEventMenu && (canImportGames || canBulkRegenDuties) && (
              <div role="menu" className="absolute right-0 mt-1 w-60 bg-white border border-brand-border rounded-md shadow-lg z-20 overflow-hidden">
                {canImportGames && (
                  <button
                    role="menuitem"
                    onClick={() => { setShowEventMenu(false); setShowH4AImport(true) }}
                    className="w-full flex items-center gap-2 text-left px-4 py-2.5 text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
                  >
                    <Download className="w-4 h-4 shrink-0" />
                    Spielplan aus Handball4All
                  </button>
                )}
                {canBulkRegenDuties && (
                  <button
                    role="menuitem"
                    onClick={() => { setShowEventMenu(false); setShowBulkRegen(true) }}
                    className="w-full flex items-center gap-2 text-left px-4 py-2.5 text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
                  >
                    <RefreshCw className="w-4 h-4 shrink-0" />
                    Dienste aktualisieren
                  </button>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Month navigation */}
      <div className="flex items-center gap-4 mb-4">
        <button onClick={prevMonth} aria-label="Vorheriger Monat" className="p-2 hover:bg-brand-border-subtle rounded-lg transition-colors text-brand-text">
          <ChevronLeft className="w-5 h-5" />
        </button>
        <span className="text-lg font-semibold w-44 text-center">{MONTHS[month]} {year}</span>
        <button onClick={nextMonth} aria-label="Nächster Monat" className="p-2 hover:bg-brand-border-subtle rounded-lg transition-colors text-brand-text">
          <ChevronRight className="w-5 h-5" />
        </button>
        <button
          onClick={goToToday}
          disabled={year === now.getFullYear() && month === now.getMonth()}
          title="Zum aktuellen Monat springen"
          className="flex items-center gap-1.5 rounded-md px-3 py-2 text-sm font-medium bg-brand-yellow text-brand-black hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <CalendarClock className="w-4 h-4" />
          <span>Heute</span>
        </button>
        <div className="flex-1" />
      </div>

      {/* Calendar */}
      <div className="rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <div
        ref={calendarRef}
        className="bg-brand-surface-card select-none"
        style={{ touchAction: 'pan-y' }}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerCancel}
      >
        <div className="grid grid-cols-7 bg-brand-surface-card border-b border-brand-border-subtle">
          {WEEKDAYS.map(d => (
            <div key={d} className="text-center text-xs font-semibold py-2 text-brand-text-muted uppercase tracking-wide">{d}</div>
          ))}
        </div>
        <div className="grid grid-cols-7">
          {Array.from({ length: firstDayOfWeek }).map((_, i) => (
            <div key={`pad-${i}`} className="min-h-[90px] border-r border-b border-brand-border-subtle" />
          ))}
          {Array.from({ length: daysInMonth }).map((_, i) => {
            const day = i + 1
            const dateStr = padDate(year, month, day)
            const dayGames = gamesByDate[dateStr] ?? []
            const dayTrainings = trainingsByDate[dateStr] ?? []
            const dayAbsences = absencesForDay(dateStr)
            const isToday = dateStr === todayStr
            return (
              <div key={day} className="relative @container group min-h-[90px] p-1.5 border-r border-b border-brand-border-subtle">
                {dayAbsences.map(({ absence, isFirst, isLast }) => (
                  <div
                    key={`abs-${absence.id}`}
                    className={`absolute top-[4px] left-[4px] right-[4px] h-5 border cursor-pointer z-20 ${
                      !absence.is_own
                        ? 'bg-brand-blue/20 border-brand-blue/60'
                        : absence.type === 'injury'
                          ? 'bg-red-400/20 border-red-400/60'
                          : 'bg-brand-yellow/20 border-brand-yellow/60'
                    } ${isFirst && isLast ? 'rounded-full' : isFirst ? 'rounded-l-full' : isLast ? 'rounded-r-full' : ''}`}
                    title={`${absence.member_name}: ${absence.type === 'vacation' ? 'Urlaub' : 'Verletzung'} ${absence.start_date}–${absence.end_date}`}
                    onPointerDown={e => e.stopPropagation()}
                    onClick={() => setInfoItem({ type: 'absence', absence })}
                  />
                ))}
                <div className="relative z-10">
                <div className="flex items-center justify-between mb-1">
                  <span className={`text-xs leading-none flex items-center justify-center relative z-20 ${isToday ? 'font-bold w-5 h-5 rounded-full bg-brand-yellow text-brand-black' : 'text-brand-text-subtle'}`}>{day}</span>
                  {(canEdit || canCreateAbsence) && (
                    <button
                      onPointerDown={e => e.stopPropagation()}
                      onClick={e => { e.stopPropagation(); openWizardWithDate(dateStr) }}
                      className="opacity-0 group-hover:opacity-100 transition-opacity p-0.5 rounded text-brand-text-subtle hover:text-brand-text hover:bg-brand-border-subtle"
                      title={canEdit ? 'Event anlegen' : 'Abwesenheit eintragen'}
                    >
                      <Plus className="w-3 h-3" />
                    </button>
                  )}
                </div>
                {dayGames.map(g => {
                  const teamsForRender = g.teams.map(t => ({
                    id: t.id,
                    name: t.name,
                    display_short: t.display_short ?? shortNames.get(t.id),
                    display_long: t.display_long ?? t.name,
                  }))
                  const tileLabel = formatTeamList(teamsForRender, 'kalender')
                  const tooltipLabel = g.teams.length > 1 ? 'Mehrere Teams' : tileLabel
                  const noteText = (g.note ?? '').trim()
                  const openSlots = g.slot_count > 0 && g.filled_count < g.total_count
                    ? g.total_count - g.filled_count
                    : 0
                  const showWarning = noteText !== '' || openSlots > 0
                  const warningParts: string[] = []
                  if (noteText !== '') warningParts.push(`Hinweis: ${noteText}`)
                  if (openSlots > 0) warningParts.push(`${openSlots} offene Dienst-Slots`)
                  const warningTitle = warningParts.join('\n')
                  return (
                  <button
                    key={g.id}
                    onPointerDown={e => e.stopPropagation()}
                    onClick={() => setInfoItem({ type: 'game', game: { ...g, teams: teamsForRender.map(t => ({ id: t.id, name: t.display_short ?? t.name })) } })}
                    title={`${tooltipLabel} · ${g.opponent || '–'} · ${g.time}`}
                    className={`w-full text-left mb-1 p-1.5 rounded-md text-xs transition-colors border ${getEventColors(g.event_type).pill}`}
                  >
                    <div className="flex items-center gap-1 mb-0.5">
                      {g.event_type === 'heim'
                        ? <Home className="w-3 h-3 text-brand-text-muted shrink-0" />
                        : g.event_type === 'auswärts'
                        ? <Plane className="w-3 h-3 text-brand-text-muted shrink-0" />
                        : <Calendar className="w-3 h-3 text-brand-text-muted shrink-0" />}
                      <span className="hidden @tile-sm:inline font-semibold truncate text-brand-text flex-1">
                        {tileLabel || '?'}
                      </span>
                      {showWarning && (
                        <span
                          className="ml-auto inline-flex items-center shrink-0"
                          aria-label={warningTitle}
                          title={warningTitle}
                        >
                          <AlertTriangle className="w-3 h-3 text-brand-danger" />
                        </span>
                      )}
                    </div>
                    <div className="hidden @tile-md:block truncate text-brand-text-muted leading-tight">
                      {g.opponent || '–'}
                    </div>
                    <div className="flex items-center gap-1 text-brand-text-subtle leading-tight">
                      <span>{g.time}</span>
                      <span className="hidden @tile-sm:inline-flex items-center gap-0.5 text-green-600">
                        <Check className="w-2.5 h-2.5" />{g.confirmed_count}
                      </span>
                      <span className="hidden @tile-sm:inline-flex items-center gap-0.5 text-brand-danger">
                        <X className="w-2.5 h-2.5" />{g.declined_count}
                      </span>
                    </div>
                  </button>
                  )
                })}
                {dayTrainings.map(t => (
                  <button
                    key={`t-${t.id}`}
                    onPointerDown={e => e.stopPropagation()}
                    title={`${shortNames.get(t.team_id) ?? (t.title || 'Training')} · ${t.start_time}`}
                    onClick={() => setInfoItem({ type: 'training', training: { ...t, team_name: shortNames.get(t.team_id) } })}
                    className={`w-full text-left mb-1 p-1.5 rounded-md text-xs border ${
                      t.status === 'cancelled'
                        ? 'bg-white/50 border-brand-border-subtle opacity-50 line-through'
                        : `${getEventColors('training').pill} transition-colors`
                    }`}
                  >
                    <div className="flex items-center gap-1 mb-0.5">
                      <Dumbbell className={`w-3 h-3 shrink-0 ${getEventColors('training').pillIcon}`} />
                      <span className="hidden @tile-sm:inline font-semibold truncate text-brand-text flex-1">
                        {shortNames.get(t.team_id) ?? (t.title || 'Training')}
                      </span>
                      {(t.note ?? '').trim() !== '' && (
                        <span
                          className="ml-auto inline-flex items-center shrink-0"
                          aria-label={`Hinweis: ${t.note}`}
                          title={`Hinweis: ${t.note}`}
                        >
                          <AlertTriangle className="w-3 h-3 text-brand-danger" />
                        </span>
                      )}
                    </div>
                    <div className="hidden @tile-md:block leading-tight">&nbsp;</div>
                    <div className="flex items-center gap-1.5 text-brand-text-subtle leading-tight">
                      <span>{t.start_time}</span>
                      {t.my_rsvp_locked ? (
                        <span className="hidden @tile-sm:inline-flex items-center gap-0.5 text-brand-danger" title="Durch Abwesenheit gesetzt">
                          <X className="w-2.5 h-2.5" />
                        </span>
                      ) : (
                        <>
                          <span className="hidden @tile-sm:inline-flex items-center gap-0.5 text-green-600">
                            <Check className="w-2.5 h-2.5" />{t.confirmed_count}
                          </span>
                          <span className="hidden @tile-sm:inline-flex items-center gap-0.5 text-brand-danger">
                            <X className="w-2.5 h-2.5" />{t.declined_count}
                          </span>
                        </>
                      )}
                    </div>
                  </button>
                ))}
                </div>
              </div>
            )
          })}
        </div>
      </div>
      </div>


      {/* Event Wizard Dialog */}
      {showCreate && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-md shadow-2xl max-h-[90vh] overflow-y-auto">
            {wizardStep === 1 && (
              <div>
                <h2 className="text-lg font-bold mb-6 text-brand-text">Welche Art von Event?</h2>
                <div className="space-y-3">
                  {(['heim', 'auswärts', 'generisch'] as const).map(type => (
                    <button
                      key={type}
                      onClick={() => {
                        setEventType(type)
                        setGameRsvpRequireReason(type === 'generisch' ? 0 : 1)
                        if (type === 'heim') {
                          const homeVenue = allVenues.find(v => v.is_home_venue)
                          setSelectedVenueId(homeVenue?.id ?? null)
                        } else {
                          setSelectedVenueId(null)
                        }
                        setWizardStep(2)
                      }}
                      className="w-full p-4 border-2 border-brand-border rounded-lg text-left hover:bg-brand-border-subtle hover:border-brand-yellow transition-colors"
                    >
                      <div className="font-semibold flex items-center gap-2 text-brand-text">
                        {type === 'heim' && <><Home className="w-4 h-4" /> Heimspiel</>}
                        {type === 'auswärts' && <><Plane className="w-4 h-4" /> Auswärtsspiel</>}
                        {type === 'generisch' && <><Calendar className="w-4 h-4" /> Sonstiges Event</>}
                      </div>
                      <div className="text-xs text-brand-text-muted mt-1">
                        {type === 'heim' && 'Heimspiel gegen eine Mannschaft'}
                        {type === 'auswärts' && 'Auswärtsspiel gegen eine Mannschaft'}
                        {type === 'generisch' && 'Event für mehrere Mannschaften'}
                      </div>
                    </button>
                  ))}
                  {canManageTrainings && (
                    <>
                      <button
                        onClick={() => { setEventType('training'); setWizardStep(2) }}
                        className="w-full p-4 border-2 border-brand-border rounded-lg text-left hover:bg-brand-border-subtle hover:border-brand-yellow transition-colors"
                      >
                        <div className="font-semibold flex items-center gap-2 text-brand-text">
                          <Dumbbell className="w-4 h-4" /> Einzeltraining
                        </div>
                        <div className="text-xs text-brand-text-muted mt-1">Einmaliger Trainingstermin</div>
                      </button>
                      <button
                        onClick={() => { setEventType('serie'); setWizardStep(2) }}
                        className="w-full p-4 border-2 border-brand-border rounded-lg text-left hover:bg-brand-border-subtle hover:border-brand-yellow transition-colors"
                      >
                        <div className="font-semibold flex items-center gap-2 text-brand-text">
                          <RefreshCw className="w-4 h-4" /> Trainingsserie
                        </div>
                        <div className="text-xs text-brand-text-muted mt-1">Wöchentlich wiederkehrender Termin</div>
                      </button>
                    </>
                  )}
                  {canCreateAbsence && (
                    <button
                      onClick={() => {
                        setEventType('abwesenheit')
                        setWizardStep(2)
                        if (user?.isParent && absenceChildren.length === 0) loadAbsenceChildren()
                      }}
                      className="w-full p-4 border-2 border-brand-border rounded-lg text-left hover:bg-brand-border-subtle hover:border-brand-yellow transition-colors"
                    >
                      <div className="font-semibold flex items-center gap-2 text-brand-text">
                        <Calendar className="w-4 h-4" /> Abwesenheit
                      </div>
                      <div className="text-xs text-brand-text-muted mt-1">Urlaub oder Verletzung / Sportverbot eintragen</div>
                    </button>
                  )}
                </div>
                <div className="flex gap-2 pt-4">
                  <button onClick={closeDialog} className={`flex-1 ${BTN_SECONDARY}`}>Abbrechen</button>
                </div>
              </div>
            )}

            {wizardStep === 2 && (eventType === 'heim' || eventType === 'auswärts' || eventType === 'generisch') && (
              <div>
                <h2 className="text-lg font-bold mb-4 text-brand-text">Event-Details</h2>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Datum *</label>
                    <input type="date" value={selectedDate} min={todayStr} onChange={e => {
                      const date = e.target.value
                      setSelectedDate(date)
                      if (eventType === 'generisch' && selectedEndDate && selectedEndDate < date) setSelectedEndDate(date)
                    }} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">
                      {eventType === 'generisch' ? 'Beginn' : 'Anwurfzeit'}
                    </label>
                    <input type="time" value={selectedTime} onChange={e => setSelectedTime(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  {eventType === 'generisch' && (
                    <div>
                      <label className="block text-sm font-medium text-brand-text-muted mb-1">Ende</label>
                      <input type="time" value={selectedEndTime} onChange={e => setSelectedEndTime(e.target.value)} className={INPUT_WIZ} />
                    </div>
                  )}
                  {eventType === 'generisch' && (
                    <div>
                      <label className="block text-sm font-medium text-brand-text-muted mb-1">Enddatum <span className="text-brand-text-subtle font-normal">(optional, für mehrtägige Events)</span></label>
                      <input type="date" value={selectedEndDate} onChange={e => setSelectedEndDate(e.target.value)}
                        min={selectedDate || todayStr} className={INPUT_WIZ} />
                      {selectedEndDate && selectedEndDate < selectedDate && (
                        <p className="text-xs text-brand-danger mt-1">Enddatum muss nach dem Startdatum liegen.</p>
                      )}
                    </div>
                  )}
                  {eventType !== 'generisch' && (
                    <div>
                      <label className="block text-sm font-medium text-brand-text-muted mb-1">Gegner *</label>
                      <input type="text" value={selectedOpponent} onChange={e => setSelectedOpponent(e.target.value)}
                        placeholder="Name des Gegners" className={INPUT_WIZ} />
                    </div>
                  )}
                  {eventType === 'generisch' && (
                    <div>
                      <label className="block text-sm font-medium text-brand-text-muted mb-1">Event-Name *</label>
                      <input type="text" value={selectedOpponent} onChange={e => setSelectedOpponent(e.target.value)}
                        placeholder="Name des Events" className={INPUT_WIZ} />
                    </div>
                  )}
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Ort</label>
                    <VenuePicker value={selectedVenueId} onChange={setSelectedVenueId} />
                  </div>
                  {eventType === 'heim' && canEdit && selectedDate && wizardHost && (
                    <div>
                      <GameDayHostSelect
                        id="wizard-game-day-host"
                        date={selectedDate}
                        value={wizardHostId}
                        options={ausrichterOptions}
                        onChange={setWizardHostId}
                      />
                      {hostDiffersInWizard && (
                        <p className="text-xs text-brand-text-muted mt-1">
                          Weicht vom geltenden Wert ab ({wizardHost.ausrichter_name}) — vor dem Anlegen erscheint eine Vorschau.
                        </p>
                      )}
                    </div>
                  )}
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-2">
                      {eventType === 'generisch' ? 'Mannschaften *' : 'Mannschaft *'}
                    </label>
                    {eventType === 'generisch' ? (
                      // Generische Events sind mannschaftsübergreifend: alle aktiven
                      // Mannschaften des Vereins (allTeamNames = /teams/names), nicht nur
                      // die eigenen. Der Server verlangt weiterhin, dass mindestens eine
                      // eigene Mannschaft dabei ist (game-mutation-team-scope).
                      <div className="space-y-2">
                        {allTeamNames.map(t => (
                          <label key={t.id} className="flex items-center gap-2">
                            <input type="checkbox" checked={selectedTeamIds.includes(t.id)}
                              onChange={e => {
                                if (e.target.checked) {
                                  setSelectedTeamIds([...selectedTeamIds, t.id])
                                } else {
                                  setSelectedTeamIds(selectedTeamIds.filter(id => id !== t.id))
                                }
                              }} className="rounded accent-brand-yellow" />
                            <span className="text-sm text-brand-text">{shortNames.get(t.id)}</span>
                          </label>
                        ))}
                      </div>
                    ) : (
                      <select value={selectedTeamIds[0] ?? ''} onChange={e => setSelectedTeamIds(e.target.value ? [Number(e.target.value)] : [])}
                        className={INPUT_WIZ}>
                        <option value="">Auswählen…</option>
                        {teams.filter(t => t.is_active).map(t => (
                          <option key={t.id} value={t.id}>{shortNames.get(t.id) ?? t.name}</option>
                        ))}
                      </select>
                    )}
                  </div>
                  <RsvpDefaultsEditor
                    idPrefix="kalender"
                    defaultPlayers={gameDefaultPlayers}
                    defaultExtended={gameDefaultExtended}
                    requireReason={gameRsvpRequireReason === 1}
                    onChangePlayers={setGameDefaultPlayers}
                    onChangeExtended={setGameDefaultExtended}
                    onChangeRequireReason={v => setGameRsvpRequireReason(v ? 1 : 0)}
                  />
                  {createError && <p className="text-brand-danger text-sm">{createError}</p>}
                </div>
                <div className="flex gap-2 pt-4">
                  <button onClick={() => setWizardStep(1)} className={BTN_SECONDARY}>← Zurück</button>
                  <button
                    onClick={() => {
                      if (selectedDate && selectedTeamIds.length > 0) {
                        loadTemplates().then(() => setWizardStep(3))
                      }
                    }}
                    disabled={!selectedDate || selectedTeamIds.length === 0 || (eventType === 'generisch' && !!selectedEndDate && selectedEndDate < selectedDate)}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >Weiter →</button>
                </div>
              </div>
            )}

            {wizardStep === 2 && eventType === 'training' && (
              <div>
                <h2 className="text-lg font-bold mb-4 text-brand-text">Einzeltraining anlegen</h2>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Titel</label>
                    <input type="text" value={trainingTitle} onChange={e => setTrainingTitle(e.target.value)}
                      placeholder="z. B. Konditionstraining" className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Datum *</label>
                    <input type="date" value={selectedDate} min={todayStr} onChange={e => setSelectedDate(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Startzeit *</label>
                    <input type="time" value={trainingStartTime} onChange={e => setTrainingStartTime(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Endzeit *</label>
                    <input type="time" value={trainingEndTime} onChange={e => setTrainingEndTime(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Ort</label>
                    <VenuePicker value={trainingVenueId} onChange={setTrainingVenueId} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Mannschaft *</label>
                    <select value={selectedTeamIds[0] ?? ''} onChange={e => setSelectedTeamIds(e.target.value ? [Number(e.target.value)] : [])}
                      className={INPUT_WIZ}>
                      <option value="">Auswählen…</option>
                      {teams.filter(t => t.is_active).map(t => (
                        <option key={t.id} value={t.id}>{shortNames.get(t.id) ?? t.name}</option>
                      ))}
                    </select>
                  </div>
                  <RsvpDefaultsEditor
                    idPrefix="kalender"
                    defaultPlayers={gameDefaultPlayers}
                    defaultExtended={gameDefaultExtended}
                    requireReason={gameRsvpRequireReason === 1}
                    onChangePlayers={setGameDefaultPlayers}
                    onChangeExtended={setGameDefaultExtended}
                    onChangeRequireReason={v => setGameRsvpRequireReason(v ? 1 : 0)}
                  />
                  {createError && <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{createError}</p>}
                </div>
                <div className="flex gap-2 pt-4">
                  <button onClick={() => setWizardStep(1)} className={BTN_SECONDARY}>← Zurück</button>
                  <button
                    onClick={doCreateTraining}
                    disabled={creating || !selectedDate || selectedTeamIds.length === 0}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >
                    {creating ? 'Anlegen…' : 'Training anlegen'}
                  </button>
                </div>
              </div>
            )}

            {wizardStep === 2 && eventType === 'serie' && (
              <div>
                <h2 className="text-lg font-bold mb-4 text-brand-text">Trainingsserie anlegen</h2>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Wochentag *</label>
                    <select value={seriesWeekday} onChange={e => setSeriesWeekday(Number(e.target.value))} className={INPUT_WIZ}>
                      {['Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag', 'Sonntag'].map((d, i) => (
                        <option key={i} value={i}>{d}</option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Startzeit *</label>
                    <input type="time" value={trainingStartTime} onChange={e => setTrainingStartTime(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Endzeit *</label>
                    <input type="time" value={trainingEndTime} onChange={e => setTrainingEndTime(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Ort</label>
                    <VenuePicker value={trainingVenueId} onChange={setTrainingVenueId} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Mannschaft *</label>
                    <select value={selectedTeamIds[0] ?? ''} onChange={e => setSelectedTeamIds(e.target.value ? [Number(e.target.value)] : [])}
                      className={INPUT_WIZ}>
                      <option value="">Auswählen…</option>
                      {teams.filter(t => t.is_active).map(t => (
                        <option key={t.id} value={t.id}>{shortNames.get(t.id) ?? t.name}</option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Gültig von *</label>
                    <input type="date" value={seriesValidFrom} min={todayStr} onChange={e => {
                      const from = e.target.value
                      setSeriesValidFrom(from)
                      if (seriesValidUntil && seriesValidUntil < from) setSeriesValidUntil(from)
                    }} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Gültig bis *</label>
                    <input type="date" value={seriesValidUntil} min={seriesValidFrom || undefined} onChange={e => setSeriesValidUntil(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <RsvpDefaultsEditor
                    idPrefix="kalender"
                    defaultPlayers={gameDefaultPlayers}
                    defaultExtended={gameDefaultExtended}
                    requireReason={gameRsvpRequireReason === 1}
                    onChangePlayers={setGameDefaultPlayers}
                    onChangeExtended={setGameDefaultExtended}
                    onChangeRequireReason={v => setGameRsvpRequireReason(v ? 1 : 0)}
                  />
                  {createError && <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{createError}</p>}
                </div>
                <div className="flex gap-2 pt-4">
                  <button onClick={() => setWizardStep(1)} className={BTN_SECONDARY}>← Zurück</button>
                  <button
                    onClick={doCreateSerie}
                    disabled={creating || selectedTeamIds.length === 0 || !seriesValidFrom || !seriesValidUntil}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >
                    {creating ? 'Anlegen…' : 'Serie anlegen'}
                  </button>
                </div>
              </div>
            )}

            {wizardStep === 2 && eventType === 'abwesenheit' && !absencePreviewEvents && (
              <div>
                <h2 className="text-lg font-bold mb-4 text-brand-text">Abwesenheit eintragen</h2>
                <div className="space-y-4">
                  {user?.isParent && absenceChildren.length > 1 && (
                    <div>
                      <label className="block text-xs font-medium text-brand-text-muted mb-1">Kinder *</label>
                      <div className="space-y-1 border border-brand-border rounded-md p-2">
                        {absenceChildren.map(c => {
                          const checked = absenceForm.member_ids.includes(c.id)
                          return (
                            <label
                              key={c.id}
                              className="flex items-center gap-2 px-2 py-2.5 sm:py-1.5 rounded hover:bg-brand-table-select cursor-pointer text-sm text-brand-text"
                            >
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => setAbsenceForm(f => ({
                                  ...f,
                                  member_ids: checked
                                    ? f.member_ids.filter(id => id !== c.id)
                                    : [...f.member_ids, c.id],
                                }))}
                                className="h-4 w-4 accent-brand-yellow"
                              />
                              <span>{c.name}</span>
                            </label>
                          )
                        })}
                      </div>
                    </div>
                  )}
                  <div>
                    <label className="block text-xs font-medium text-brand-text-muted mb-1">Typ</label>
                    <select
                      value={absenceForm.type}
                      onChange={e => setAbsenceForm(f => ({ ...f, type: e.target.value }))}
                      className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                    >
                      <option value="vacation">Urlaub / Sonstige Abwesenheit</option>
                      <option value="injury">Verletzung / Sportverbot</option>
                    </select>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-brand-text-muted mb-1">Von</label>
                      <input
                        type="date"
                        value={absenceForm.start_date}
                        min={todayStr}
                        onChange={e => {
                          const start = e.target.value
                          setAbsenceForm(f => ({
                            ...f,
                            start_date: start,
                            end_date: f.end_date && f.end_date < start ? start : f.end_date,
                          }))
                        }}
                        className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-brand-text-muted mb-1">Bis</label>
                      <input
                        type="date"
                        value={absenceForm.end_date}
                        min={absenceForm.start_date || undefined}
                        onChange={e => setAbsenceForm(f => ({ ...f, end_date: e.target.value }))}
                        className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-brand-text-muted mb-1">Notiz (optional)</label>
                    <input
                      type="text"
                      value={absenceForm.note}
                      onChange={e => setAbsenceForm(f => ({ ...f, note: e.target.value }))}
                      placeholder="z.B. Familienurlaub, Knieoperation…"
                      className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                    />
                  </div>
                  {absenceError && (
                    <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{absenceError}</p>
                  )}
                </div>
                <div className="flex gap-2 pt-5">
                  <button onClick={closeDialog} className="flex-1 border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors">Abbrechen</button>
                  <button
                    onClick={handleAbsencePreview}
                    disabled={absencePreviewLoading}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    {absencePreviewLoading ? 'Prüfe…' : 'Weiter'}
                  </button>
                </div>
              </div>
            )}

            {wizardStep === 2 && eventType === 'abwesenheit' && absencePreviewEvents && (
              <div>
                <div className="flex items-start gap-3 mb-4">
                  <AlertTriangle className="w-5 h-5 text-brand-danger shrink-0 mt-0.5" />
                  <div>
                    <h2 className="text-base font-semibold text-brand-text">Folgende Trainings &amp; Spiele werden automatisch abgesagt</h2>
                    <p className="text-sm text-brand-text-muted mt-1">Bestätigte Zusagen werden zurückgezogen, offene Termine abgesagt.</p>
                  </div>
                </div>
                <ul className="space-y-1.5 mb-5 max-h-48 overflow-y-auto">
                  {absencePreviewEvents.map(ev => (
                    <li key={`${ev.event_type}-${ev.event_id}`} className={`flex items-center gap-2 text-sm ${ev.pending ? 'text-brand-text-muted' : 'text-brand-text'}`}>
                      <span className="text-brand-text-subtle w-16 shrink-0">{ev.date}</span>
                      <span>{ev.name}</span>
                      <span className="ml-auto text-xs text-brand-text-subtle">{ev.event_type === 'training' ? 'Training' : 'Spiel'}</span>
                    </li>
                  ))}
                </ul>
                <div className="flex gap-2">
                  <button
                    onClick={() => setAbsencePreviewEvents(null)}
                    className="flex-1 border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
                  >
                    Zurück
                  </button>
                  <button
                    onClick={doSaveAbsence}
                    disabled={absenceSaving}
                    className="flex-1 bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    {absenceSaving ? 'Speichert…' : 'Trotzdem eintragen'}
                  </button>
                </div>
              </div>
            )}

            {wizardStep === 3 && (
              <div>
                <h2 className="text-lg font-bold mb-4 text-brand-text">Dienstplan-Vorlage</h2>
                {(() => {
                  const filteredTemplates = templates.filter(t => t.template_type === eventType)
                  return (
                    <div className="space-y-2 mb-4">
                      <label className="flex items-center gap-2 p-3 border border-brand-border-subtle rounded-lg hover:bg-brand-border-subtle cursor-pointer">
                        <input type="radio" name="template" checked={selectedTemplate === null}
                          onChange={() => setSelectedTemplate(null)} className="rounded-full accent-brand-yellow" />
                        <div className="flex-1">
                          <div className="font-medium text-sm text-brand-text">— Keine Vorlage (keine Auto-Dienste) —</div>
                          <div className="text-xs text-brand-text-muted">Es werden keine Dienste automatisch erzeugt.</div>
                        </div>
                      </label>
                      {filteredTemplates.map(t => (
                        <label key={t.id} className="flex items-center gap-2 p-3 border border-brand-border-subtle rounded-lg hover:bg-brand-border-subtle cursor-pointer">
                          <input type="radio" name="template" checked={selectedTemplate === t.id}
                            onChange={() => setSelectedTemplate(t.id)} className="rounded-full accent-brand-yellow" />
                          <div className="flex-1">
                            <div className="font-medium text-sm text-brand-text">{t.name}</div>
                            {t.template_type === 'generisch' && (
                              <div className="text-xs text-brand-text-muted">{t.duration_minutes} Min</div>
                            )}
                          </div>
                        </label>
                      ))}
                    </div>
                  )
                })()}
                <div className="flex gap-2 pt-4">
                  <button onClick={() => setWizardStep(2)} className={BTN_SECONDARY}>← Zurück</button>
                  <button
                    onClick={() => {
                      if (selectedTemplate) {
                        handleFetchPreview()
                      } else {
                        setWizardStep(4)
                      }
                    }}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                    disabled={previewLoading || creating}
                  >
                    {previewLoading || creating ? 'Laden…' : 'Weiter →'}
                  </button>
                </div>
              </div>
            )}

            {wizardStep === 4 && (
              <div>
                <h2 className="text-lg font-bold mb-4 text-brand-text">Dienste bestätigen</h2>
                {preview.length === 0 ? (
                  <p className="text-sm text-brand-text-muted mb-4">Keine Dienste vorhanden.</p>
                ) : (
                  <>
                    <p className="text-sm text-brand-text-muted mb-3">
                      Dienste ({selectedSlotIndices.size} ausgewählt):
                    </p>
                    <div className="space-y-1.5 mb-4 max-h-56 overflow-y-auto">
                      {preview.map((s, i) => (
                        <label key={i} className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-brand-border-subtle cursor-pointer">
                          <input type="checkbox" checked={selectedSlotIndices.has(i)} onChange={() => toggleSlot(i)}
                            className="rounded accent-brand-yellow" />
                          <span className="font-mono text-sm font-semibold w-12 text-brand-text">{s.event_time}</span>
                          <span className="text-sm flex-1 text-brand-text">{s.duty_type_name}</span>
                          {s.role_desc && <span className="text-xs text-brand-text-subtle">({s.role_desc})</span>}
                          <span className="text-xs text-brand-text-subtle ml-auto">{s.slots_count}×</span>
                        </label>
                      ))}
                    </div>
                  </>
                )}
                {createError && <p className="text-brand-danger text-sm mb-3">{createError}</p>}
                <div className="flex gap-2 pt-2">
                  <button onClick={() => setWizardStep(3)} className={BTN_SECONDARY}>← Zurück</button>
                  <button
                    onClick={() => confirmCreateGame(selectedTemplate ? preview.filter((_, i) => selectedSlotIndices.has(i)) : [])}
                    disabled={creating || hostBusy}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >
                    {creating ? 'Anlegen…' : 'Bestätigen'}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
      {hostPreview && (
        <GameDayHostPreviewDialog
          preview={hostPreview}
          targetName={ausrichterOptions.find(o => o.id === wizardHostId)?.name ?? ''}
          busy={hostBusy || creating}
          error={hostError}
          onCancel={() => { setHostPreview(null); setHostError(null) }}
          onConfirm={() => applyHostThenCreate(selectedTemplate ? preview.filter((_, i) => selectedSlotIndices.has(i)) : [])}
        />
      )}
      {editingGame && (
        <GameEditModal
          game={editingGame}
          onClose={() => setEditingGame(null)}
          onSaved={s => { if (s) setRegenSummary(s); loadGames(); setEditingGame(null) }}
          onDeleted={s => { if (s) setRegenSummary(s); loadGames(); setEditingGame(null) }}
        />
      )}
      {editingTraining && (
        <TrainingEditModal
          session={editingTraining}
          teamName={teams.find(t => t.id === editingTraining.team_id)?.name}
          onClose={() => setEditingTraining(null)}
          onSaved={() => { loadTrainings(); setEditingTraining(null) }}
        />
      )}
      {infoItem && (
        <EventInfoModal
          type={infoItem.type}
          game={infoItem.game}
          training={infoItem.training}
          absence={infoItem.absence}
          onClose={() => setInfoItem(null)}
          onEdit={canEdit && infoItem.type !== 'absence' ? () => {
            if (infoItem.type === 'game' && infoItem.game) { setInfoItem(null); setEditingGame(infoItem.game) }
            else if (infoItem.type === 'training' && infoItem.training) { setInfoItem(null); setEditingTraining(infoItem.training) }
          } : undefined}
          onDienste={canEdit && infoItem.type === 'game' && infoItem.game
            ? () => { const id = infoItem.game!.id; setInfoItem(null); setDetailGameId(id) }
            : undefined}
          canEditAbsence={infoItem.type === 'absence' && !!infoItem.absence && infoItem.absence.can_edit}
          onAbsenceChanged={() => { loadAbsences(); setInfoItem(null) }}
          canManageGameDayHost={canEdit}
          onGameDayHostApplied={loadGames}
        />
      )}
      {detailGameId !== null && (
        <SpieltagDetailModal
          gameId={detailGameId}
          onClose={() => setDetailGameId(null)}
          onChanged={loadGames}
          onDeleted={() => { loadGames(); setDetailGameId(null) }}
        />
      )}
      {/* Nur im geöffneten Zustand gemountet — beim Schließen verschwinden die
          eingegebenen H4A-Zugangsdaten mit dem Komponenten-State. */}
      {showH4AImport && (
      <H4AImportModal
        isOpen
        onClose={() => setShowH4AImport(false)}
        onImported={result => {
          setImportResult({ imported: result.imported, updated: result.updated, skipped: result.skipped })
          if (result.regen_summary) setRegenSummary(result.regen_summary as RegenSummary)
          loadGames()
        }}
      />
      )}
      {/* Nur im geöffneten Zustand gemountet — analog zum H4A-Import bleibt der
          Preview-/Formular-State beim Schließen nicht hängen. */}
      {showBulkRegen && (
      <DutyBulkRegenModal
        isOpen
        onClose={() => setShowBulkRegen(false)}
        onApplied={result => {
          setBulkRegenResult(result)
          loadGames()
        }}
      />
      )}
    </div>
  )
}
