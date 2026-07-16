import { useEffect, useMemo, useState } from 'react'
import axios from 'axios'
import { AlertTriangle, Ban, CheckSquare, Percent, X } from 'lucide-react'
import { api } from '../../lib/api'
import { useVault } from '../../contexts/VaultContext'
import { useLiveUpdates } from '../../hooks/useLiveUpdates'
import { formatBetrag, isValidIBAN, normalizeIBAN } from '../../lib/sepa'
import { decryptBankData, decryptClubSepa } from '../../lib/bankCrypto'
import { buildPainXML, saisonStamp, type SepaItem } from '../../lib/sepaXml'

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
  half: boolean
  half_reason?: string
  included: boolean
  warnings: string[]
  exclusions: string[]
}

const HALF_LABEL: Record<string, string> = {
  erstjahr: 'halber Beitrag (erstes Abrechnungsjahr)',
  eintritt: 'halber Beitrag (Eintritt im Jahr)',
  austritt: 'halber Beitrag (Austritt im Jahr)',
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

type Kategorie = 'aktiv_mit' | 'aktiv_ohne' | 'passiv'
const EXPORT_KATEGORIEN: Array<{ value: Kategorie; label: string }> = [
  { value: 'aktiv_mit', label: 'Aktiv (mit Stammverein)' },
  { value: 'aktiv_ohne', label: 'Aktiv (ohne Stammverein)' },
  { value: 'passiv', label: 'Passiv' },
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
  const { privateKey } = useVault()
  const [seasons, setSeasons] = useState<Season[]>([])
  const [saisonId, setSaisonId] = useState<number | null>(null)
  const [preview, setPreview] = useState<PreviewResp | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [protocolText, setProtocolText] = useState<string | null>(null)
  const [toast, setToast] = useState<string | null>(null)
  const [kategorieFilter, setKategorieFilter] = useState<string>('')
  const [hinweisFilter, setHinweisFilter] = useState<string>('')
  const [exportScope, setExportScope] = useState<Set<Kategorie>>(
    new Set<Kategorie>(['aktiv_mit', 'aktiv_ohne', 'passiv'])
  )
  const [exportDialogOpen, setExportDialogOpen] = useState(false)
  // Pro-Lauf-Override des SEPA-Einzugsdatums (ReqdColltnDt); Default = Preview-Fälligkeit
  // (01.07. der Saison). Wird bei Preview-Load initialisiert.
  const [faelligkeitOverride, setFaelligkeitOverride] = useState<string>('')

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
      setFaelligkeitOverride(data.faelligkeit)
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

  const memberIDsForScope = (scope: Set<Kategorie>): number[] => {
    if (!preview) return []
    return preview.items
      .filter(it => it.included && selected.has(it.member_id) && it.kategorie && scope.has(it.kategorie as Kategorie))
      .map(it => it.member_id)
  }

  // Zero-Knowledge-Fee-Run (Modell B): Der Server liefert nur Ciphertext + Wraps; die
  // pain.008-Datei wird hier im Browser erzeugt. Erfordert einen entsperrten Tresor.
  const downloadXML = async (scope: Set<Kategorie>) => {
    if (!saisonId) return
    if (!privateKey) {
      setToast('Bitte zuerst den Bankdaten-Tresor entsperren (Menü „Tresor").')
      return
    }
    const ids = memberIDsForScope(scope)
    if (ids.length === 0) return

    interface ExportDataItem {
      name: string; member_number: string; betrag_cent: number
      street: string; zip: string; city: string; mandat_datum: string
      bank_ciphertext: string; bank_dek_enc: string
    }
    interface ExportData {
      saison_kurz: string; faelligkeit: string; club_name: string
      club_sepa: { ciphertext: string; dek_enc: string }
      items: ExportDataItem[]
    }

    try {
      const { data } = await api.post<ExportData>('/fee-run/export-data', {
        saison_id: saisonId, member_ids: ids, faelligkeit: faelligkeitOverride,
      })

      const club = await decryptClubSepa(
        { sepa_ciphertext: data.club_sepa.ciphertext, sepa_dek_enc: data.club_sepa.dek_enc },
        privateKey,
      )

      const items: SepaItem[] = []
      const skipped: string[] = []
      for (const it of data.items) {
        const bank = await decryptBankData(
          { bank_ciphertext: it.bank_ciphertext, bank_dek_enc: it.bank_dek_enc },
          privateKey,
        )
        const iban = normalizeIBAN(bank.iban)
        if (!isValidIBAN(iban)) {
          skipped.push(`${it.name} (${it.member_number})`)
          continue
        }
        items.push({
          name: bank.account_holder || it.name,
          street: it.street, zip: it.zip, city: it.city,
          iban, betragCent: it.betrag_cent,
          mandatRef: it.member_number, mandatDatum: it.mandat_datum, memberNumber: it.member_number,
        })
      }
      if (items.length === 0) {
        setToast('Keine gültige IBAN in der Auswahl — es wurde keine Datei erzeugt.')
        return
      }

      const { xml, warnings } = buildPainXML({
        saisonKurz: data.saison_kurz,
        clubName: data.club_name,
        glaeubigerId: club.glaeubiger_id,
        clubIban: normalizeIBAN(club.iban),
        bic: normalizeIBAN(club.bic),
        kontoinhaber: club.kontoinhaber,
        faelligkeit: data.faelligkeit,
        createdAt: new Date(),
        items,
      })

      // Truncation-Warnings zeigen, bevor die Datei heruntergeladen wird — bei Debitor-
      // Nm ist die Kürzung bank-relevant (Identifikation beim Zahler), stille Mutation
      // wäre gefährlich.
      if (warnings.length > 0) {
        const lines = warnings.map(w => {
          const who = w.memberNumber ? `Mitglied ${w.memberNumber}` : 'Verein'
          return `• ${who} (${w.location}): ${w.original.length} → ${w.maxLen} Zeichen`
        }).join('\n')
        const proceed = window.confirm(
          `Für ${warnings.length} Feld(er) wird auf DK-TVS-Länge gekürzt:\n\n${lines}\n\n` +
          `Trotzdem herunterladen?`,
        )
        if (!proceed) return
      }

      const url = URL.createObjectURL(new Blob([xml], { type: 'application/xml' }))
      const a = document.createElement('a')
      a.href = url
      a.download = `beitragslauf_${saisonStamp(data.saison_kurz)}.xml`
      a.click()
      URL.revokeObjectURL(url)

      if (skipped.length > 0) {
        setToast(`${skipped.length} Mitglied(er) mit ungültiger IBAN übersprungen: ${skipped.join(', ')}`)
      }
    } catch (err) {
      // Server-Fehler (http.Error) durchreichen, damit die eigentliche Ursache sichtbar
      // ist — z. B. „faelligkeit liegt in der Vergangenheit". Nur wenn keine Server-Meldung
      // da ist, generisch fallbacken.
      let msg = 'Export fehlgeschlagen — Tresor entsperrt? Vereins-SEPA-Stammdaten gepflegt?'
      if (axios.isAxiosError(err) && typeof err.response?.data === 'string' && err.response.data.trim()) {
        msg = err.response.data.trim()
      } else if (err instanceof Error && err.message) {
        msg = err.message
      }
      setToast(msg)
    }
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
            <button onClick={() => setExportDialogOpen(true)} disabled={summary.count === 0} className={BTN_PRIMARY}>XML herunterladen</button>
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
            <span className="text-brand-text-muted w-full inline-flex items-center gap-2">
              Fälligkeit:
              <input
                type="date"
                value={faelligkeitOverride}
                onChange={e => setFaelligkeitOverride(e.target.value)}
                className="border border-brand-border rounded-md px-2 py-1 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                title="SEPA-Einzugsdatum (ReqdColltnDt) im XML; muss heute oder in der Zukunft liegen. Default: 01.07. der Saison."
              />
              {faelligkeitOverride !== preview.faelligkeit && (
                <button
                  type="button"
                  onClick={() => setFaelligkeitOverride(preview.faelligkeit)}
                  className="text-brand-text-subtle hover:text-brand-text text-xs underline"
                >
                  auf {preview.faelligkeit} zurück
                </button>
              )}
            </span>
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
                      {it.included && it.half && (
                        <span className="inline-flex items-center gap-1 text-brand-text-muted" title={HALF_LABEL[it.half_reason ?? ''] ?? 'halber Beitrag'}>
                          <Percent className="w-4 h-4 shrink-0" /> {HALF_LABEL[it.half_reason ?? ''] ?? 'halber Beitrag'}
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
                  {it.included && it.half && (
                    <span className="inline-flex items-center gap-1">· <Percent className="w-4 h-4 shrink-0" />{HALF_LABEL[it.half_reason ?? ''] ?? 'halber Beitrag'}</span>
                  )}
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

      {exportDialogOpen && preview && (
        <ExportScopeDialog
          preview={preview}
          selected={selected}
          initialScope={exportScope}
          onClose={() => setExportDialogOpen(false)}
          onConfirm={async (scope) => {
            setExportScope(scope)
            setExportDialogOpen(false)
            await downloadXML(scope)
          }}
        />
      )}

      {confirmOpen && preview && (
        <ConfirmDialog
          preview={preview}
          selected={selected}
          scope={exportScope}
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

function ConfirmDialog({ preview, selected, scope, onClose, onDone }: {
  preview: PreviewResp
  selected: Set<number>
  scope: Set<Kategorie>
  onClose: () => void
  onDone: (msg: string) => void
}) {
  const items = preview.items.filter(i =>
    selected.has(i.member_id) && i.included && i.kategorie && scope.has(i.kategorie as Kategorie))
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

function ExportScopeDialog({ preview, selected, initialScope, onClose, onConfirm }: {
  preview: PreviewResp
  selected: Set<number>
  initialScope: Set<Kategorie>
  onClose: () => void
  onConfirm: (scope: Set<Kategorie>) => void
}) {
  const [scope, setScope] = useState<Set<Kategorie>>(new Set(initialScope))

  const stats = useMemo(() => {
    const out: Record<Kategorie, { count: number; sum: number }> = {
      aktiv_mit: { count: 0, sum: 0 },
      aktiv_ohne: { count: 0, sum: 0 },
      passiv: { count: 0, sum: 0 },
    }
    for (const it of preview.items) {
      if (!it.included || !selected.has(it.member_id) || !it.kategorie) continue
      const k = it.kategorie as Kategorie
      if (!(k in out)) continue
      out[k].count++
      out[k].sum += it.betrag_cent ?? 0
    }
    return out
  }, [preview, selected])

  const toggle = (k: Kategorie) => {
    const next = new Set(scope)
    if (next.has(k)) next.delete(k); else next.add(k)
    setScope(next)
  }

  const totalCount = EXPORT_KATEGORIEN.reduce((n, k) => n + (scope.has(k.value) ? stats[k.value].count : 0), 0)
  const totalSum = EXPORT_KATEGORIEN.reduce((s, k) => s + (scope.has(k.value) ? stats[k.value].sum : 0), 0)

  return (
    <Modal title="Welche Beiträge in die XML aufnehmen?" onClose={onClose}>
      <p className="text-sm text-brand-text-muted mb-3">
        Wähle, welche Kategorien in den Beitragslauf einfließen. Die Auswahl gilt auch für „Lauf bestätigen".
      </p>
      <div className="space-y-2">
        {EXPORT_KATEGORIEN.map(k => {
          const { count, sum } = stats[k.value]
          return (
            <label key={k.value} className="flex items-center justify-between gap-3 px-3 py-2 border border-brand-border-subtle rounded-lg">
              <span className="flex items-center gap-2 text-brand-text">
                <input type="checkbox" checked={scope.has(k.value)} onChange={() => toggle(k.value)} />
                {k.label}
              </span>
              <span className="text-sm text-brand-text-muted">
                {count} Mitglied{count === 1 ? '' : 'er'} · <span className="font-semibold text-brand-text">{formatBetrag(sum)}</span>
              </span>
            </label>
          )
        })}
      </div>
      <div className="border-t border-brand-border-subtle mt-3 pt-3 flex justify-between text-sm">
        <span className="text-brand-text-muted">Auswahl gesamt</span>
        <span className="text-brand-text">{totalCount} Mitglied{totalCount === 1 ? '' : 'er'} · <span className="font-semibold">{formatBetrag(totalSum)}</span></span>
      </div>
      <div className="flex justify-end gap-2 mt-4">
        <button onClick={onClose} className={BTN_SECONDARY}>Abbrechen</button>
        <button onClick={() => onConfirm(scope)} disabled={totalCount === 0} className={BTN_PRIMARY}>XML herunterladen</button>
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
