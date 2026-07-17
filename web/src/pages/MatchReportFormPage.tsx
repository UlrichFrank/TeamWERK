import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import { compressImage } from '../lib/imageCompress'
import MarkdownRenderer from '../components/MarkdownRenderer'
import { AlertTriangle, ImageOff, Trash2, Upload, X, Eye, EyeOff, Send } from 'lucide-react'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useAuth } from '../contexts/AuthContext'

const MAX_IMAGES = 10

// Server-Fehlercodes aus internal/matchreports/images.go → deutsche User-Meldung.
// Netzfehler (keine Response) und unbekannte Codes bekommen den Fallback-Text.
function translateUploadError(status: number | undefined, code: string | undefined): string {
  switch (code) {
    case 'too_many_images':
      return 'Limit von 10 Bildern erreicht'
    case 'unsupported_mime':
      return 'Format nicht unterstützt (nur JPG/PNG)'
    case 'image_too_large':
      return 'Datei ist zu groß nach Verkleinerung'
    case 'bad_multipart':
      return 'Datei konnte nicht gelesen werden'
    case 'in_progress':
    case 'already_published':
    case 'not_found':
      return 'Bericht ist nicht mehr editierbar'
  }
  if (status && status >= 400) return 'Upload fehlgeschlagen — bitte erneut versuchen'
  return 'Upload fehlgeschlagen — bitte erneut versuchen'
}

type ReportImage = {
    id: number
    position: number
    caption: string
    url: string
}

type ConsentMember = { first_name: string; last_name: string }

type MatchReport = {
    id: number
    game_id: number
    duty_slot_id: number | null
    author_user_id: number
    state: 'draft' | 'pending_review' | 'publishing' | 'published' | 'publish_failed'
    title: string
    home_goals: number | null
    away_goals: number | null
    home_goals_ht: number | null
    away_goals_ht: number | null
    tournament: boolean
    abstract: string
    body_md: string
    published_url: string | null
    typo3_page_uid: number | null
    error_message: string | null
    images: ReportImage[]
    photo_consent_missing: ConsentMember[] | null
}

const btnPrimary =
    'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const btnDanger =
    'bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const btnSmall =
    'bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const input =
    'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function MatchReportFormPage() {
    const { id } = useParams<{ id: string }>()
    const navigate = useNavigate()
    const { user } = useAuth()
    const [report, setReport] = useState<MatchReport | null>(null)
    const [error, setError] = useState<string | null>(null)
    const [saving, setSaving] = useState(false)
    const [publishing, setPublishing] = useState(false)
    const [submitting, setSubmitting] = useState(false)
    const [preview, setPreview] = useState(false)

    const [homeGoals, setHomeGoals] = useState('')
    const [awayGoals, setAwayGoals] = useState('')
    const [homeGoalsHT, setHomeGoalsHT] = useState('')
    const [awayGoalsHT, setAwayGoalsHT] = useState('')
    const [title, setTitle] = useState<string>('')
    const [tournament, setTournament] = useState(false)
    const [abstract, setAbstract] = useState('')
    const [bodyMd, setBodyMd] = useState('')

    const reportID = Number(id)

    // Die editierbaren Felder (Titel/Ergebnis/Abstract/Body) werden nur beim
    // ERSTEN Laden pro reportID aus dem Server geseedet. Spätere Reloads
    // (SSE `match-report-event`, Bild-Upload via ImagesSection.onChange)
    // aktualisieren nur die nicht-editierbaren Teile (Status, Bilder, Consent) —
    // sonst würden ungespeicherte Eingaben mit dem (noch leeren) Serverstand
    // überschrieben und der Bericht käme leer zur Prüfung an.
    const seededRef = useRef(false)

    const load = () => {
        if (!reportID) return
        api
            .get(`/match-reports/${reportID}`)
            .then(res => {
                const r = res.data as MatchReport
                setReport(r)
                if (!seededRef.current) {
                    setTitle(r.title ?? '')
                    setHomeGoals(r.home_goals?.toString() ?? '')
                    setAwayGoals(r.away_goals?.toString() ?? '')
                    setHomeGoalsHT(r.home_goals_ht?.toString() ?? '')
                    setAwayGoalsHT(r.away_goals_ht?.toString() ?? '')
                    setTournament(r.tournament)
                    setAbstract(r.abstract)
                    setBodyMd(r.body_md)
                    seededRef.current = true
                }
            })
            .catch(err => setError(err.response?.data?.error ?? 'Bericht nicht gefunden'))
    }
    useEffect(() => {
        seededRef.current = false
        load()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [reportID])

    useLiveUpdates(evt => {
        if (evt === 'match-report-event') load()
    })

    if (error) return <div className="p-6 text-brand-danger">{error}</div>
    if (!report) return <div className="p-6 text-brand-text-muted">Lade Bericht…</div>

    // Rollen des aktuellen Users im Bericht-Kontext:
    // - Autor: darf im Draft editieren, danach nur noch lesen.
    // - Freigeber (medien|vorstand|admin): darf pending_review + publish_failed
    //   editieren und publishen.
    const isAuthor = user?.id === report.author_user_id
    const isReviewer =
        user?.role === 'admin' ||
        user?.clubFunctions?.includes('medien') ||
        user?.clubFunctions?.includes('vorstand') ||
        false

    // Wer darf gerade schreiben?
    let canEdit = false
    let canSubmit = false
    let canPublish = false
    if (report.state === 'draft') {
        canEdit = isAuthor || user?.role === 'admin'
        canSubmit = canEdit
    } else if (report.state === 'pending_review' || report.state === 'publish_failed') {
        canEdit = isReviewer
        canPublish = isReviewer
    }
    const readOnly = !canEdit

    // Liefert true bei Erfolg, false bei Fehler. submitForReview/publish MÜSSEN
    // den Rückgabewert prüfen und abbrechen, wenn das Speichern scheitert —
    // sonst würde ein Bericht mit nicht-persistierten (leeren) Feldern
    // eingereicht/veröffentlicht.
    const saveDraft = async (): Promise<boolean> => {
        setSaving(true)
        try {
            await api.put(`/match-reports/${reportID}`, {
                title,
                home_goals: numOrNull(homeGoals),
                away_goals: numOrNull(awayGoals),
                home_goals_ht: numOrNull(homeGoalsHT),
                away_goals_ht: numOrNull(awayGoalsHT),
                tournament,
                abstract,
                body_md: bodyMd,
            })
            return true
        } catch (err) {
            setError((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Speichern fehlgeschlagen')
            return false
        } finally {
            setSaving(false)
        }
    }

    const submitForReview = async () => {
        if (!confirm('Bericht zur Prüfung senden? Nach dem Absenden kannst du ihn nicht mehr bearbeiten — nur Medien oder Vorstand veröffentlichen oder korrigieren dann noch.')) return
        setSubmitting(true)
        try {
            if (!(await saveDraft())) return
            await api.post(`/match-reports/${reportID}/submit-for-review`)
            load()
        } catch (err) {
            const detail = (err as { response?: { data?: { detail?: string; error?: string } } })?.response?.data
            setError(detail?.detail || detail?.error || 'Einreichen fehlgeschlagen')
        } finally {
            setSubmitting(false)
        }
    }

    const publish = async () => {
        if (!confirm('Bericht veröffentlichen? Nach dem Publish sind Änderungen nur direkt in Typo3 möglich.')) return
        setPublishing(true)
        try {
            if (!(await saveDraft())) return
            const res = await api.post(`/match-reports/${reportID}/publish`)
            const { url } = res.data as { pageUid: number; url: string }
            alert(`Veröffentlicht: ${url}`)
            load()
        } catch (err) {
            const detail = (err as { response?: { data?: { detail?: string; error?: string } } })?.response?.data
            if (detail?.error === 'no_active_season') {
                setError('Keine aktive Saison — bitte im Verein/Saisonen setzen.')
            } else {
                setError(detail?.detail || detail?.error || 'Publish fehlgeschlagen')
            }
        } finally {
            setPublishing(false)
        }
    }

    const deleteDraft = async () => {
        if (!confirm('Draft komplett löschen?')) return
        try {
            await api.delete(`/match-reports/${reportID}`)
            navigate('/termine')
        } catch (err) {
            setError((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Löschen fehlgeschlagen')
        }
    }

    return (
        <div className="max-w-4xl mx-auto p-4 sm:p-8 space-y-6">
            <div className="flex items-center justify-between gap-4">
                <h1 className="text-2xl font-bold text-brand-text">Spielbericht</h1>
                <span className="text-xs uppercase tracking-wide text-brand-text-muted">
                    Status: {report.state}
                </span>
            </div>

            {report.state === 'published' && report.published_url && (
                <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                    Veröffentlicht:{' '}
                    <a href={report.published_url} target="_blank" rel="noreferrer" className="underline">
                        {report.published_url}
                    </a>
                </div>
            )}
            {report.state === 'publish_failed' && report.error_message && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger space-y-1">
                    <div className="font-medium">Letzter Publish-Fehler:</div>
                    <pre className="whitespace-pre-wrap break-words font-mono text-xs">{report.error_message}</pre>
                </div>
            )}

            {report.photo_consent_missing && report.photo_consent_missing.length > 0 && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex gap-2">
                    <AlertTriangle className="w-5 h-5 shrink-0" />
                    <div>
                        <strong>Foto-Freigabe fehlt bei:</strong>{' '}
                        {report.photo_consent_missing.map(m => `${m.first_name} ${m.last_name}`).join(', ')}
                        <div className="mt-1 text-brand-text">
                            Bitte prüfen, dass diese Personen auf Bildern nicht erkennbar sind. Verantwortung liegt beim Autor.
                        </div>
                    </div>
                </div>
            )}

            <div className="space-y-2">
                <label className="block text-sm font-medium text-brand-text">Titel</label>
                <input
                    type="text"
                    className={input}
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    disabled={readOnly}
                    maxLength={200}
                    placeholder="Kurzer, aussagekräftiger Titel"
                />
            </div>

            <ScoreFieldset
                tournament={tournament}
                homeGoals={homeGoals} awayGoals={awayGoals}
                homeGoalsHT={homeGoalsHT} awayGoalsHT={awayGoalsHT}
                onChange={{ setTournament, setHomeGoals, setAwayGoals, setHomeGoalsHT, setAwayGoalsHT }}
                readOnly={readOnly}
            />

            <div className="space-y-2">
                <label className="block text-sm font-medium text-brand-text">Abstract (Teaser, max 500 Zeichen)</label>
                <textarea
                    className={input}
                    rows={2}
                    maxLength={500}
                    value={abstract}
                    onChange={e => setAbstract(e.target.value)}
                    disabled={readOnly}
                    placeholder="Kurzer Anriss für Kachel-Ansicht auf der Homepage."
                />
                <div className="text-xs text-brand-text-muted text-right">{abstract.length}/500</div>
            </div>

            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="block text-sm font-medium text-brand-text">Bericht (Markdown)</label>
                    <button
                        type="button"
                        className="text-xs text-brand-text-muted hover:text-brand-text inline-flex items-center gap-1"
                        onClick={() => setPreview(p => !p)}
                    >
                        {preview ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        {preview ? 'Editor' : 'Vorschau'}
                    </button>
                </div>
                {preview ? (
                    <div className="border border-brand-border rounded-md p-3 min-h-[200px] bg-brand-surface-card">
                        <MarkdownRenderer markdown={bodyMd} />
                    </div>
                ) : (
                    <textarea
                        className={input}
                        rows={12}
                        value={bodyMd}
                        onChange={e => setBodyMd(e.target.value)}
                        disabled={readOnly}
                        placeholder="## Erste Halbzeit&#10;..."
                    />
                )}
            </div>

            <ImagesSection
                reportID={reportID}
                images={report.images ?? []}
                readOnly={readOnly}
                onChange={load}
            />

            {/* Hinweis-Banner nach Submit für den Autor */}
            {report.state === 'pending_review' && isAuthor && !isReviewer && (
                <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                    Zur Prüfung eingereicht — nur Medien oder Vorstand kann jetzt bearbeiten oder veröffentlichen.
                </div>
            )}

            {canEdit && (
                <div className="flex flex-wrap gap-3 pt-4 border-t border-brand-border-subtle">
                    <button className={btnPrimary} onClick={saveDraft} disabled={saving}>
                        {saving ? 'Speichere…' : 'Entwurf speichern'}
                    </button>
                    {canSubmit && (
                        <button className={btnPrimary} onClick={submitForReview} disabled={submitting}>
                            <Send className="inline-block w-4 h-4 mr-1" />
                            {submitting ? 'Sende…' : 'Zur Prüfung senden'}
                        </button>
                    )}
                    {canPublish && (
                        <button className={btnPrimary} onClick={publish} disabled={publishing}>
                            <Send className="inline-block w-4 h-4 mr-1" />
                            {publishing ? 'Veröffentliche…' : 'Veröffentlichen'}
                        </button>
                    )}
                    {isAuthor && report.state === 'draft' && (
                        <button className={btnDanger} onClick={deleteDraft}>
                            <Trash2 className="inline-block w-4 h-4 mr-1" />
                            Draft löschen
                        </button>
                    )}
                </div>
            )}
        </div>
    )
}

function ScoreFieldset(props: {
    tournament: boolean
    homeGoals: string; awayGoals: string
    homeGoalsHT: string; awayGoalsHT: string
    onChange: {
        setTournament: (v: boolean) => void
        setHomeGoals: (v: string) => void
        setAwayGoals: (v: string) => void
        setHomeGoalsHT: (v: string) => void
        setAwayGoalsHT: (v: string) => void
    }
    readOnly: boolean
}) {
    return (
        <fieldset className="space-y-3 p-4 border border-brand-border-subtle rounded-lg">
            <legend className="px-2 text-sm font-medium text-brand-text">Ergebnis</legend>
            <label className="flex items-center gap-2 text-sm text-brand-text">
                <input
                    type="checkbox"
                    checked={props.tournament}
                    onChange={e => props.onChange.setTournament(e.target.checked)}
                    disabled={props.readOnly}
                />
                Turnier (kein Endergebnis nötig)
            </label>
            {!props.tournament && (
                <div className="grid grid-cols-2 gap-3">
                    <div>
                        <label className="block text-xs text-brand-text-muted mb-1">Endstand Heim</label>
                        <input className={input} type="number" min={0} value={props.homeGoals}
                            onChange={e => props.onChange.setHomeGoals(e.target.value)} disabled={props.readOnly} />
                    </div>
                    <div>
                        <label className="block text-xs text-brand-text-muted mb-1">Endstand Gast</label>
                        <input className={input} type="number" min={0} value={props.awayGoals}
                            onChange={e => props.onChange.setAwayGoals(e.target.value)} disabled={props.readOnly} />
                    </div>
                    <div>
                        <label className="block text-xs text-brand-text-muted mb-1">Halbzeit Heim (optional)</label>
                        <input className={input} type="number" min={0} value={props.homeGoalsHT}
                            onChange={e => props.onChange.setHomeGoalsHT(e.target.value)} disabled={props.readOnly} />
                    </div>
                    <div>
                        <label className="block text-xs text-brand-text-muted mb-1">Halbzeit Gast (optional)</label>
                        <input className={input} type="number" min={0} value={props.awayGoalsHT}
                            onChange={e => props.onChange.setAwayGoalsHT(e.target.value)} disabled={props.readOnly} />
                    </div>
                </div>
            )}
        </fieldset>
    )
}

function ImagesSection(props: {
    reportID: number
    images: ReportImage[]
    readOnly: boolean
    onChange: () => void
}) {
    // Multi-Select-Upload: `progress` steuert das Button-Label (`Lade x/y…`).
    // `errors` und `trimInfo` sind sichtbare, persistente Meldungen — kein Toast,
    // damit die/der Nutzer:in bei 10 Files noch sieht, welche fehlgeschlagen sind.
    const [progress, setProgress] = useState<{ done: number; total: number }>({ done: 0, total: 0 })
    const [errors, setErrors] = useState<{ name: string; reason: string }[]>([])
    const [trimInfo, setTrimInfo] = useState<string | null>(null)

    const uploading = progress.total > 0
    const remaining = Math.max(0, MAX_IMAGES - props.images.length)

    const uploadOne = async (file: File): Promise<{ ok: true } | { ok: false; reason: string }> => {
        // Server-Whitelist ist image/jpeg+image/png — deshalb JPEG-only.
        // Bei Decoder-Fehlern (HEIC etc.) reicht compressImage die Original-
        // Datei durch; der Server lehnt dann mit unsupported_mime ab, was
        // per translateUploadError als klare User-Meldung landet.
        let blob: Blob = file
        let fileName = file.name
        try {
            const r = await compressImage(file, {
                formats: [{ mime: 'image/jpeg', ext: '.jpg' }],
            })
            blob = r.blob
            fileName = r.fileName
        } catch {
            /* Original-Datei nutzen, Server entscheidet */
        }
        try {
            const form = new FormData()
            form.append('file', blob, fileName)
            form.append('caption', '')
            await api.post(`/match-reports/${props.reportID}/images`, form)
            return { ok: true }
        } catch (err) {
            const resp = (err as { response?: { status?: number; data?: { error?: string } } })?.response
            return { ok: false, reason: translateUploadError(resp?.status, resp?.data?.error) }
        }
    }

    const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const picked = Array.from(e.target.files ?? [])
        // Sofort clearen: gleiche Datei kann so nach Fehler erneut ausgewählt werden.
        e.target.value = ''
        if (picked.length === 0) return

        // Alte Meldungen wegräumen — der aktuelle Upload liefert eine frische Sicht.
        setErrors([])
        setTrimInfo(null)

        // Vorab-Trim: Client-Sicht auf `remaining` kürzen. Der Server-Cap
        // MaxImages=10 bleibt der Backstop; im Race-Fall (zweiter Autor lädt
        // parallel) übernimmt die 400-Antwort die Fehleranzeige.
        let files = picked
        if (picked.length > remaining) {
            files = picked.slice(0, remaining)
            const dropped = picked.length - remaining
            setTrimInfo(
                remaining === 0
                    ? 'Limit von 10 Bildern bereits erreicht — keine weiteren Uploads.'
                    : `Nur die ersten ${remaining} Bild${remaining === 1 ? '' : 'er'} werden hochgeladen — Limit 10 erreicht (${dropped} übersprungen).`,
            )
            if (files.length === 0) {
                return
            }
        }

        setProgress({ done: 0, total: files.length })
        const collected: { name: string; reason: string }[] = []
        try {
            for (let i = 0; i < files.length; i++) {
                setProgress({ done: i, total: files.length })
                const file = files[i]
                const res = await uploadOne(file)
                if (!res.ok) collected.push({ name: file.name, reason: res.reason })
            }
        } finally {
            setProgress({ done: 0, total: 0 })
            if (collected.length > 0) setErrors(collected)
            // Ein einziges Reload am Ende reicht — bei 10 Uploads sonst 10× Reload.
            props.onChange()
        }
    }

    const handleDelete = async (imgId: number) => {
        if (!confirm('Bild löschen?')) return
        await api.delete(`/match-reports/${props.reportID}/images/${imgId}`)
        props.onChange()
    }

    const buttonLabel = uploading
        ? `Lade ${progress.done + 1}/${progress.total}…`
        : 'Bilder wählen'

    return (
        <div className="space-y-3">
            <div className="flex items-center justify-between">
                <label className="block text-sm font-medium text-brand-text">
                    Bilder ({props.images.length}/{MAX_IMAGES})
                </label>
                {!props.readOnly && remaining > 0 && (
                    <label
                        className={
                            btnSmall +
                            (uploading ? ' opacity-40 cursor-not-allowed' : ' cursor-pointer')
                        }
                    >
                        <Upload className="inline-block w-3 h-3 mr-1" />
                        {buttonLabel}
                        <input
                            type="file"
                            accept="image/jpeg,image/png"
                            multiple
                            className="hidden"
                            onChange={handleUpload}
                            disabled={uploading}
                        />
                    </label>
                )}
            </div>
            {trimInfo && (
                <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                    {trimInfo}
                </div>
            )}
            {errors.length > 0 && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                    <div className="font-medium mb-1">Nicht hochgeladen:</div>
                    <ul className="list-disc list-inside space-y-0.5">
                        {errors.map((e, i) => (
                            <li key={i}>
                                <span className="font-mono text-xs">{e.name}</span> — {e.reason}
                            </li>
                        ))}
                    </ul>
                </div>
            )}
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
                {props.images.map(img => (
                    <ImageTile
                        key={img.id}
                        image={img}
                        reportID={props.reportID}
                        readOnly={props.readOnly}
                        onDelete={() => handleDelete(img.id)}
                        onChange={props.onChange}
                    />
                ))}
            </div>
        </div>
    )
}

function ImageTile(props: {
    image: ReportImage
    reportID: number
    readOnly: boolean
    onDelete: () => void
    onChange: () => void
}) {
    const [caption, setCaption] = useState(props.image.caption)
    const [dirty, setDirty] = useState(false)
    const [previewUrl, setPreviewUrl] = useState<string | null>(null)
    const [previewError, setPreviewError] = useState(false)

    // Bilder-Endpoints erwarten Bearer-Auth — <img src> würde ohne Header laufen.
    // Deshalb via api.get(..., responseType:'blob') laden und Object-URL anzeigen.
    useEffect(() => {
        let revoked = false
        let objectUrl: string | null = null
        setPreviewError(false)
        setPreviewUrl(null)
        // Server liefert URL jetzt OHNE `/api`-Prefix (baseURL setzt der axios-Client).
        // Sanity-Check: falls doch mit `/api` reingekommen, warnen — nicht crashen.
        const url = props.image.url
        if (url.startsWith('/api/')) {
            // eslint-disable-next-line no-console
            console.warn('ReportImage.url sollte kein /api-Prefix haben:', url)
        }
        api.get(url, { responseType: 'blob' })
            .then(res => {
                if (revoked) return
                objectUrl = URL.createObjectURL(res.data as Blob)
                setPreviewUrl(objectUrl)
            })
            .catch(() => {
                if (revoked) return
                setPreviewError(true)
            })
        return () => {
            revoked = true
            if (objectUrl) URL.revokeObjectURL(objectUrl)
        }
    }, [props.image.id, props.image.url])

    return (
        <div className="border border-brand-border-subtle rounded-md p-2 bg-brand-surface-card space-y-2">
            {previewError ? (
                <div className="aspect-square w-full bg-brand-surface-card rounded overflow-hidden flex flex-col items-center justify-center text-brand-danger gap-1">
                    <ImageOff className="w-6 h-6" />
                    <span className="text-xs">Bild konnte nicht geladen werden</span>
                </div>
            ) : previewUrl ? (
                <img
                    src={previewUrl}
                    alt={`Bild ${props.image.position}`}
                    className="aspect-square object-cover w-full rounded"
                />
            ) : (
                <div className="aspect-square w-full bg-brand-border-subtle rounded overflow-hidden flex items-center justify-center text-xs text-brand-text-muted">
                    Lade…
                </div>
            )}
            <input
                type="text"
                className="w-full text-xs border border-brand-border rounded px-2 py-1"
                placeholder="Bildunterschrift"
                value={caption}
                disabled={props.readOnly}
                onChange={e => { setCaption(e.target.value); setDirty(true) }}
            />
            {!props.readOnly && (
                <div className="flex gap-2">
                    {dirty && (
                        <button className="text-xs text-brand-text-muted hover:text-brand-text"
                            onClick={() => { setDirty(false); props.onChange() /* Caption-Update via Re-Upload wär eigener Endpoint — skipped für v1 */ }}>
                            (Caption-Änderung nach v1)
                        </button>
                    )}
                    <button className="text-xs text-brand-danger hover:underline ml-auto" onClick={props.onDelete}>
                        <X className="inline-block w-3 h-3" /> löschen
                    </button>
                </div>
            )}
        </div>
    )
}

function numOrNull(s: string): number | null {
    if (s === '') return null
    const n = Number(s)
    return Number.isFinite(n) ? n : null
}
