import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { AlertTriangle, Clock, Pencil, Trash2, X } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import VideoStatusPill from '../components/VideoStatusPill'
import CastButton from '../components/CastButton'
import { fmtDuration, fmtVideoDate } from '../lib/videoFormat'

interface VideoDetail {
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
  failure_reason?: string | null
}

interface PlayResponse {
  token: string
  master_url: string
}

interface GameTeam {
  id: number
  name: string
}

interface Game {
  id: number
  date: string
  opponent: string
  teams: GameTeam[]
}

// fmtGameOption bildet die Spiel-Auswahl "DD.MM.YYYY · Gegner" (wie im Upload-Formular).
function fmtGameOption(g: Game): string {
  const d = g.date.slice(0, 10)
  const [y, mo, da] = d.split('-')
  const datePart = y && mo && da ? `${da}.${mo}.${y}` : d
  return `${datePart} · ${g.opponent || 'Spiel'}`
}

function VideoPlayer({ id }: { id: number }) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const [error, setError] = useState('')
  const [unsupported, setUnsupported] = useState(false)
  // masterURL wird für Chromecast-Wurf gebraucht — CastButton übergibt die
  // komplette URL inkl. ?st=-Token an den Receiver, der direkt vom Server holt.
  const [masterURL, setMasterURL] = useState('')

  useEffect(() => {
    let cancelled = false
    // hls.js wird dynamisch geladen, damit es nur auf der Detailseite ins Bundle kommt.
    let destroy: (() => void) | undefined

    async function setup() {
      let play: PlayResponse
      try {
        const res = await api.get<PlayResponse>(`/videos/${id}/play`)
        play = res.data
      } catch {
        if (!cancelled) setError('Wiedergabe-Token konnte nicht geladen werden.')
        return
      }
      const video = videoRef.current
      if (!video || cancelled) return

      const { default: Hls } = await import('hls.js')
      if (cancelled) return

      // StreamTokenMiddleware verlangt ?st=<token> auf der Master-Playlist und
      // jeder Rendition-Anfrage. master_url kommt vom Backend ohne Query —
      // Token clientseitig anhängen; ServeMaster setzt ihn auf die referenzierten
      // Rendition-Playlists fort.
      const sep = play.master_url.includes('?') ? '&' : '?'
      const url = `${play.master_url}${sep}st=${encodeURIComponent(play.token)}`
      const masterURL = url
      // Absolute URL für Cast — der Receiver muss den Origin kennen. Backend
      // liefert `/api/videos/…` (Pfad); CastButton kombiniert mit window.location.
      setMasterURL(new URL(url, window.location.origin).toString())

      if (Hls.isSupported()) {
        // Buffer großzügiger als Default (30 s vorwärts): 60 s Vorlauf verkraftet
        // kurze Netz-Aussetzer, ohne dass der Puffer leerläuft. maxBufferHole
        // erlaubt hls.js, kleine Segment-Löcher zu überspringen, statt beim
        // ersten Fehl-Frame in einen Retry-Loop zu kippen (mit sichtbarem
        // Zurückspringen als Symptom).
        const hls = new Hls({
          maxBufferLength: 60,
          maxBufferSize: 30 * 1024 * 1024,
          maxBufferHole: 0.5,
        })
        let mediaRecoveryAttempts = 0
        hls.loadSource(masterURL)
        hls.attachMedia(video)
        hls.on(Hls.Events.ERROR, (_e, data) => {
          if (cancelled) return
          // Nicht-fatale Fehler behandelt hls.js selbst (interner nudgeOffset/
          // nudgeMaxRetry für BUFFER_STALLED, automatisches Retry auf 5xx).
          // Manuelle Eingriffe hier haben in der Vergangenheit doppelte Nudges
          // und Playhead-Sprünge verursacht.
          if (!data.fatal) return
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              hls.startLoad()
              break
            case Hls.ErrorTypes.MEDIA_ERROR:
              // Offizielle zweistufige Recovery (hls.js API-Doku):
              // 1. recoverMediaError(); 2. bei Rückfall swapAudioCodec + recoverMediaError.
              if (mediaRecoveryAttempts === 0) {
                mediaRecoveryAttempts++
                hls.recoverMediaError()
              } else if (mediaRecoveryAttempts === 1) {
                mediaRecoveryAttempts++
                hls.swapAudioCodec()
                hls.recoverMediaError()
              } else {
                setError('Das Video konnte nicht abgespielt werden.')
              }
              break
            default:
              setError('Das Video konnte nicht abgespielt werden.')
          }
        })
        destroy = () => hls.destroy()
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        // Safari / iOS: natives HLS.
        video.src = masterURL
      } else {
        if (!cancelled) setUnsupported(true)
      }
    }

    setup()
    return () => {
      cancelled = true
      if (destroy) destroy()
    }
  }, [id])

  if (unsupported) {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text flex items-start gap-2">
        <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0 text-brand-info" />
        Dein Browser unterstützt das Streaming-Format (HLS) nicht. Bitte öffne das Video in
        einem aktuellen Browser (z.B. Safari, Chrome oder Firefox).
      </div>
    )
  }

  return (
    <div>
      <video
        ref={videoRef}
        controls
        playsInline
        // AirPlay auf Safari explizit erlauben — belt-and-suspenders gegen
        // künftige Safari-Default-Verschärfungen. Zusammen mit CODECS im
        // master.m3u8 (siehe worker.go writeMasterManifest) sorgt das dafür,
        // dass AirPlay auf AppleTV Bild + Ton zeigt (nicht nur Ton).
        {...{ 'x-webkit-airplay': 'allow' }}
        className="w-full rounded-lg bg-brand-black aspect-video"
      />
      {masterURL && (
        <div className="mt-2 flex justify-end">
          <CastButton masterURL={masterURL} />
        </div>
      )}
      {error && (
        <div className="mt-2 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />{error}
        </div>
      )}
    </div>
  )
}

export default function VideoDetailPage() {
  const { id } = useParams<{ id: string }>()
  const videoId = Number(id)
  const navigate = useNavigate()
  const { user } = useAuth()

  const [video, setVideo] = useState<VideoDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)

  const [showEdit, setShowEdit] = useState(false)
  const [showDelete, setShowDelete] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const [editDescription, setEditDescription] = useState('')
  const [editGameId, setEditGameId] = useState('')
  const [games, setGames] = useState<Game[]>([])
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [actionError, setActionError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<VideoDetail>(`/videos/${videoId}`)
      setVideo(res.data)
      setNotFound(false)
    } catch {
      setNotFound(true)
    } finally {
      setLoading(false)
    }
  }, [videoId])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    load()
  }, [load])

  useLiveUpdates((event) => {
    if (event === 'video-ready' || event === 'video-updated' || event === 'video-queued') {
      load()
    }
    if (event === 'video-deleted') {
      // Konnte das angezeigte Video gelöscht worden sein → neu laden (404 → not found).
      load()
    }
  })

  // Verwaltung (Bearbeiten/Löschen): Trainer/sportl. Leitung/Vorstand/Admin.
  // Der Server erzwingt die teamgenaue Prüfung zusätzlich.
  const canManage = !!user && (
    user.role === 'admin' ||
    user.clubFunctions.some(f => f === 'trainer' || f === 'sportliche_leitung' || f === 'vorstand')
  )

  const openEdit = () => {
    if (!video) return
    setEditTitle(video.title)
    setEditDescription(video.description ?? '')
    setEditGameId(video.game_id != null ? String(video.game_id) : '')
    setActionError('')
    setShowEdit(true)
    // Spiele des Video-Teams für den Zuordnungs-Selector laden (clientseitig filtern,
    // wie im Upload-Formular). Fehler still schlucken — der Selector bleibt dann leer.
    api.get<{ items: Game[]; total: number }>('/games?limit=500')
      .then(r => {
        const all = Array.isArray(r.data?.items) ? r.data.items : []
        setGames(all.filter(g => (g.teams ?? []).some(t => t.id === video.team_id)))
      })
      .catch(() => setGames([]))
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editTitle.trim()) return
    setSaving(true)
    setActionError('')
    try {
      await api.patch(`/videos/${videoId}`, {
        title: editTitle.trim(),
        description: editDescription.trim(),
        // Tri-State: gewählte Spiel-ID → Zahl, "Kein Spiel zuordnen" → null (löscht die Zuordnung).
        game_id: editGameId ? Number(editGameId) : null,
      })
      setShowEdit(false)
      load()
    } catch {
      setActionError('Speichern fehlgeschlagen.')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    setDeleting(true)
    setActionError('')
    try {
      await api.delete(`/videos/${videoId}`)
      navigate('/videos')
    } catch {
      setActionError('Löschen fehlgeschlagen.')
      setDeleting(false)
    }
  }

  if (loading) {
    return <div className="text-sm text-brand-text-muted">Laden…</div>
  }

  if (notFound || !video) {
    return (
      <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
        Video nicht gefunden oder kein Zugriff.
      </div>
    )
  }

  return (
    <div className="max-w-3xl">
      <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 mb-4">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold break-words">{video.title}</h1>
          <div className="mt-1 flex items-center gap-2 flex-wrap text-sm text-brand-text-muted">
            <VideoStatusPill status={video.status} />
            <span>{video.team_name}</span>
            <span>·</span>
            <span>{fmtVideoDate(video.created_at)}</span>
            <span>·</span>
            <span>{fmtDuration(video.duration_sec)}</span>
          </div>
        </div>
        {canManage && (
          <div className="flex gap-2 shrink-0">
            <button
              onClick={openEdit}
              className="inline-flex items-center gap-1.5 border border-brand-border text-brand-text rounded-md px-3 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-surface-card transition-colors"
            >
              <Pencil className="w-4 h-4" />
              Bearbeiten
            </button>
            <button
              onClick={() => { setActionError(''); setShowDelete(true) }}
              className="inline-flex items-center gap-1.5 bg-brand-danger text-white rounded-md px-3 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              Löschen
            </button>
          </div>
        )}
      </div>

      {/* Player / Status-Hinweis */}
      <div className="mb-6">
        {video.status === 'ready' ? (
          <VideoPlayer id={video.id} />
        ) : video.status === 'failed' ? (
          <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-start gap-2">
            <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
            <div>
              Die Verarbeitung ist fehlgeschlagen.
              {video.failure_reason ? <div className="mt-1 text-brand-text-muted">{video.failure_reason}</div> : null}
            </div>
          </div>
        ) : (
          <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text flex items-start gap-2">
            <Clock className="w-4 h-4 mt-0.5 shrink-0 text-brand-info" />
            Das Video wird verarbeitet… Sobald es bereit ist, kannst du es hier abspielen.
          </div>
        )}
      </div>

      {/* Metadaten */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 space-y-3">
        <div>
          <div className="text-xs uppercase text-brand-text-muted mb-1">Beschreibung</div>
          <div className="text-sm text-brand-text whitespace-pre-wrap">
            {video.description?.trim() ? video.description : <span className="text-brand-text-subtle">Keine Beschreibung</span>}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-xs uppercase text-brand-text-muted mb-1">Team</div>
            <div className="text-sm text-brand-text">{video.team_name}</div>
          </div>
          <div>
            <div className="text-xs uppercase text-brand-text-muted mb-1">Datum</div>
            <div className="text-sm text-brand-text">{fmtVideoDate(video.created_at)}</div>
          </div>
          <div>
            <div className="text-xs uppercase text-brand-text-muted mb-1">Dauer</div>
            <div className="text-sm text-brand-text">{fmtDuration(video.duration_sec)}</div>
          </div>
          {video.game_id != null && (
            <div>
              <div className="text-xs uppercase text-brand-text-muted mb-1">Verknüpftes Spiel</div>
              <button
                onClick={() => navigate(`/termine/spiel/${video.game_id}`)}
                className="text-sm text-brand-blue hover:underline"
              >
                Spiel öffnen
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Bearbeiten-Modal */}
      {showEdit && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-md">
            <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
              <h2 className="font-semibold text-base text-brand-text">Video bearbeiten</h2>
              <button onClick={() => setShowEdit(false)} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleSave} className="px-6 py-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Titel</label>
                <input
                  autoFocus
                  type="text"
                  value={editTitle}
                  onChange={e => setEditTitle(e.target.value)}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Beschreibung</label>
                <textarea
                  value={editDescription}
                  onChange={e => setEditDescription(e.target.value)}
                  rows={4}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <div>
                <label htmlFor="edit-game" className="block text-sm font-medium text-brand-text-muted mb-1">Spiel</label>
                <select
                  id="edit-game"
                  value={editGameId}
                  onChange={e => setEditGameId(e.target.value)}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                >
                  <option value="">Kein Spiel zuordnen</option>
                  {games.map(g => (
                    <option key={g.id} value={g.id}>{fmtGameOption(g)}</option>
                  ))}
                </select>
              </div>
              {actionError && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{actionError}</div>
              )}
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setShowEdit(false)} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                  Abbrechen
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {saving ? 'Speichern…' : 'Speichern'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Löschen-Modal */}
      {showDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm">
            <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
              <h2 className="font-semibold text-base text-brand-text">Video löschen</h2>
              <button onClick={() => setShowDelete(false)} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="px-6 py-5 space-y-4">
              <p className="text-sm text-brand-text">
                Video „{video.title}" wirklich löschen? Alle Dateien werden unwiderruflich entfernt.
              </p>
              {actionError && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{actionError}</div>
              )}
              <div className="flex justify-end gap-2">
                <button onClick={() => setShowDelete(false)} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                  Abbrechen
                </button>
                <button
                  onClick={handleDelete}
                  disabled={deleting}
                  className="bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {deleting ? 'Löschen…' : 'Löschen'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
