import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Home, MapPin, Calendar, Plus, Dumbbell, RefreshCw, Check, X } from 'lucide-react'
import { api } from '../lib/api'
import { getEventColors } from '../lib/eventColors'
import { useAuth, hasFunction } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'

import TrainingEditModal from '../components/TrainingEditModal'

interface Training {
  id: number
  title: string
  date: string
  start_time: string
  end_time: string
  location: string
  status: 'active' | 'cancelled'
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
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
  opponent: string
  teams: Array<{ id: number; name: string }>
  event_type: string
  slot_count: number
  filled_count: number
  total_count: number
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

function buildTeamShortNames(teams: Team[]): Map<number, string> {
  const gLetter = (g: string) => g === 'f' ? 'w' : g === 'mixed' ? 'g' : 'm'
  const aLetter = (a: string) => { const m = a.match(/^([A-F])/i); return m ? m[1].toUpperCase() : a.charAt(0) }

  const groups = new Map<string, Team[]>()
  for (const t of teams) {
    const key = `${t.gender}|${t.age_class}`
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(t)
  }

  const result = new Map<number, string>()
  for (const [, group] of groups) {
    const multi = group.length > 1
    for (const t of group) {
      const base = `${gLetter(t.gender)}${aLetter(t.age_class)}`
      if (multi) {
        const numMatch = t.name.match(/\d+$/)
        result.set(t.id, `${base}${numMatch ? numMatch[0] : ''}`)
      } else {
        result.set(t.id, base)
      }
    }
  }
  return result
}

function padDate(year: number, month: number, day: number): string {
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

const BTN_SECONDARY = 'border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors'
const INPUT_WIZ = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

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
  const [teams, setTeams] = useState<Team[]>([])
  const [loading, setLoading] = useState(true)
  const [filterTeamId, setFilterTeamId] = useState<number | null>(null)
  const [filterTypes, setFilterTypes] = useState<Set<string>>(new Set(['heim', 'auswärts', 'generisch', 'training']))
  const compact = useCompactHeader(950)

  // Day-regen dialog
  const [showDayRegen, setShowDayRegen] = useState(false)
  const [dayRegenDate, setDayRegenDate] = useState('')
  const [dayRegenLoading, setDayRegenLoading] = useState(false)
  const [dayRegenResult, setDayRegenResult] = useState<{
    games: Array<{ game_id: number; slots_created: number; kept_slots: number; skipped?: boolean }>
    conflicts: Array<{ duty_type_id: number; event_time: string; game_ids: number[] }>
  } | null>(null)
  const [dayRegenError, setDayRegenError] = useState<string | null>(null)

  // Wizard dialog
  const [showCreate, setShowCreate] = useState(false)
  const [wizardStep, setWizardStep] = useState(1)
  const [eventType, setEventType] = useState<'heim' | 'auswärts' | 'generisch' | 'training' | 'serie' | ''>('')
  const [selectedDate, setSelectedDate] = useState('')
  const [selectedTime, setSelectedTime] = useState('15:00')
  const [selectedOpponent, setSelectedOpponent] = useState('')
  const [selectedTeamIds, setSelectedTeamIds] = useState<number[]>([])
  const [selectedEndTime, setSelectedEndTime] = useState('16:00')
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
  const [trainingLocation, setTrainingLocation] = useState('')
  const [seriesWeekday, setSeriesWeekday] = useState(1)
  const [seriesValidFrom, setSeriesValidFrom] = useState('')
  const [seriesValidUntil, setSeriesValidUntil] = useState('')
  // Inline edit modal
  const [editingTraining, setEditingTraining] = useState<Training | null>(null)

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

  useEffect(() => {
    const loadInitialData = async () => {
      await Promise.all([
        loadGames(),
        loadTrainings(),
        api.get('/teams')
          .then(r => setTeams(Array.isArray(r.data) ? r.data : (r.data?.teams ?? [])))
          .catch(() => setTeams([])),
        api.get('/admin/seasons')
          .then(r => {
            const seasons = Array.isArray(r.data) ? r.data : []
            const active = seasons.find((s: any) => s.is_active)
            if (active) setActiveSeasonId(active.id)
          })
          .catch(() => {}),
      ])
      setLoading(false)
    }
    loadInitialData()
  }, [])

  useEffect(() => { loadTrainings() }, [year, month])

  useLiveUpdates((event) => { if (event === 'games') loadGames() })

  const prevMonth = () => month === 0 ? (setMonth(11), setYear(y => y - 1)) : setMonth(m => m - 1)
  const nextMonth = () => month === 11 ? (setMonth(0), setYear(y => y + 1)) : setMonth(m => m + 1)

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
    setWizardStep(1)
    loadTemplates()
  }

  const toggleType = (type: string) => {
    setFilterTypes(prev => {
      const next = new Set(prev)
      next.has(type) ? next.delete(type) : next.add(type)
      return next
    })
  }

  const shortNames = useMemo(() => buildTeamShortNames(teams), [teams])

  const safeGames = Array.isArray(games) ? games : []
  const monthGames = safeGames.filter(g => {
    const y = parseInt(g.date.slice(0, 4))
    const m = parseInt(g.date.slice(5, 7)) - 1
    if (y !== year || m !== month) return false
    if (!filterTypes.has(g.event_type)) return false
    if (filterTeamId !== null && !g.teams.some(t => t.id === filterTeamId)) return false
    return true
  })

  const gamesByDate: Record<string, Game[]> = {}
  for (const g of monthGames) {
    const key = g.date.slice(0, 10)
    if (!gamesByDate[key]) gamesByDate[key] = []
    gamesByDate[key].push(g)
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

  const doCreateGame = async (slots: SlotPreview[]) => {
    setCreating(true)
    setCreateError(null)
    try {
      await api.post('/admin/kalender', {
        date: selectedDate,
        time: selectedTime,
        end_time: eventType === 'generisch' ? selectedEndTime : undefined,
        opponent: selectedOpponent,
        team_ids: selectedTeamIds,
        event_type: eventType,
        template_id: selectedTemplate ?? undefined,
        slots: slots.map(s => ({
          duty_type_id: s.duty_type_id,
          event_time: s.event_time,
          slots_count: s.slots_count,
          role_desc: s.role_desc,
        })),
      })
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
        location: trainingLocation,
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
        location: trainingLocation,
        day_of_week: seriesWeekday,
        start_time: trainingStartTime,
        end_time: trainingEndTime,
        valid_from: seriesValidFrom,
        valid_until: seriesValidUntil,
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
      const r = await api.get('/admin/duty-templates')
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
      const r = await api.get(`/admin/duty-templates/${selectedTemplate}/preview?time=${selectedTime}${dateParam}${endTimeParam}`)
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

  const openDayRegen = (dateStr: string) => {
    setDayRegenDate(dateStr)
    setDayRegenResult(null)
    setDayRegenError(null)
    setShowDayRegen(true)
  }

  const closeDayRegen = () => {
    setShowDayRegen(false)
    setDayRegenDate('')
    setDayRegenResult(null)
    setDayRegenError(null)
  }

  const doRegenDay = async () => {
    setDayRegenLoading(true)
    setDayRegenError(null)
    setDayRegenResult(null)
    try {
      const r = await api.post(`/admin/kalender/regenerate-day?date=${dayRegenDate}`)
      setDayRegenResult(r.data)
      await loadGames()
    } catch {
      setDayRegenError('Generierung fehlgeschlagen. Ist eine aktive Saison vorhanden?')
    } finally {
      setDayRegenLoading(false)
    }
  }

  const closeDialog = () => {
    setShowCreate(false)
    setWizardStep(1)
    setEventType('')
    setSelectedDate('')
    setSelectedTime('15:00')
    setSelectedEndTime('16:00')
    setSelectedOpponent('')
    setSelectedTeamIds([])
    setSelectedTemplate(null)
    setPreview([])
    setSelectedSlotIndices(new Set())
    setCreateError(null)
    setTrainingTitle('')
    setTrainingStartTime('18:00')
    setTrainingEndTime('19:30')
    setTrainingLocation('')
    setSeriesWeekday(1)
    setSeriesValidFrom('')
    setSeriesValidUntil('')
  }

  useEscapeKey(
    showDayRegen ? () => setShowDayRegen(false) :
    showCreate ? closeDialog :
    editingTraining ? () => setEditingTraining(null) :
    null
  )

  return (
    <div>
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold shrink-0">Kalender</h1>
        <div className="flex items-center gap-1.5 flex-1 flex-nowrap min-w-0">
          <select
            value={filterTeamId ?? ''}
            onChange={e => setFilterTeamId(e.target.value === '' ? null : Number(e.target.value))}
            className="border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow min-w-0 shrink"
          >
            <option value="">Alle Teams</option>
            {teams.map(t => (
              <option key={t.id} value={t.id}>{t.name}</option>
            ))}
          </select>
          {([
            ['heim',      'Heim',       <Home className="w-3.5 h-3.5" />],
            ['auswärts',  'Auswärts',   <MapPin className="w-3.5 h-3.5" />],
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
        {user && (user.role === 'admin' || hasFunction(user, 'vorstand') || hasFunction(user, 'trainer')) && (
          <button
            onClick={() => setShowCreate(true)}
            aria-label="Event"
            className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium bg-brand-yellow text-brand-black border border-brand-yellow hover:bg-brand-black hover:text-brand-yellow transition-colors shrink-0 ${compact ? 'px-2' : 'px-3'}`}
          >
            <Plus className="w-3.5 h-3.5" />
            {!compact && <span>Event</span>}
          </button>
        )}
      </div>

      {/* Month navigation */}
      <div className="flex items-center gap-4 mb-4">
        <button onClick={prevMonth} className="p-2 hover:bg-brand-border-subtle rounded-lg transition-colors text-brand-text">◀</button>
        <span className="text-lg font-semibold w-44 text-center">{MONTHS[month]} {year}</span>
        <button onClick={nextMonth} className="p-2 hover:bg-brand-border-subtle rounded-lg transition-colors text-brand-text">▶</button>
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
            const isToday = dateStr === todayStr
            const canEdit = user && (user.role === 'admin' || hasFunction(user, 'vorstand') || hasFunction(user, 'trainer'))
            const canRegen = canEdit && dayGames.length > 0
            return (
              <div key={day} className="@container group min-h-[90px] p-1.5 border-r border-b border-brand-border-subtle">
                <div className="flex items-center justify-between mb-1">
                  <span className={`text-xs leading-none flex items-center justify-center ${isToday ? 'font-bold w-5 h-5 rounded-full bg-brand-yellow text-brand-black' : 'text-brand-text-subtle'}`}>{day}</span>
                  {canEdit && (
                    <button
                      onPointerDown={e => e.stopPropagation()}
                      onClick={e => { e.stopPropagation(); openWizardWithDate(dateStr) }}
                      className="opacity-0 group-hover:opacity-100 transition-opacity p-0.5 rounded text-brand-text-subtle hover:text-brand-text hover:bg-brand-border-subtle"
                      title="Event anlegen"
                    >
                      <Plus className="w-3 h-3" />
                    </button>
                  )}
                </div>
                {dayGames.map(g => (
                  <button
                    key={g.id}
                    onPointerDown={e => e.stopPropagation()}
                    onClick={() => navigate(`/kalender/${g.id}`)}
                    title={`${g.teams.length > 1 ? 'Mehrere Teams' : (shortNames.get(g.teams[0]?.id) ?? g.teams[0]?.name ?? '?')} · ${g.opponent || '–'} · ${g.time}`}
                    className={`w-full text-left mb-1 p-1.5 rounded-md text-xs transition-colors border ${getEventColors(g.event_type).pill}`}
                  >
                    <div className="flex items-center gap-1 mb-0.5">
                      {g.event_type === 'heim'
                        ? <Home className="w-3 h-3 text-brand-text-muted shrink-0" />
                        : g.event_type === 'auswärts'
                        ? <MapPin className="w-3 h-3 text-brand-text-muted shrink-0" />
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
                    onClick={() => {
                      if (user && (user.role === 'admin' || hasFunction(user, 'trainer'))) {
                        setEditingTraining(t)
                      } else {
                        navigate(`/trainings/${t.id}`)
                      }
                    }}
                    className={`w-full text-left mb-1 p-1.5 rounded-md text-xs transition-colors border ${
                      t.status === 'cancelled'
                        ? 'bg-white/50 border-brand-border-subtle opacity-50 line-through'
                        : getEventColors('training').pill
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
                      <span className="hidden @tile-sm:inline-flex items-center gap-0.5 text-green-600">
                        <Check className="w-2.5 h-2.5" />{t.confirmed_count}
                      </span>
                      <span className="hidden @tile-sm:inline-flex items-center gap-0.5 text-brand-danger">
                        <X className="w-2.5 h-2.5" />{t.declined_count}
                      </span>
                    </div>
                  </button>
                ))}
                {canRegen && (
                  <button
                    onPointerDown={e => e.stopPropagation()}
                    onClick={() => openDayRegen(dateStr)}
                    className="w-full mt-0.5 text-center text-[10px] text-brand-blue hover:underline leading-tight py-0.5"
                  >
                    Dienste generieren
                  </button>
                )}
              </div>
            )
          })}
        </div>
      </div>
      </div>

      {!loading && monthGames.length === 0 && (
        <p className="text-brand-text-subtle text-center mt-10 text-sm">Keine Heimspiele in diesem Monat</p>
      )}

      {/* Day Regeneration Dialog */}
      {showDayRegen && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-md shadow-2xl max-h-[90vh] overflow-y-auto">
            <h2 className="text-lg font-bold mb-1 text-brand-text">Dienste generieren</h2>
            <p className="text-sm text-brand-text-muted mb-4">{dayRegenDate}</p>

            <div className="space-y-2 mb-4">
              {(gamesByDate[dayRegenDate] ?? []).map(g => (
                <div key={g.id} className="p-3 border border-brand-border-subtle rounded-lg bg-brand-surface-card text-sm">
                  <div className="font-semibold text-brand-text">{g.time} — {g.teams.length > 1 ? 'Mehrere Teams' : g.teams[0]?.name}</div>
                  <div className="text-brand-text-muted text-xs">
                    {g.event_type === 'generisch' ? (g.opponent || '–') : `Team vs ${g.opponent || '–'}`} · {g.event_type}
                  </div>
                </div>
              ))}
            </div>

            {dayRegenResult && (
              <div className="mb-4 space-y-2">
                <div className="p-3 bg-brand-success-light border border-brand-success/30 rounded-lg text-sm">
                  <div className="font-semibold text-brand-success mb-1">Generierung abgeschlossen</div>
                  {dayRegenResult.games.map(gr => (
                    <div key={gr.game_id} className="text-brand-success text-xs">
                      {gr.skipped
                        ? `Spiel #${gr.game_id}: kein Template — übersprungen`
                        : `Spiel #${gr.game_id}: ${gr.slots_created} Dienste erstellt, ${gr.kept_slots} behalten`}
                    </div>
                  ))}
                </div>
                {dayRegenResult.conflicts.length > 0 && (
                  <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm">
                    <div className="font-semibold text-brand-danger mb-1">Konflikte erkannt</div>
                    <p className="text-brand-danger text-xs mb-1">
                      Gleicher Diensttyp zur gleichen Zeit bei mehreren Spielen — bitte Optimierungsregeln prüfen.
                    </p>
                    {dayRegenResult.conflicts.map((c, i) => (
                      <div key={i} className="text-brand-danger text-xs">
                        {c.event_time} · Diensttyp #{c.duty_type_id} bei Spielen {c.game_ids.join(', ')}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {dayRegenError && (
              <div className="mb-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                {dayRegenError}
              </div>
            )}

            <div className="flex gap-2 pt-2">
              <button onClick={closeDayRegen} className={BTN_SECONDARY}>
                {dayRegenResult ? 'Schließen' : 'Abbrechen'}
              </button>
              {!dayRegenResult && (
                <button
                  onClick={doRegenDay}
                  disabled={dayRegenLoading}
                  className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                >
                  {dayRegenLoading ? 'Generiere…' : 'Generieren'}
                </button>
              )}
            </div>
          </div>
        </div>
      )}

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
                        setWizardStep(2)
                      }}
                      className="w-full p-4 border-2 border-brand-border rounded-lg text-left hover:bg-brand-border-subtle hover:border-brand-yellow transition-colors"
                    >
                      <div className="font-semibold flex items-center gap-2 text-brand-text">
                        {type === 'heim' && <><Home className="w-4 h-4" /> Heimspiel</>}
                        {type === 'auswärts' && <><MapPin className="w-4 h-4" /> Auswärtsspiel</>}
                        {type === 'generisch' && <><Calendar className="w-4 h-4" /> Sonstiges Event</>}
                      </div>
                      <div className="text-xs text-brand-text-muted mt-1">
                        {type === 'heim' && 'Heimspiel gegen eine Mannschaft'}
                        {type === 'auswärts' && 'Auswärtsspiel gegen eine Mannschaft'}
                        {type === 'generisch' && 'Event für mehrere Mannschaften'}
                      </div>
                    </button>
                  ))}
                  {user && (user.role === 'admin' || hasFunction(user, 'trainer')) && (
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
                            <span className="text-sm text-brand-text">{t.name}</span>
                          </label>
                        ))}
                      </div>
                    ) : (
                      <select value={selectedTeamIds[0] ?? ''} onChange={e => setSelectedTeamIds(e.target.value ? [Number(e.target.value)] : [])}
                        className={INPUT_WIZ}>
                        <option value="">Auswählen…</option>
                        {teams.filter(t => t.is_active).map(t => (
                          <option key={t.id} value={t.id}>{t.name}</option>
                        ))}
                      </select>
                    )}
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
                    disabled={!selectedDate || selectedTeamIds.length === 0}
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
                    <input type="text" value={trainingLocation} onChange={e => setTrainingLocation(e.target.value)}
                      placeholder="Sporthalle…" className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Mannschaft *</label>
                    <select value={selectedTeamIds[0] ?? ''} onChange={e => setSelectedTeamIds(e.target.value ? [Number(e.target.value)] : [])}
                      className={INPUT_WIZ}>
                      <option value="">Auswählen…</option>
                      {teams.filter(t => t.is_active).map(t => (
                        <option key={t.id} value={t.id}>{t.name}</option>
                      ))}
                    </select>
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
                    <input type="text" value={trainingLocation} onChange={e => setTrainingLocation(e.target.value)}
                      placeholder="Sporthalle…" className={INPUT_WIZ} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Mannschaft *</label>
                    <select value={selectedTeamIds[0] ?? ''} onChange={e => setSelectedTeamIds(e.target.value ? [Number(e.target.value)] : [])}
                      className={INPUT_WIZ}>
                      <option value="">Auswählen…</option>
                      {teams.filter(t => t.is_active).map(t => (
                        <option key={t.id} value={t.id}>{t.name}</option>
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
      {editingTraining && (
        <TrainingEditModal
          session={editingTraining}
          teamName={teams.find(t => t.id === editingTraining.team_id)?.name}
          onClose={() => setEditingTraining(null)}
          onSaved={() => { loadTrainings(); setEditingTraining(null) }}
        />
      )}
    </div>
  )
}
