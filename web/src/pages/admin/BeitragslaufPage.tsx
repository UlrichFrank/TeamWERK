import { useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Ban, CheckSquare, X } from 'lucide-react'
import { api } from '../../lib/api'
import { useLiveUpdates } from '../../hooks/useLiveUpdates'
import { formatBetrag } from '../../lib/sepa'

const BTN_PRIMARY = 'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_SECONDARY = 'border border-brand-border text-brand-text rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-table-select transition-colors disabled:opacity-40 disabled:cursor-not-allowed'

interface Season { id: number; name: string; is_active: boolean }

interface PreviewItem {
  member_id: number
  name: string
  status: string
  kategorie?: string
  kategorie_label?: string
  betrag_cent?: number
  included: boolean
  warnings: string[]
  exclusions: string[]
}

interface PreviewResp {
  saison_id: number
  saison_label: string
  faelligkeit: string
  items: PreviewItem[]
}

const EXCL_LABEL: Record<string, string> = {
  status_inaktiv: 'Status ohne Beitrag',
  beitragsfrei: 'beitragsfrei',
  kein_sepa_mandat: 'kein SEPA-Mandat',
  iban_fehlt: 'IBAN fehlt',
  iban_ungueltig: 'IBAN ungültig',
  mitgliedsnummer_fehlt: 'Mitgliedsnummer fehlt',
  adresse_unvollstaendig: 'Adresse unvollständig',
  kein_beitragssatz: 'kein Beitragssatz hinterlegt',
}

const KATEGORIE_OPTIONS: Array<{ value: string; label: string }> = [
  { value: '', label: 'alle Kategorien' },
  { value: 'aktiv_mit', label: 'Aktiv (mit Stammverein)' },
  { value: 'aktiv_ohne', label: 'Aktiv (ohne Stammverein)' },
  { value: 'passiv', label: 'Passiv' },
  { value: '__none__', label: '(keine Kategorie)' },
]

const HINWEIS_OPTIONS: Array<{ value: string; label: string }> = [
  { value: '', label: 'alle Hinweise' },
  { value: '__included__', label: 'enthalten' },
  { value: 'kein_sepa_mandat', label: 'kein SEPA-Mandat' },
  { value: 'iban_fehlt', label: 'IBAN fehlt' },
  { value: 'iban_ungueltig', label: 'IBAN ungültig' },
  { value: 'mitgliedsnummer_fehlt', label: 'Mitgliedsnummer fehlt' },
  { value: 'adresse_unvollstaendig', label: 'Adresse unvollständig' },
  { value: 'beitragsfrei', label: 'beitragsfrei' },
  { value: 'kein_beitragssatz', label: 'kein Beitragssatz hinterlegt' },
]

export default function BeitragslaufPage() {
  const [seasons, setSeasons] = useState<Season[]>([])
  const [saisonId, setSaisonId] = useState<number | null>(null)
  const [preview, setPreview] = useState<PreviewResp | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [protocolText, setProtocolText] = useState<string | null>(null)
  const [toast, setToast] = useState<string | null>(null)
  const [kategorieFilter, setKategorieFilter] = useState<string>('')
  const [hinweisFilter, setHinweisFilter] = useState<string>('')

  useEffect(() => {
    api.get('/seasons').then(r => {
      const list: Season[] = r.data ?? []
      setSeasons(list)
      const active = list.find(s => s.is_active) ?? list[0]
      if (active) setSaisonId(active.id)
    })
  }, [])

  const loadPreview = () => {
    if (!saisonId) return
    api.get(`/fee-run/preview?saison_id=${saisonId}`).then(r => {
      const data: PreviewResp = r.data
      setPreview(data)
      setSelected(new Set(data.items.filter(i => i.included).map(i => i.member_id)))
    })
  }
  useEffect(loadPreview, [saisonId])
  useLiveUpdates(event => { if (event === 'members-changed' || event === 'members') loadPreview() })

  const toggle = (id: number) => {
    const next = new Set(selected)
    if (next.has(id)) next.delete(id); else next.add(id)
    setSelected(next)
  }

  const filteredItems = useMemo(() => {
    if (!preview) return [] as PreviewItem[]
    return preview.items.filter(it => {
      if (kategorieFilter) {
        const k = it.kategorie ?? ''
        if (kategorieFilter === '__none__') {
          if (k !== '') return false
        } else if (k !== kategorieFilter) {
          return false
        }
      }
      if (hinweisFilter) {
        if (hinweisFilter === '__included__') {
          if (!it.included) return false
        } else if (!it.exclusions.includes(hinweisFilter)) {
          return false
        }
      }
      return true
    })
  }, [preview, kategorieFilter, hinweisFilter])

  const summary = useMemo(() => {
    if (!preview) return { count: 0, warn: 0, excl: 0, sepaSum: 0, exclSum: 0 }
    let count = 0, warn = 0, excl = 0, sepaSum = 0, exclSum = 0
    for (const it of filteredItems) {
      if (!it.included) {
        excl++
        exclSum += it.betrag_cent ?? 0
        continue
      }
      if (selected.has(it.member_id)) {
        count++
        sepaSum += it.betrag_cent ?? 0
        if (it.warnings.length > 0) warn++
      }
    }
    return { count, warn, excl, sepaSum, exclSum }
  }, [preview, filteredItems, selected])

  const downloadXML = async () => {
    if (!saisonId) return
    const res = await api.post('/fee-run/export',
      { saison_id: saisonId, member_ids: [...selected] },
      { responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = `beitragslauf_${preview?.saison_label.replace('/', '-')}.xml`
    a.click()
    URL.revokeObjectURL(url)
  }

  const openProtocol = async () => {
    if (!saisonId) return
    const res = await api.get(`/fee-run/protocol?saison_id=${saisonId}`, { responseType: 'text' })
    setProtocolText(res.data || '(noch kein Lauf bestätigt)')
  }

  return (
    <div className="max-w-4xl">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
        <h1 className="text-2xl font-bold text-brand-text">Beitragslauf</h1>
        {preview && (
          <div className="flex flex-wrap gap-3">
            <button onClick={downloadXML} disabled={summary.count === 0} className={BTN_PRIMARY}>XML herunterladen</button>
            <button onClick={() => setConfirmOpen(true)} className={BTN_SECONDARY}>Lauf bestätigen</button>
            <button onClick={openProtocol} className={BTN_SECONDARY}>Protokoll ansehen</button>
          </div>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-3 mb-4">
        <label className="text-sm text-brand-text-muted">Saison</label>
        <select
          value={saisonId ?? ''}
          onChange={e => setSaisonId(Number(e.target.value))}
          className="border border-brand-border rounded-md pl-3 pr-8 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow"
        >
          {seasons.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>

        <label className="text-sm text-brand-text-muted sm:ml-4">Kategorie</label>
        <select
          value={kategorieFilter}
          onChange={e => setKategorieFilter(e.target.value)}
          className="border border-brand-border rounded-md pl-3 pr-8 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow"
        >
          {KATEGORIE_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>

        <label className="text-sm text-brand-text-muted sm:ml-4">Hinweis</label>
        <select
          value={hinweisFilter}
          onChange={e => setHinweisFilter(e.target.value)}
          className="border border-brand-border rounded-md pl-3 pr-8 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow"
        >
          {HINWEIS_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>

      {preview && (
        <>
          <div className="bg-brand-surface-card rounded-xl border-t-4 border-brand-yellow shadow px-5 py-3 mb-4 text-sm text-brand-text flex flex-wrap gap-x-6 gap-y-1">
            <span className="inline-flex items-center gap-1"><CheckSquare className="w-4 h-4 shrink-0" /> {summary.count} angehakt</span>
            <span className="inline-flex items-center gap-1 text-brand-text-muted"><AlertTriangle className="w-4 h-4 shrink-0" /> {summary.warn} Warnungen</span>
            <span className="inline-flex items-center gap-1 text-brand-text-muted"><Ban className="w-4 h-4 shrink-0" /> {summary.excl} ausgeschlossen</span>
            <span className="text-brand-text-muted w-full">Fälligkeit: {preview.faelligkeit}</span>
            <span className="w-full border-t border-brand-border-subtle pt-2 flex flex-wrap gap-x-6 gap-y-1">
              <span>SEPA-Summe: <span className="font-semibold">{formatBetrag(summary.sepaSum)}</span></span>
              <span className="text-brand-text-muted">nicht abbuchbar: <span className="font-semibold">{formatBetrag(summary.exclSum)}</span></span>
              <span className="ml-auto">Gesamtsumme: <span className="font-semibold">{formatBetrag(summary.sepaSum + summary.exclSum)}</span></span>
            </span>
          </div>

          {/* Desktop-Tabelle */}
          <div className="hidden sm:block bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-4">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-brand-surface-card text-brand-text-muted text-xs uppercase">
                  <th className="px-4 py-3 text-left w-8"></th>
                  <th className="px-4 py-3 text-left">Name</th>
                  <th className="px-4 py-3 text-left">Status</th>
                  <th className="px-4 py-3 text-left">Kategorie</th>
                  <th className="px-4 py-3 text-right">Betrag</th>
                  <th className="px-4 py-3 text-left">Hinweis</th>
                </tr>
              </thead>
              <tbody>
                {filteredItems.map(it => (
                  <tr key={it.member_id} className={`border-t border-brand-border-subtle ${!it.included ? 'opacity-60' : ''}`}>
                    <td className="px-4 py-2">
                      <input
                        type="checkbox"
                        disabled={!it.included}
                        checked={selected.has(it.member_id)}
                        onChange={() => toggle(it.member_id)}
                      />
                    </td>
                    <td className="px-4 py-2 text-brand-text">{it.name}</td>
                    <td className="px-4 py-2 text-brand-text-muted">{it.status}</td>
                    <td className="px-4 py-2 text-brand-text">{it.kategorie_label ?? '—'}</td>
                    <td className="px-4 py-2 text-right text-brand-text">
                      {it.included
                        ? formatBetrag(it.betrag_cent ?? 0)
                        : (it.betrag_cent ?? 0) > 0
                          ? <span className="line-through text-brand-text-muted" title="Wird nicht abgebucht">{formatBetrag(it.betrag_cent ?? 0)}</span>
                          : '—'}
                    </td>
                    <td className="px-4 py-2">
                      {!it.included && (
                        <span className="inline-flex items-center gap-1 text-brand-danger" title={it.exclusions.map(e => EXCL_LABEL[e] ?? e).join(', ')}>
                          <Ban className="w-4 h-4 shrink-0" />
                          {it.exclusions.map(e => EXCL_LABEL[e] ?? e).join(', ')}
                        </span>
                      )}
                      {it.included && it.warnings.length > 0 && (
                        <span className="inline-flex items-center gap-1 text-brand-text-muted" title="Stammverein unklar">
                          <AlertTriangle className="w-4 h-4 shrink-0" /> Stammverein unklar
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile-Cards */}
          <div className="sm:hidden space-y-2 mb-4">
            {filteredItems.map(it => (
              <div key={it.member_id} className={`bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow px-4 py-3 ${!it.included ? 'opacity-60' : ''}`}>
                <div className="flex items-center justify-between">
                  <label className="flex items-center gap-2 font-medium text-brand-text">
                    <input type="checkbox" disabled={!it.included} checked={selected.has(it.member_id)} onChange={() => toggle(it.member_id)} />
                    {it.name}
                  </label>
                  <span className="text-brand-text">
                    {it.included
                      ? formatBetrag(it.betrag_cent ?? 0)
                      : (it.betrag_cent ?? 0) > 0
                        ? <span className="line-through text-brand-text-muted">{formatBetrag(it.betrag_cent ?? 0)}</span>
                        : '—'}
                  </span>
                </div>
                <div className="text-xs text-brand-text-muted mt-1 flex flex-wrap items-center gap-x-1">
                  <span>{it.status} · {it.kategorie_label ?? '—'}</span>
                  {!it.included && (
                    <span className="inline-flex items-center gap-1">· <Ban className="w-4 h-4 shrink-0" />{it.exclusions.map(e => EXCL_LABEL[e] ?? e).join(', ')}</span>
                  )}
                  {it.included && it.warnings.length > 0 && (
                    <span className="inline-flex items-center gap-1">· <AlertTriangle className="w-4 h-4 shrink-0" /> Stammverein unklar</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {confirmOpen && preview && (
        <ConfirmDialog
          preview={preview}
          selected={selected}
          onClose={() => setConfirmOpen(false)}
          onDone={msg => { setConfirmOpen(false); setToast(msg); setTimeout(() => setToast(null), 3000); loadPreview() }}
        />
      )}

      {protocolText !== null && (
        <Modal title="Protokoll" onClose={() => setProtocolText(null)}>
          <pre className="text-xs whitespace-pre-wrap text-brand-text max-h-[60vh] overflow-auto">{protocolText}</pre>
        </Modal>
      )}

      {toast && (
        <div className="fixed bottom-4 right-4 bg-brand-black text-brand-yellow rounded-lg px-4 py-2 text-sm shadow-lg">{toast}</div>
      )}
    </div>
  )
}

function ConfirmDialog({ preview, selected, onClose, onDone }: {
  preview: PreviewResp
  selected: Set<number>
  onClose: () => void
  onDone: (msg: string) => void
}) {
  const items = preview.items.filter(i => selected.has(i.member_id) && i.included)
  const [failed, setFailed] = useState<Set<number>>(new Set())
  const [busy, setBusy] = useState(false)

  const toggleFail = (id: number) => {
    const next = new Set(failed)
    if (next.has(id)) next.delete(id); else next.add(id)
    setFailed(next)
  }

  const submit = async () => {
    setBusy(true)
    const results = items.map(it => ({
      member_id: it.member_id,
      betrag_cent: it.betrag_cent ?? 0,
      success: !failed.has(it.member_id),
    }))
    const res = await api.post('/fee-run/confirm', { saison_id: preview.saison_id, results })
    const d = res.data
    onDone(`Protokoll fortgeschrieben: ${d.erfolgreich} erfolgreich, ${d.nicht_erfolgreich} nicht erfolgreich.`)
  }

  return (
    <Modal title="Lauf bestätigen" onClose={onClose}>
      <p className="text-sm text-brand-text-muted mb-3">
        Standardmäßig gilt jeder Einzug als erfolgreich. Hake Mitglieder ab, bei denen der Einzug <strong>nicht</strong> geklappt hat.
      </p>
      <div className="max-h-[50vh] overflow-auto border border-brand-border-subtle rounded-lg">
        <table className="w-full text-sm">
          <tbody>
            {items.map(it => (
              <tr key={it.member_id} className="border-b border-brand-border-subtle last:border-0">
                <td className="px-3 py-2 text-brand-text">{it.name}</td>
                <td className="px-3 py-2 text-right text-brand-text-muted">{formatBetrag(it.betrag_cent ?? 0)}</td>
                <td className="px-3 py-2 text-right">
                  <label className="inline-flex items-center gap-1 text-xs text-brand-danger">
                    <input type="checkbox" checked={failed.has(it.member_id)} onChange={() => toggleFail(it.member_id)} />
                    nicht eingezogen
                  </label>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex justify-end gap-2 mt-4">
        <button onClick={onClose} className={BTN_SECONDARY}>Abbrechen</button>
        <button onClick={submit} disabled={busy} className={BTN_PRIMARY}>Bestätigen ({items.length})</button>
      </div>
    </Modal>
  )
}

function Modal({ title, children, onClose }: { title: string; children: React.ReactNode; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={onClose}>
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-2xl" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">{title}</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text">
            <X className="w-5 h-5" />
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}
