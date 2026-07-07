import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { FileText, AlertTriangle, Image as ImageIcon } from 'lucide-react'

type PendingItem = {
    id: number
    game_id: number
    opponent: string
    match_date: string
    submitted_at: string
    author_name: string
    image_count: number
}

const btnSmall =
    'bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors'

// Deadline-Schwelle: nach 5 Tagen wird der Bericht rot markiert
// (identisch zum Backend-Reminder-Job).
const OVERDUE_DAYS = 5

function daysSince(iso: string): number {
    if (!iso) return 0
    const then = new Date(iso).getTime()
    if (!Number.isFinite(then)) return 0
    return Math.floor((Date.now() - then) / (24 * 3600 * 1000))
}

export default function MatchReportPendingListPage() {
    const navigate = useNavigate()
    const [items, setItems] = useState<PendingItem[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    const load = () => {
        setLoading(true)
        api
            .get('/match-reports/pending')
            .then(res => {
                setItems(res.data ?? [])
                setError(null)
            })
            .catch(() => setError('Konnte offene Berichte nicht laden.'))
            .finally(() => setLoading(false))
    }
    useEffect(load, [])

    useLiveUpdates(evt => {
        if (evt === 'match-report-event') load()
    })

    return (
        <div className="max-w-4xl mx-auto p-4 sm:p-8 space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold text-brand-text">Berichte zur Prüfung</h1>
            </div>

            {error && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                    {error}
                </div>
            )}

            {loading ? (
                <div className="text-brand-text-muted">Lade…</div>
            ) : items.length === 0 ? (
                <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 text-brand-text-muted">
                    Aktuell wartet kein Bericht auf Freigabe.
                </div>
            ) : (
                <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
                    <ul className="divide-y divide-brand-border-subtle">
                        {items.map(item => {
                            const age = daysSince(item.submitted_at)
                            const overdue = age >= OVERDUE_DAYS
                            return (
                                <li
                                    key={item.id}
                                    className="p-4 hover:bg-brand-table-select transition-colors cursor-pointer"
                                    onClick={() => navigate(`/spielberichte/${item.id}`)}
                                >
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex items-start gap-3">
                                            <FileText className="w-5 h-5 mt-0.5 text-brand-text-muted shrink-0" />
                                            <div>
                                                <div className="font-medium text-brand-text">
                                                    {item.opponent}
                                                </div>
                                                <div className="text-xs text-brand-text-muted mt-0.5">
                                                    Autor: {item.author_name} · Spiel am{' '}
                                                    {item.match_date?.slice(0, 10)}
                                                </div>
                                                <div className="text-xs text-brand-text-muted mt-0.5 flex items-center gap-3">
                                                    <span>Eingereicht: {age === 0 ? 'heute' : `vor ${age} Tag${age === 1 ? '' : 'en'}`}</span>
                                                    {item.image_count > 0 && (
                                                        <span className="inline-flex items-center gap-1">
                                                            <ImageIcon className="w-3 h-3" />
                                                            {item.image_count}
                                                        </span>
                                                    )}
                                                </div>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2 shrink-0">
                                            {overdue && (
                                                <span
                                                    className="inline-flex items-center gap-1 bg-brand-danger-light text-brand-danger text-xs font-medium rounded-full px-2 py-0.5"
                                                    title={`Seit ${age} Tagen offen`}
                                                >
                                                    <AlertTriangle className="w-3 h-3" />
                                                    Überfällig
                                                </span>
                                            )}
                                            <button
                                                type="button"
                                                className={btnSmall}
                                                onClick={e => {
                                                    e.stopPropagation()
                                                    navigate(`/spielberichte/${item.id}`)
                                                }}
                                            >
                                                Prüfen
                                            </button>
                                        </div>
                                    </div>
                                </li>
                            )
                        })}
                    </ul>
                </div>
            )}
        </div>
    )
}
