import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import MarkdownRenderer from '../components/MarkdownRenderer'
import { AlertTriangle, Trash2, Upload, X, Eye, EyeOff, Send } from 'lucide-react'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useAuth } from '../contexts/AuthContext'

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
    const [tournament, setTournament] = useState(false)
    const [abstract, setAbstract] = useState('')
    const [bodyMd, setBodyMd] = useState('')

    const reportID = Number(id)

    const load = () => {
        if (!reportID) return
        api
            .get(`/match-reports/${reportID}`)
            .then(res => {
                const r = res.data as MatchReport
                setReport(r)
                setHomeGoals(r.home_goals?.toString() ?? '')
                setAwayGoals(r.away_goals?.toString() ?? '')
                setHomeGoalsHT(r.home_goals_ht?.toString() ?? '')
                setAwayGoalsHT(r.away_goals_ht?.toString() ?? '')
                setTournament(r.tournament)
                setAbstract(r.abstract)
                setBodyMd(r.body_md)
            })
            .catch(err => setError(err.response?.data?.error ?? 'Bericht nicht gefunden'))
    }
    useEffect(load, [reportID])

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

    const saveDraft = async () => {
        setSaving(true)
        try {
            await api.put(`/match-reports/${reportID}`, {
                home_goals: numOrNull(homeGoals),
                away_goals: numOrNull(awayGoals),
                home_goals_ht: numOrNull(homeGoalsHT),
                away_goals_ht: numOrNull(awayGoalsHT),
                tournament,
                abstract,
                body_md: bodyMd,
            })
        } catch (err) {
            setError((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Speichern fehlgeschlagen')
        } finally {
            setSaving(false)
        }
    }

    const submitForReview = async () => {
        if (!confirm('Bericht zur Prüfung senden? Nach dem Absenden kannst du ihn nicht mehr bearbeiten — nur Medien oder Vorstand veröffentlichen oder korrigieren dann noch.')) return
        setSubmitting(true)
        try {
            await saveDraft()
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
            await saveDraft()
            const res = await api.post(`/match-reports/${reportID}/publish`)
            const { url } = res.data as { pageUid: number; url: string }
            alert(`Veröffentlicht: ${url}`)
            load()
        } catch (err) {
            const detail = (err as { response?: { data?: { detail?: string; error?: string } } })?.response?.data
            setError(detail?.detail || detail?.error || 'Publish fehlgeschlagen')
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
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                    Letzter Publish-Fehler: {report.error_message}
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
    const [uploading, setUploading] = useState(false)

    const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (!file) return
        e.target.value = ''
        setUploading(true)
        try {
            const form = new FormData()
            form.append('file', file)
            form.append('caption', '')
            await api.post(`/match-reports/${props.reportID}/images`, form)
            props.onChange()
        } finally {
            setUploading(false)
        }
    }

    const handleDelete = async (imgId: number) => {
        if (!confirm('Bild löschen?')) return
        await api.delete(`/match-reports/${props.reportID}/images/${imgId}`)
        props.onChange()
    }

    return (
        <div className="space-y-3">
            <div className="flex items-center justify-between">
                <label className="block text-sm font-medium text-brand-text">
                    Bilder ({props.images.length}/10)
                </label>
                {!props.readOnly && props.images.length < 10 && (
                    <label className={btnSmall + ' cursor-pointer'}>
                        <Upload className="inline-block w-3 h-3 mr-1" />
                        {uploading ? 'Lade…' : 'Bild wählen'}
                        <input type="file" accept="image/jpeg,image/png" className="hidden" onChange={handleUpload} />
                    </label>
                )}
            </div>
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

    // Bilder-Endpoints erwarten Bearer — <img src> nutzt Cookie-Auth (SSE-Stil) nicht.
    // Für den Draft-Zweck reicht ein einfacher <a>-Link (öffnet mit Session-Auth).
    return (
        <div className="border border-brand-border-subtle rounded-md p-2 bg-brand-surface-card space-y-2">
            <a
                href={`/api${props.image.url}`}
                target="_blank"
                rel="noreferrer"
                className="block aspect-square bg-brand-border-subtle rounded overflow-hidden text-xs text-brand-text-muted flex items-center justify-center"
            >
                Bild {props.image.position} anzeigen
            </a>
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
