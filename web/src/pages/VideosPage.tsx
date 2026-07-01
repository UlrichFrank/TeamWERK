import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Upload, Video } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useMediaQuery } from '../lib/useMediaQuery'
import MobileCard from '../components/MobileCard'
import VideoStatusPill from '../components/VideoStatusPill'
import { fmtDuration, fmtVideoDate } from '../lib/videoFormat'
import { buildTeamShortNames } from '../lib/teamName'

interface VideoItem {
  id: number
  title: string
  description?: string | null
  team_id: number
  team_name: string
  season_id: number
  game_id?: number | null
  status: string
  duration_sec?: number | null
  created_by: number
  created_at: string
  ready_at?: string | null
}

interface VideoListResponse {
  items: VideoItem[]
  total: number
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

const PAGE_SIZE = 50

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'Alle Status' },
  { value: 'queued', label: 'In Warteschlange' },
  { value: 'processing', label: 'Wird verarbeitet' },
  { value: 'ready', label: 'Bereit' },
  { value: 'failed', label: 'Fehlgeschlagen' },
]

export default function VideosPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const isMobile = useMediaQuery('(max-width: 639px)')

  // Upload nur für Trainer/sportl. Leitung/Vorstand/Admin (Server erzwingt zusätzlich).
  const canUpload = !!user && (
    user.role === 'admin' ||
    user.clubFunctions.some(f => f === 'trainer' || f === 'sportliche_leitung' || f === 'vorstand')
  )

  const [teams, setTeams] = useState<Team[]>([])
  const [teamFilter, setTeamFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')

  const [items, setItems] = useState<VideoItem[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.get<Team[]>('/teams')
      .then(r => setTeams(Array.isArray(r.data) ? r.data : []))
      .catch(() => setTeams([]))
  }, [])

  const fetchPage = useCallback(async (nextOffset: number, replace: boolean) => {
    setLoading(true)
    setError('')
    try {
      const params = new URLSearchParams()
      if (teamFilter) params.append('team_id', teamFilter)
      if (statusFilter) params.append('status', statusFilter)
      params.append('limit', String(PAGE_SIZE))
      params.append('offset', String(nextOffset))
      const res = await api.get<VideoListResponse>(`/videos?${params}`)
      setTotal(res.data.total)
      setOffset(nextOffset + res.data.items.length)
      setItems(prev => replace ? res.data.items : [...prev, ...res.data.items])
    } catch {
      setError('Videos konnten nicht geladen werden.')
    } finally {
      setLoading(false)
    }
  }, [teamFilter, statusFilter])

  // Filterwechsel → Liste von vorne laden.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    fetchPage(0, true)
  }, [fetchPage])

  // Live-Updates: bei jeder Video-Mutation Liste neu laden.
  useLiveUpdates((event) => {
    if (event === 'video-queued' || event === 'video-ready' || event === 'video-updated' || event === 'video-deleted') {
      fetchPage(0, true)
    }
  })

  const hasMore = items.length < total
  const activeTeams = useMemo(() => teams.filter(t => t.is_active), [teams])
  const shortNames = useMemo(() => buildTeamShortNames(activeTeams), [activeTeams])

  return (
    <div>
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Videos</h1>
          <div className="flex flex-col sm:flex-row gap-2">
            <select
              value={teamFilter}
              onChange={e => setTeamFilter(e.target.value)}
              aria-label="Team filtern"
              className="border border-brand-border rounded-md px-2 py-2.5 sm:py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-full sm:w-24 sm:shrink-0"
            >
              <option value="">Teams</option>
              {activeTeams.map(t => (
                <option key={t.id} value={t.id}>{shortNames.get(t.id) ?? t.name}</option>
              ))}
            </select>
            <select
              value={statusFilter}
              onChange={e => setStatusFilter(e.target.value)}
              aria-label="Status filtern"
              className="border border-brand-border rounded-md px-2 py-2.5 sm:py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-full sm:w-auto sm:shrink-0"
            >
              {STATUS_OPTIONS.map(o => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
            {canUpload && (
              <button
                onClick={() => navigate('/videos/upload')}
                className="inline-flex items-center justify-center gap-1 rounded-md px-3 py-2.5 sm:py-1.5 text-xs font-medium bg-brand-yellow text-brand-black border border-brand-yellow hover:bg-brand-black hover:text-brand-yellow transition-colors sm:shrink-0"
              >
                <Upload className="w-3.5 h-3.5" />
                Video hochladen
              </button>
            )}
          </div>
        </div>
      </div>

      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4">
          {error}
        </div>
      )}

      {items.length === 0 && !loading && !error && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 text-sm text-brand-text-muted flex items-center gap-2">
          <Video className="w-5 h-5 text-brand-text-subtle" />
          Keine Videos vorhanden.
        </div>
      )}

      {/* Mobile: Card-Layout */}
      {isMobile ? (
        <div>
          {items.map(v => (
            <MobileCard
              key={v.id}
              title={v.title}
              subtitle={`${v.team_name} · ${fmtVideoDate(v.created_at)}`}
              onClick={() => navigate(`/videos/${v.id}`)}
            >
              <div className="flex items-center justify-between">
                <VideoStatusPill status={v.status} />
                <span className="text-brand-text-muted">{fmtDuration(v.duration_sec)}</span>
              </div>
            </MobileCard>
          ))}
        </div>
      ) : (
        items.length > 0 && (
          <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Titel</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Team</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Datum</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Dauer</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-subtle">
                {items.map(v => (
                  <tr
                    key={v.id}
                    className="hover:bg-brand-table-select transition-colors cursor-pointer"
                    onClick={() => navigate(`/videos/${v.id}`)}
                  >
                    <td className="px-4 py-3 font-medium text-brand-text">{v.title}</td>
                    <td className="px-4 py-3 text-brand-text-muted">{v.team_name}</td>
                    <td className="px-4 py-3 text-brand-text-muted">{fmtVideoDate(v.created_at)}</td>
                    <td className="px-4 py-3 text-brand-text-muted">{fmtDuration(v.duration_sec)}</td>
                    <td className="px-4 py-3"><VideoStatusPill status={v.status} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}

      {hasMore && (
        <div className="flex justify-center mt-6">
          <button
            onClick={() => fetchPage(offset, false)}
            disabled={loading}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {loading ? 'Lädt…' : 'Mehr laden'}
          </button>
        </div>
      )}
    </div>
  )
}
