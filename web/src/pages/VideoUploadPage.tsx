import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { AxiosError } from 'axios'
import * as tus from 'tus-js-client'
import { Upload, X, AlertTriangle, Info } from 'lucide-react'
import { api, getAccessToken } from '../lib/api'
import { buildTeamShortNames } from '../lib/teamName'

const MAX_SIZE = 2 * 1024 * 1024 * 1024 // 2 GiB

// tus-js-client exportiert die PreviousUpload-Form nicht als Typ → über den
// Rückgabewert von findPreviousUploads() ableiten.
type PreviousUpload = Awaited<ReturnType<tus.Upload['findPreviousUploads']>>[number]

interface Team {
  id: number
  name: string
  age_class: string
  gender: string
  team_number: number
  group_count: number
  is_active: boolean
}

interface Season {
  id: number
  is_active: boolean
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

interface CreateUploadResponse {
  video_id: number
  upload_url: string
}

const inputCls =
  'w-full border border-brand-border rounded-md px-3 py-2.5 sm:py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

function fmtRemaining(seconds: number): string {
  if (!isFinite(seconds) || seconds <= 0) return '–'
  const m = Math.floor(seconds / 60)
  const s = Math.round(seconds % 60)
  if (m <= 0) return `${s}s`
  return `${m}m ${s.toString().padStart(2, '0')}s`
}

function fmtGameDate(iso: string): string {
  // SQLite-Datumsfelder kommen als ISO-Timestamp; nur das Datum verwenden.
  const d = iso.slice(0, 10)
  const [y, mo, da] = d.split('-')
  if (!y || !mo || !da) return d
  return `${da}.${mo}.${y}`
}

export default function VideoUploadPage() {
  const navigate = useNavigate()

  const [teams, setTeams] = useState<Team[]>([])
  const [games, setGames] = useState<Game[]>([])
  const [seasons, setSeasons] = useState<Season[]>([])

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [teamId, setTeamId] = useState('')
  const [gameId, setGameId] = useState('')
  const [file, setFile] = useState<File | null>(null)

  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [remaining, setRemaining] = useState('')
  const [done, setDone] = useState<number | null>(null)
  const [resumable, setResumable] = useState<PreviousUpload | null>(null)

  const uploadRef = useRef<tus.Upload | null>(null)
  const startTimeRef = useRef(0)

  const activeTeams = useMemo(() => teams.filter(t => t.is_active), [teams])
  const shortNames = useMemo(() => buildTeamShortNames(activeTeams), [activeTeams])
  const activeSeasonId = useMemo(() => seasons.find(s => s.is_active)?.id ?? 0, [seasons])

  // Teams laden (nur die der User hochladen darf — Server erzwingt zusätzlich 403).
  useEffect(() => {
    api.get<Team[]>('/teams')
      .then(r => setTeams(Array.isArray(r.data) ? r.data : []))
      .catch(() => setTeams([]))
  }, [])

  // Saisons laden — das Video wird der aktiven Saison zugeordnet (Server verlangt season_id).
  useEffect(() => {
    api.get<Season[]>('/seasons')
      .then(r => setSeasons(Array.isArray(r.data) ? r.data : []))
      .catch(() => setSeasons([]))
  }, [])

  // Spiele für das gewählte Team laden (clientseitig nach teams[].id filtern).
  useEffect(() => {
    setGameId('')
    if (!teamId) {
      setGames([])
      return
    }
    api.get<Game[]>('/games')
      .then(r => {
        const all = Array.isArray(r.data) ? r.data : []
        const tid = Number(teamId)
        setGames(all.filter(g => (g.teams ?? []).some(t => t.id === tid)))
      })
      .catch(() => setGames([]))
  }, [teamId])

  // tus-Resume: liegt für die aktuelle Datei eine frühere Upload-Session im
  // localStorage vor, bieten wir "Upload fortsetzen?" an.
  useEffect(() => {
    setResumable(null)
    if (!file) return
    const probe = new tus.Upload(file, {
      endpoint: '/api/videos/upload/',
      storeFingerprintForResuming: true,
    })
    probe.findPreviousUploads()
      .then(prev => {
        if (prev.length > 0) setResumable(prev[0])
      })
      .catch(() => {})
  }, [file])

  const onFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setError('')
    const f = e.target.files?.[0] ?? null
    if (f && f.size > MAX_SIZE) {
      setFile(null)
      setError('Datei zu groß: maximal 2 GB erlaubt.')
      return
    }
    setFile(f)
  }, [])

  function startTus(videoId: number, f: File, resume: PreviousUpload | null) {
    setUploading(true)
    setProgress(0)
    setRemaining('')
    startTimeRef.current = Date.now()

    const upload = new tus.Upload(f, {
      endpoint: '/api/videos/upload/',
      metadata: {
        video_id: String(videoId),
        filename: f.name,
        filetype: f.type,
      },
      // tus läuft NICHT über die axios-Instanz → Bearer-Token explizit setzen.
      // Aktuellen Access-Token zum Upload-Start lesen (kann zwischendurch rotieren).
      headers: { Authorization: `Bearer ${getAccessToken() ?? ''}` },
      storeFingerprintForResuming: true,
      removeFingerprintOnSuccess: true,
      onError: (err) => {
        setUploading(false)
        setError(`Upload fehlgeschlagen: ${err.message}`)
      },
      onProgress: (bytesSent, bytesTotal) => {
        const pct = bytesTotal > 0 ? Math.round((bytesSent / bytesTotal) * 100) : 0
        setProgress(pct)
        const elapsed = (Date.now() - startTimeRef.current) / 1000
        if (elapsed > 0 && bytesSent > 0) {
          const rate = bytesSent / elapsed // bytes/s
          const left = (bytesTotal - bytesSent) / rate
          setRemaining(fmtRemaining(left))
        }
      },
      onSuccess: () => {
        setUploading(false)
        setProgress(100)
        setDone(videoId)
      },
    })
    uploadRef.current = upload
    if (resume) upload.resumeFromPreviousUpload(resume)
    upload.start()
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!title.trim()) {
      setError('Titel ist erforderlich.')
      return
    }
    if (!teamId) {
      setError('Bitte ein Team auswählen.')
      return
    }
    if (!file) {
      setError('Bitte eine Videodatei auswählen.')
      return
    }
    if (file.size > MAX_SIZE) {
      setError('Datei zu groß: maximal 2 GB erlaubt.')
      return
    }
    if (!activeSeasonId) {
      setError('Keine aktive Saison gesetzt — bitte unter Einstellungen → Saisons eine Saison aktivieren.')
      return
    }

    setUploading(true)
    try {
      const res = await api.post<CreateUploadResponse>('/videos', {
        title: title.trim(),
        team_id: Number(teamId),
        season_id: activeSeasonId,
        description: description.trim() || undefined,
        game_id: gameId ? Number(gameId) : undefined,
        size_bytes: file.size,
      })
      startTus(res.data.video_id, file, resumable)
    } catch (err) {
      setUploading(false)
      const ax = err as AxiosError<{ error?: string }>
      const status = ax.response?.status
      if (status === 507) {
        setError('Server-Speicher voll, bitte später erneut versuchen oder Admin informieren.')
      } else if (status === 403) {
        setError('Keine Berechtigung für dieses Team.')
      } else if (status === 400) {
        setError(ax.response?.data?.error || 'Ungültige Eingabe.')
      } else {
        setError('Upload konnte nicht gestartet werden.')
      }
    }
  }

  function handleResume() {
    if (!file || !resumable) return
    setError('')
    // Ohne neue video_id keine Pre-Upload-Zeile — der gespeicherte tus-Upload
    // trägt seine video_id-Metadaten bereits in sich. Direkt fortsetzen.
    const upload = new tus.Upload(file, {
      endpoint: '/api/videos/upload/',
      headers: { Authorization: `Bearer ${getAccessToken() ?? ''}` },
      storeFingerprintForResuming: true,
      removeFingerprintOnSuccess: true,
      onError: (err) => {
        setUploading(false)
        setError(`Upload fehlgeschlagen: ${err.message}`)
      },
      onProgress: (bytesSent, bytesTotal) => {
        const pct = bytesTotal > 0 ? Math.round((bytesSent / bytesTotal) * 100) : 0
        setProgress(pct)
        const elapsed = (Date.now() - startTimeRef.current) / 1000
        if (elapsed > 0 && bytesSent > 0) {
          const rate = bytesSent / elapsed
          setRemaining(fmtRemaining((bytesTotal - bytesSent) / rate))
        }
      },
      onSuccess: () => {
        setUploading(false)
        setProgress(100)
        const vid = Number(resumable.metadata?.video_id)
        setDone(Number.isFinite(vid) ? vid : null)
      },
    })
    uploadRef.current = upload
    startTimeRef.current = Date.now()
    setUploading(true)
    setProgress(0)
    upload.resumeFromPreviousUpload(resumable)
    upload.start()
  }

  if (done !== null) {
    return (
      <div className="max-w-xl">
        <h1 className="text-2xl font-bold mb-4">Video hochladen</h1>
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <p className="text-sm text-brand-text mb-4">
            Upload abgeschlossen. Das Video wird jetzt verarbeitet.
          </p>
          <button
            onClick={() => navigate(`/videos/${done}`)}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            Zum Video
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-xl">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">Video hochladen</h1>
        <button
          onClick={() => navigate('/videos')}
          aria-label="Abbrechen"
          className="inline-flex items-center justify-center w-11 h-11 sm:w-9 sm:h-9 rounded-md text-brand-text-muted hover:bg-brand-table-select transition-colors"
        >
          <X className="w-5 h-5" />
        </button>
      </div>

      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4 flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

        <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text mb-4 flex items-start gap-2">
           <Info className="w-4 h-4 mt-0.5 shrink-0 text-brand-info" />
           <div className="space-y-1">
             <p className="font-medium">So läuft die Bereitstellung</p>
             <ol className="list-decimal list-inside text-brand-text-muted space-y-0.5">
               <li>Upload (max. 2 GB, pausierbar/wiederaufnehmbar).</li>
              <li>Konvertierung im Hintergrund nach HLS 720p + 360p — dauert grob so lange wie das Video selbst.</li>
              <li>Sobald fertig: Push-Benachrichtigung an Uploader, Spieler, Eltern und Trainer; Video erscheint in der Liste.</li>
            </ol>
            <p className="text-brand-text-muted">
              Sichtbar nur für Team-Mitglieder, Eltern, Trainer, Vorstand. Automatische Löschung 90 Tage nach Saisonende.
            </p>
          </div>
        </div>
        <form onSubmit={handleSubmit} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 space-y-4">
        <div>
          <label htmlFor="title" className="block text-sm font-medium text-brand-text mb-1">Titel *</label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={e => setTitle(e.target.value)}
            disabled={uploading}
            className={inputCls}
            placeholder="z.B. Heimspiel vs. TV Musterstadt"
          />
        </div>

        <div>
          <label htmlFor="description" className="block text-sm font-medium text-brand-text mb-1">Beschreibung</label>
          <textarea
            id="description"
            value={description}
            onChange={e => setDescription(e.target.value)}
            disabled={uploading}
            rows={3}
            className={inputCls}
            placeholder="Optional"
          />
        </div>

        <div>
          <label htmlFor="team" className="block text-sm font-medium text-brand-text mb-1">Team *</label>
          <select
            id="team"
            value={teamId}
            onChange={e => setTeamId(e.target.value)}
            disabled={uploading}
            className={inputCls}
          >
            <option value="">Team auswählen…</option>
            {activeTeams.map(t => (
              <option key={t.id} value={t.id}>{shortNames.get(t.id) ?? t.name}</option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="game" className="block text-sm font-medium text-brand-text mb-1">Spiel (optional)</label>
          <select
            id="game"
            value={gameId}
            onChange={e => setGameId(e.target.value)}
            disabled={uploading || !teamId}
            className={inputCls}
          >
            <option value="">Kein Spiel zuordnen</option>
            {games.map(g => (
              <option key={g.id} value={g.id}>
                {fmtGameDate(g.date)} · {g.opponent || 'Spiel'}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="file" className="block text-sm font-medium text-brand-text mb-1">Videodatei *</label>
          <input
            id="file"
            type="file"
            accept="video/*"
            onChange={onFileChange}
            disabled={uploading}
            className="block w-full text-sm text-brand-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-yellow file:text-brand-black file:px-4 file:py-2 file:text-sm file:font-medium hover:file:bg-brand-black hover:file:text-brand-yellow file:cursor-pointer"
          />
          <p className="text-xs text-brand-text-muted mt-1">Maximal 2 GB.</p>
        </div>

        {resumable && !uploading && (
          <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            <p className="mb-2">Für diese Datei liegt ein unterbrochener Upload vor.</p>
            <button
              type="button"
              onClick={handleResume}
              className="bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
            >
              Upload fortsetzen
            </button>
          </div>
        )}

        {uploading && (
          <div>
            <div className="flex items-center justify-between text-xs text-brand-text-muted mb-1">
              <span>{progress}%</span>
              {remaining && <span>noch ca. {remaining}</span>}
            </div>
            <div className="w-full h-2 bg-brand-border-subtle rounded-full overflow-hidden">
              <div
                className="h-full bg-brand-yellow transition-all"
                style={{ width: `${progress}%` }}
                role="progressbar"
                aria-valuenow={progress}
                aria-valuemin={0}
                aria-valuemax={100}
              />
            </div>
          </div>
        )}

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={uploading}
            className="inline-flex items-center justify-center gap-2 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <Upload className="w-4 h-4" />
            {uploading ? 'Lädt hoch…' : 'Hochladen'}
          </button>
        </div>
      </form>
    </div>
  )
}
