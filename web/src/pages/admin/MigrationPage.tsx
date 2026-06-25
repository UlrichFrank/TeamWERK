import { useEffect, useState } from 'react'
import { DatabaseZap, Lock, AlertTriangle, Check } from 'lucide-react'
import { api } from '../../lib/api'
import { useVault } from '../../contexts/VaultContext'
import { useLiveUpdates } from '../../hooks/useLiveUpdates'
import { encryptBankData, encryptClubSepa, encryptFile } from '../../lib/bankCrypto'
import { b64ToBuf, bufToB64 } from '../../lib/crypto'

const BTN_PRIMARY =
  'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6'
const ALERT_ERR = 'p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger'
const ALERT_INFO = 'p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text'

interface Status {
  bridge_available: boolean
  pending_members: number
  pending_club: boolean
  pending_mandates: number
  pending_drafts: number
  complete: boolean
}

interface LegacyData {
  members: { member_id: number; iban: string; account_holder: string }[]
  club: { glaeubiger_id: string; iban: string; bic: string; kontoinhaber: string } | null
  mandates: { member_id: number; pdf_base64: string }[]
}

export default function MigrationPage() {
  const { isUnlocked } = useVault()
  const [status, setStatus] = useState<Status | null>(null)
  const [running, setRunning] = useState(false)
  const [progress, setProgress] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  const load = () => {
    api.get<Status>('/admin/migrate-legacy/status').then(r => setStatus(r.data)).catch(() => {})
  }
  useEffect(load, [])
  useLiveUpdates(event => {
    if (event === 'members' || event === 'settings') load()
  })

  // Mandate werden seitenweise migriert, damit nie mehr als MANDAT_BATCH PDFs gleichzeitig im
  // Speicher liegen (1-GB-VPS). Mitglieds-Bankdaten + Vereins-SEPA sind klein → ein Request.
  const MANDAT_BATCH = 5

  async function runMigration() {
    setError(null)
    setRunning(true)
    try {
      // 1. Kern (Mitglieds-Bankdaten + Vereins-SEPA) — klein, ein Request.
      setProgress('Lade Mitglieds-/Vereinsdaten…')
      const { data: core } = await api.get<LegacyData>('/admin/migrate-legacy/data?kind=core')
      const members = []
      for (const m of core.members) {
        const env = await encryptBankData({ iban: m.iban, account_holder: m.account_holder })
        members.push({ member_id: m.member_id, bank_ciphertext: env.bank_ciphertext, bank_dek_enc: env.bank_dek_enc })
      }
      let club = null
      if (core.club) {
        const env = await encryptClubSepa(core.club)
        club = { sepa_ciphertext: env.sepa_ciphertext, sepa_dek_enc: env.sepa_dek_enc }
      }
      if (members.length || club) {
        await api.post('/admin/migrate-legacy/upload', { members, club, mandates: [] })
      }

      // 2. SEPA-Mandat-PDFs seitenweise (self-advancing: migrierte fallen aus der Auswahl).
      const totalMandates = status?.pending_mandates ?? 0
      let migrated = 0
      for (;;) {
        const { data: page } = await api.get<LegacyData>(
          `/admin/migrate-legacy/data?kind=mandates&limit=${MANDAT_BATCH}`,
        )
        if (!page.mandates.length) break
        const mandates = []
        for (const md of page.mandates) {
          const { blob, dekEnc } = await encryptFile(new Uint8Array(b64ToBuf(md.pdf_base64)))
          mandates.push({ member_id: md.member_id, blob_base64: bufToB64(blob.buffer as ArrayBuffer), dek_enc: dekEnc })
        }
        await api.post('/admin/migrate-legacy/upload', { members: [], club: null, mandates })
        migrated += page.mandates.length
        setProgress(`SEPA-Mandate: ${migrated}/${totalMandates || migrated}`)
      }

      setDone(true)
      setProgress('')
      load()
    } catch (e) {
      const detail = (e as { response?: { data?: string } })?.response?.data
      setError(
        `Migration fehlgeschlagen${detail ? `: ${detail}` : ''}. Tresor entsperrt und Brücken-Schlüssel gesetzt? Der Lauf ist idempotent und kann wiederholt werden.`,
      )
    } finally {
      setRunning(false)
    }
  }

  const pendingTotal =
    status ? status.pending_members + (status.pending_club ? 1 : 0) + status.pending_mandates : 0

  return (
    <div className="max-w-xl space-y-6">
      <div className="flex items-center gap-2">
        <DatabaseZap className="w-6 h-6 text-brand-text" />
        <h1 className="text-xl font-semibold text-brand-text">Bankdaten-Migration</h1>
      </div>

      <p className="text-sm text-brand-text-muted">
        Einmalige Überführung des Altbestands vom alten serverseitigen Verschlüsselungsmodell auf das
        Zero-Knowledge-Envelope-Modell. Der Browser lädt den Altbestand über die Server-Brücke, verschlüsselt
        ihn neu an den Gruppenschlüssel und lädt ihn hoch. Der Lauf ist idempotent und kann gefahrlos
        wiederholt werden.
      </p>

      {status === null && <p className="text-sm text-brand-text-muted">Lädt…</p>}

      {status && status.complete && (
        <div className={CARD}>
          <div className="flex items-center gap-2 text-brand-text">
            <Check className="w-5 h-5 text-brand-success" />
            <span className="text-sm font-medium">Migration abgeschlossen</span>
          </div>
          <p className="mt-3 text-sm text-brand-text-muted">
            Kein Altbestand mehr vorhanden. Der Brücken-Schlüssel <code>FIELD_ENCRYPTION_KEY</code> kann jetzt
            vom Server entfernt werden (<code>make zk-finalize-remote</code>).
          </p>
        </div>
      )}

      {status && !status.complete && (
        <div className={CARD + ' space-y-4'}>
          <div className="flex items-start gap-2 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg">
            <AlertTriangle className="w-5 h-5 text-brand-danger shrink-0 mt-0.5" />
            <div className="text-sm text-brand-danger">
              <strong>Vor dem Start ein DB-Backup ziehen.</strong> Die Migration nullt nach erfolgreichem
              Hochladen die alten Bankspalten — dieser Schritt ist nicht umkehrbar.
            </div>
          </div>

          <ul className="text-sm text-brand-text space-y-1">
            <li>Offene Mitglieds-Bankdaten: <strong>{status.pending_members}</strong></li>
            <li>Vereins-SEPA offen: <strong>{status.pending_club ? 'ja' : 'nein'}</strong></li>
            <li>Offene SEPA-Mandat-PDFs: <strong>{status.pending_mandates}</strong></li>
            <li>
              Offene Bankdaten-Anträge (Drafts): <strong>{status.pending_drafts}</strong>
              {status.pending_drafts > 0 && (
                <span className="text-brand-text-muted"> — bitte zuerst im Mitglied annehmen/ablehnen</span>
              )}
            </li>
          </ul>

          {!status.bridge_available && (
            <div className={ALERT_ERR}>
              Die Server-Brücke ist nicht verfügbar (<code>FIELD_ENCRYPTION_KEY</code> nicht gesetzt). Ohne sie
              kann der Altbestand nicht entschlüsselt werden.
            </div>
          )}

          {!isUnlocked && (
            <div className={ALERT_INFO}>
              <span className="inline-flex items-center gap-2">
                <Lock className="w-4 h-4" /> Bitte zuerst den Bankdaten-Tresor entsperren (Menü „Tresor").
              </span>
            </div>
          )}

          {done && (
            <div className={ALERT_INFO}>
              Lauf abgeschlossen. {pendingTotal > 0 ? 'Es verbleibt Altbestand — Lauf erneut starten.' : ''}
            </div>
          )}
          {error && <div className={ALERT_ERR}>{error}</div>}
          {running && progress && <div className={ALERT_INFO}>{progress}</div>}

          <button
            onClick={runMigration}
            disabled={running || !isUnlocked || !status.bridge_available}
            className={BTN_PRIMARY}
          >
            {running ? 'Migration läuft…' : 'Migration starten'}
          </button>
        </div>
      )}
    </div>
  )
}
