import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { FileText, Plus, ExternalLink } from 'lucide-react'

type ReportItem = {
    id: number
    game_id: number
    state: 'draft' | 'pending_review' | 'publishing' | 'published' | 'publish_failed'
    match_date: string
    opponent: string
    published_url: string | null
}

type SlotItem = {
    slot_id: number
    game_id: number
    match_date: string
    opponent: string
}

const btnPrimary =
    'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const btnSmall =
    'bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'

const STATE_LABEL: Record<string, string> = {
    draft: 'Entwurf',
    pending_review: 'Wartet auf Freigabe',
    publishing: 'Wird veröffentlicht…',
    published: 'Veröffentlicht',
    publish_failed: 'Fehler',
}

export default function MatchReportListPage() {
    const navigate = useNavigate()
    const [reports, setReports] = useState<ReportItem[]>([])
    const [openSlots, setOpenSlots] = useState<SlotItem[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [creating, setCreating] = useState<number | null>(null)

    const load = () => {
        setLoading(true)
        api
            .get('/match-reports/my')
            .then(res => {
                setReports(res.data.reports ?? [])
                setOpenSlots(res.data.open_slots ?? [])
                setError(null)
            })
            .catch(() => setError('Konnte Berichte nicht laden.'))
            .finally(() => setLoading(false))
    }
    useEffect(load, [])

    useLiveUpdates(evt => {
        if (evt === 'match-report-event' || evt === 'duty-event') load()
    })

    const startReport = async (slot: SlotItem) => {
        setCreating(slot.slot_id)
        try {
            const res = await api.post('/match-reports', {
                game_id: slot.game_id,
                duty_slot_id: slot.slot_id,
            })
            const id = res.data?.id as number
            if (id) navigate(`/spielberichte/${id}`)
        } catch (err) {
            const detail = (err as { response?: { data?: { error?: string } } })?.response?.data
            alert('Anlegen fehlgeschlagen: ' + (detail?.error ?? 'unbekannt'))
        } finally {
            setCreating(null)
        }
    }

    return (
        <div className="max-w-4xl mx-auto p-4 sm:p-8 space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold text-brand-text">Spielberichte</h1>
            </div>

            {error && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                    {error}
                </div>
            )}

            {/* Offene Slots — Einstiegs-CTA */}
            <section className="space-y-3">
                <h2 className="text-sm font-medium text-brand-text-muted uppercase">
                    Offene Aufträge ({openSlots.length})
                </h2>
                {openSlots.length === 0 ? (
                    <p className="text-sm text-brand-text-muted">
                        Keine offenen Spielbericht-Aufträge. Übernimm einen Slot in der{' '}
                        <a href="/dienste" className="underline text-brand-text">Dienstbörse</a>.
                    </p>
                ) : (
                    <ul className="space-y-2">
                        {openSlots.map(slot => (
                            <li
                                key={slot.slot_id}
                                className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4 flex items-center justify-between gap-3"
                            >
                                <div>
                                    <div className="text-sm font-medium text-brand-text">
                                        {formatDate(slot.match_date)} — {slot.opponent}
                                    </div>
                                    <div className="text-xs text-brand-text-muted">Bericht ausstehend</div>
                                </div>
                                <button
                                    className={btnPrimary}
                                    onClick={() => startReport(slot)}
                                    disabled={creating === slot.slot_id}
                                >
                                    <Plus className="inline-block w-4 h-4 mr-1" />
                                    {creating === slot.slot_id ? 'Lege an…' : 'Bericht schreiben'}
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </section>

            {/* Bericht-Liste */}
            <section className="space-y-3">
                <h2 className="text-sm font-medium text-brand-text-muted uppercase">
                    Meine Berichte ({reports.length})
                </h2>
                {loading ? (
                    <p className="text-sm text-brand-text-muted">Lade…</p>
                ) : reports.length === 0 ? (
                    <p className="text-sm text-brand-text-muted">Noch keine Berichte angelegt.</p>
                ) : (
                    <ul className="space-y-2">
                        {reports.map(r => (
                            <li
                                key={r.id}
                                className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4 flex items-center justify-between gap-3"
                            >
                                <div className="flex items-start gap-3">
                                    <FileText className="w-5 h-5 text-brand-text-muted mt-0.5" />
                                    <div>
                                        <div className="text-sm font-medium text-brand-text">
                                            {formatDate(r.match_date)} — {r.opponent}
                                        </div>
                                        <div className="text-xs text-brand-text-muted">
                                            {STATE_LABEL[r.state] ?? r.state}
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2">
                                    {r.published_url && (
                                        <a
                                            href={r.published_url}
                                            target="_blank"
                                            rel="noreferrer"
                                            className="text-xs text-brand-text-muted hover:text-brand-text inline-flex items-center gap-1"
                                        >
                                            <ExternalLink className="w-3 h-3" />
                                            Homepage
                                        </a>
                                    )}
                                    <button
                                        className={btnSmall}
                                        onClick={() => navigate(`/spielberichte/${r.id}`)}
                                    >
                                        Öffnen
                                    </button>
                                </div>
                            </li>
                        ))}
                    </ul>
                )}
            </section>
        </div>
    )
}

function formatDate(iso: string): string {
    // "2026-05-15" → "15.05.2026"
    if (!iso || iso.length < 10) return iso
    const [y, m, d] = iso.slice(0, 10).split('-')
    return `${d}.${m}.${y}`
}
