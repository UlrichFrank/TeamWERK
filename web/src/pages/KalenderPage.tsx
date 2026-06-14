import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Home, Plane, Calendar, CalendarDays, Plus, Dumbbell, RefreshCw, Check, X, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { getEventColors } from '../lib/eventColors'
import { buildTeamShortNames, TeamForName } from '../lib/teamName'
import { useAuth, hasFunction } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'

import TrainingEditModal from '../components/TrainingEditModal'
import GameEditModal from '../components/GameEditModal'
import EventInfoModal from '../components/EventInfoModal'
import VenuePicker, { Venue as VenueType } from '../components/VenuePicker'
import RegenSummaryCard, { RegenSummary } from '../components/RegenSummaryCard'

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
}

interface Game {
  id: number
  date: string
  time: string
  end_time?: string | null
  end_date?: string | null
  opponent: string
  teams: Array<{ id: number; name: string }>
  event_type: string
  slot_count: number
  filled_count: number
  total_count: number
  confirmed_count: number
  declined_count: number
  maybe_count: number
  venue?: VenueRef | null
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

function dutyDotColor(filled: number, total: number): string {
  if (total === 0) return 'bg-brand-danger'
  const pct = filled / total
  if (pct >= 0.9) return 'bg-brand-success'
  if (pct >= 0.3) return 'bg-brand-warning'
  return 'bg-brand-danger'
}


function padDate(year: number, month: number, day: number): string {
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

const BTN_SECONDARY = 'border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors'
const INPUT_WIZ = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

function canSeeTeamAbsences(user: ReturnType<typeof useAuth>['user']): boolean {
  if (!user) return false
  return user.role === 'admin' || hasFunction(user, 'trainer') ||
    hasFunction(user, 'vorstand') || hasFunction(user, 'sportliche_leitung')
}

export default function KalenderPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
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
  const [filterTeamId, setFilterTeamId] = useState<number | null>(null)
  const [filterTypes, setFilterTypes] = useState<Set<string>>(new Set(['heim', 'auswärts', 'generisch', 'training']))
  const [showTeamAbsences, setShowTeamAbsences] = useState<boolean>(
    () => sessionStorage.getItem('kalender_show_team_absences') === 'true'
  )
  const compact = useCompactHeader(950)

  const [regenSummary, setRegenSummary] = useState<RegenSummary | null>(null)

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
  const [templates, setTemplates] = useState<any[]>([])
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
  const [gameRsvpOptOut, setGameRsvpOptOut] = useState(0)
  const [gameRsvpRequireReason, setGameRsvpRequireReason] = useState(1)
  // Absence wizard states
  const [absenceForm, setAbsenceForm] = useState<{ member_ids: number[]; type: string; start_date: string; end_date: string; note: string }>({ member_ids: [], type: 'vacation', start_date: '', end_date: '', note: '' })
  const [absencePreviewEvents, setAbsencePreviewEvents] = useState<Array<{ event_type: string; event_id: number; name: string; date: string; pending: boolean }> | null>(null)
  const [absencePreviewLoading, setAbsencePreviewLoading] = useState(false)
  const [absenceChildren, setAbsenceChildren] = useState<Array<{ id: number; name: string }>>([])
  const [absenceSaving, setAbsenceSaving] = useState(false)
  const [absenceError, setAbsenceError] = useState('')
  // Inline edit modal
  const [editingTraining, setEditingTraining] = useState<Training | null>(null)
  const [editingGame, setEditingGame] = useState<Game | null>(null)
  const [infoItem, setInfoItem] = useState<{ type: 'game' | 'training' | 'absence'; game?: Game; training?: Training; absence?: Absence } | null>(null)


  const loadGames = async () => {
    try {
      const r = await api.get('/kalender')
      const data = r.data
      const payload = Array.isArray(data) ? data : (data?.games ?? [])
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
      const r = await api.get(`/training-sessions?from=${from}&to=${to}`)
      setTrainings(Array.isArray(r.data) ? r.data : [])
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
      if (show && canSeeTeamAbsences(user)) {
        url += '&show_team=true'
        if (tid !== null) url += `&team_id=${tid}`
      }
      const r = await api.get(url)
      setAbsences(Array.isArray(r.data) ? r.data : [])
    } catch {
      setAbsences([])
    }
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
            const active = seasons.find((s: any) => s.is_active)
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
  }, [])

  useEffect(() => { loadTrainings(); loadAbsences() }, [year, month]) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { loadAbsences() }, [filterTeamId, showTeamAbsences]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-select the only child once children have loaded — keeps the parent
  // with exactly one linked kid from being forced through a useless selector.
  useEffect(() => {
    if (eventType === 'abwesenheit' && absenceChildren.length === 1 && absenceForm.member_ids.length === 0) {
      setAbsenceForm(f => ({ ...f, member_ids: [absenceChildren[0].id] }))
    }
  }, [eventType, absenceChildren, absenceForm.member_ids.length])

  useLiveUpdates((event) => {
    if (event === 'games') loadGames()
    if (event === 'absences') loadAbsences()
    if (event === 'trainings') loadTrainings()
  })

  const prevMonth = () => month === 0 ? (setMonth(11), setYear(y => y - 1)) : setMonth(m => m - 1)
  const nextMonth = () => month === 11 ? (setMonth(0), setYear(y => y + 1)) : setMonth(m => m + 1)
  const goToToday = () => {
    const today = new Date()
    setYear(today.getFullYear())
    setMonth(today.getMonth())
  }

  const calendarRef = useRef<HTMLDivElement>(null)
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
      isNext ? nextMonth() : prevMonth()
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
      next.has(type) ? next.delete(type) : next.add(type)
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
      const r = await api.post('/kalender', {
        date: selectedDate,
        time: selectedTime,
        end_time: eventType === 'generisch' ? selectedEndTime : undefined,
        end_date: eventType === 'generisch' && selectedEndDate ? selectedEndDate : undefined,
        opponent: selectedOpponent,
        team_ids: selectedTeamIds,
        event_type: eventType,
        template_id: selectedTemplate ?? undefined,
        venue_id: selectedVenueId,
        rsvp_opt_out: gameRsvpOptOut,
        rsvp_require_reason: gameRsvpRequireReason,
        slots: slotsPayload,
      })
      if (r.data?.regen_summary) {
        setRegenSummary(r.data.regen_summary)
      }
      await loadGames()
      closeDialog()
    } catch {
      setCreateError('Event konnte nicht angelegt werden. Ist eine aktive Saison vorhanden?')
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
        rsvp_opt_out: gameRsvpOptOut,
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
        rsvp_opt_out: gameRsvpOptOut,
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
      const dateParam = eventType === 'heim' ? `&date=${selectedDate}` : ''
      const endTimeParam = eventType === 'generisch' ? `&end_time=${selectedEndTime}` : ''
      const r = await api.get(`/duty-templates/${selectedTemplate}/preview?time=${selectedTime}${dateParam}${endTimeParam}`)
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
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const loadAbsenceChildren = async () => {
    try {
      const r = await api.get('/profile/me')
      const kinder: Array<{ id: number; first_name: string; last_name: string }> = r.data?.children ?? []
      setAbsenceChildren(kinder.map(k => ({ id: k.id, name: `${k.first_name} ${k.last_name}` })))
    } catch {}
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
    setGameRsvpOptOut(0)
    setGameRsvpRequireReason(1)
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
    null
  )

  const canEdit = Boolean(user && (user.role === 'admin' || hasFunction(user, 'trainer') || hasFunction(user, 'vorstand') || hasFunction(user, 'sportliche_leitung')))
  const canCreateAbsence = Boolean(user && (hasFunction(user, 'spieler') || user.isParent))

  return (
    <div>
      {regenSummary && (
        <RegenSummaryCard summary={regenSummary} onDismiss={() => setRegenSummary(null)} />
      )}
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold shrink-0">Kalender</h1>
        <select
          value={filterTeamId ?? ''}
          onChange={e => setFilterTeamId(e.target.value === '' ? null : Number(e.target.value))}
          className="border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow shrink-0 max-w-[6rem]"
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
        </div>
        {canSeeTeamAbsences(user) && (
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
            <CalendarDays className="w-3.5 h-3.5" />
            {!compact && <span>Abwesenheiten</span>}
          </button>
        )}
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
            aria-label="Event"
            className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium bg-brand-yellow text-brand-black border border-brand-yellow hover:bg-brand-black hover:text-brand-yellow transition-colors shrink-0 ${compact ? 'px-2' : 'px-3'}`}
          >
            <Plus className="w-3.5 h-3.5" />
            {!compact && <span>{canEdit ? 'Event' : 'Abwesenheit'}</span>}
          </button>
        )}
      </div>

      {/* Month navigation */}
      <div className="flex items-center gap-4 mb-4">
        <button onClick={prevMonth} className="p-2 hover:bg-brand-border-subtle rounded-lg transition-colors text-brand-text">◀</button>
        <span className="text-lg font-semibold w-44 text-center">{MONTHS[month]} {year}</span>
        <button onClick={nextMonth} className="p-2 hover:bg-brand-border-subtle rounded-lg transition-colors text-brand-text">▶</button>
        <button
          onClick={goToToday}
          disabled={year === now.getFullYear() && month === now.getMonth()}
          aria-label="Heute"
          title="Heute"
          className="rounded-md p-2 bg-brand-yellow text-brand-black hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <CalendarDays className="w-4 h-4" />
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
                {dayGames.map(g => (
                  <button
                    key={g.id}
                    onPointerDown={e => e.stopPropagation()}
                    onClick={() => setInfoItem({ type: 'game', game: { ...g, teams: g.teams.map(t => ({ id: t.id, name: shortNames.get(t.id) ?? t.name })) } })}
                    title={`${g.teams.length > 1 ? 'Mehrere Teams' : (shortNames.get(g.teams[0]?.id) ?? g.teams[0]?.name ?? '?')} · ${g.opponent || '–'} · ${g.time}`}
                    className={`w-full text-left mb-1 p-1.5 rounded-md text-xs transition-colors border ${getEventColors(g.event_type).pill}`}
                  >
                    <div className="flex items-center gap-1 mb-0.5">
                      {g.event_type === 'heim'
                        ? <Home className="w-3 h-3 text-brand-text-muted shrink-0" />
                        : g.event_type === 'auswärts'
                        ? <Plane className="w-3 h-3 text-brand-text-muted shrink-0" />
                        : <Calendar className="w-3 h-3 text-brand-text-muted shrink-0" />}
                      <span className="hidden @tile-sm:inline font-semibold truncate text-brand-text">
                        {g.teams.length > 1 ? 'Mehrere' : (shortNames.get(g.teams[0]?.id) ?? '?')}
                      </span>
                    </div>
                    <div className="hidden @tile-md:block truncate text-brand-text-muted leading-tight">
                      {g.opponent || '–'}
                    </div>
                    <div className="flex items-center gap-1 text-brand-text-subtle leading-tight">
                      <span>{g.time}</span>
                      {g.slot_count > 0 && (
                        <div className={`hidden @tile-sm:inline-flex w-1.5 h-1.5 rounded-full flex-shrink-0 ${dutyDotColor(g.filled_count, g.total_count)}`} />
                      )}
                    </div>
                  </button>
                ))}
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
                      <span className="hidden @tile-sm:inline font-semibold truncate text-brand-text">
                        {shortNames.get(t.team_id) ?? (t.title || 'Training')}
                      </span>
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
                  {user && (user.role === 'admin' || hasFunction(user, 'trainer') || hasFunction(user, 'sportliche_leitung')) && (
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
                    <input type="date" value={selectedDate} onChange={e => setSelectedDate(e.target.value)} className={INPUT_WIZ} />
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
                        min={selectedDate || undefined} className={INPUT_WIZ} />
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
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-2">
                      {eventType === 'generisch' ? 'Mannschaften *' : 'Mannschaft *'}
                    </label>
                    {eventType === 'generisch' ? (
                      <div className="space-y-2">
                        {teams.filter(t => t.is_active).map(t => (
                          <label key={t.id} className="flex items-center gap-2">
                            <input type="checkbox" checked={selectedTeamIds.includes(t.id)}
                              onChange={e => {
                                if (e.target.checked) {
                                  setSelectedTeamIds([...selectedTeamIds, t.id])
                                } else {
                                  setSelectedTeamIds(selectedTeamIds.filter(id => id !== t.id))
                                }
                              }} className="rounded accent-brand-yellow" />
                            <span className="text-sm text-brand-text">{shortNames.get(t.id) ?? t.name}</span>
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
                  <div className="space-y-2 pt-2 border-t border-brand-border-subtle">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={gameRsvpOptOut === 1}
                        onChange={e => setGameRsvpOptOut(e.target.checked ? 1 : 0)}
                        className="w-4 h-4 accent-brand-yellow" />
                      <span className="text-sm text-brand-text">Alle Spieler standardmäßig zugesagt (Opt-Out)</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={gameRsvpRequireReason === 1}
                        onChange={e => setGameRsvpRequireReason(e.target.checked ? 1 : 0)}
                        className="w-4 h-4 accent-brand-yellow" />
                      <span className="text-sm text-brand-text">Begründung bei Absage erforderlich</span>
                    </label>
                  </div>
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
                    <input type="date" value={selectedDate} onChange={e => setSelectedDate(e.target.value)} className={INPUT_WIZ} />
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
                  <div className="space-y-2 pt-2 border-t border-brand-border-subtle">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={gameRsvpOptOut === 1}
                        onChange={e => setGameRsvpOptOut(e.target.checked ? 1 : 0)}
                        className="w-4 h-4 accent-brand-yellow" />
                      <span className="text-sm text-brand-text">Alle Spieler standardmäßig zugesagt (Opt-Out)</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={gameRsvpRequireReason === 1}
                        onChange={e => setGameRsvpRequireReason(e.target.checked ? 1 : 0)}
                        className="w-4 h-4 accent-brand-yellow" />
                      <span className="text-sm text-brand-text">Begründung bei Absage erforderlich</span>
                    </label>
                  </div>
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
                    <input type="date" value={seriesValidFrom} onChange={e => setSeriesValidFrom(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Gültig bis *</label>
                    <input type="date" value={seriesValidUntil} onChange={e => setSeriesValidUntil(e.target.value)} className={INPUT_WIZ} />
                  </div>
                  <div className="space-y-2 pt-2 border-t border-brand-border-subtle">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={gameRsvpOptOut === 1}
                        onChange={e => setGameRsvpOptOut(e.target.checked ? 1 : 0)}
                        className="w-4 h-4 accent-brand-yellow" />
                      <span className="text-sm text-brand-text">Alle Spieler standardmäßig zugesagt (Opt-Out)</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="checkbox" checked={gameRsvpRequireReason === 1}
                        onChange={e => setGameRsvpRequireReason(e.target.checked ? 1 : 0)}
                        className="w-4 h-4 accent-brand-yellow" />
                      <span className="text-sm text-brand-text">Begründung bei Absage erforderlich</span>
                    </label>
                  </div>
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
                        onChange={e => setAbsenceForm(f => ({ ...f, start_date: e.target.value }))}
                        className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-brand-text-muted mb-1">Bis</label>
                      <input
                        type="date"
                        value={absenceForm.end_date}
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
                  return filteredTemplates.length === 0 ? (
                    <div className="text-center py-6">
                      <p className="text-brand-text-muted">Keine passende Vorlage — Event wird ohne Dienste angelegt.</p>
                    </div>
                  ) : (
                    <div className="space-y-2 mb-4">
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
                      const filteredTemplates = templates.filter(t => t.template_type === eventType)
                      if (selectedTemplate) {
                        handleFetchPreview()
                      } else if (filteredTemplates.length === 0) {
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
                    onClick={() => doCreateGame([])}
                    disabled={creating}
                    className="border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text-muted hover:bg-brand-border-subtle hover:text-brand-text transition-colors disabled:opacity-50"
                  >Ohne Dienste</button>
                  <button
                    onClick={() => doCreateGame(preview.filter((_, i) => selectedSlotIndices.has(i)))}
                    disabled={creating}
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
          onDienste={infoItem.type === 'game' && infoItem.game
            ? () => { setInfoItem(null); navigate(`/kalender/${infoItem.game!.id}`) }
            : undefined}
          canEditAbsence={infoItem.type === 'absence' && !!infoItem.absence && infoItem.absence.can_edit}
          onAbsenceChanged={() => { loadAbsences(); setInfoItem(null) }}
        />
      )}
    </div>
  )
}
