import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { ExternalLink, Trash2, AlertTriangle } from 'lucide-react'
import { api } from '../../lib/api'
import { errorMessage } from '../../lib/errors'
import { useAuth } from '../../contexts/AuthContext'
import { useVault } from '../../contexts/VaultContext'
import { encryptFile, decryptFile, decryptBankData, BankEnvelope } from '../../lib/bankCrypto'

const formatIBAN = (raw: string) =>
  raw.replace(/\s/g, '').toUpperCase().match(/.{1,4}/g)?.join(' ') ?? ''

const validateIBAN = (iban: string): boolean => {
  const clean = iban.replace(/\s/g, '').toUpperCase()
  if (!/^DE[0-9]{20}$/.test(clean)) return false
  const rearranged = clean.slice(4) + clean.slice(0, 4)
  const numeric = rearranged.split('').map(c =>
    c >= 'A' ? (c.charCodeAt(0) - 55).toString() : c
  ).join('')
  let remainder = 0
  for (const ch of numeric) remainder = (remainder * 10 + parseInt(ch)) % 97
  return remainder === 1
}

interface Member {
  iban?: string
  account_holder?: string
  beitragsfrei?: boolean
  beitragsfrei_grund?: string
  sepa_mandat?: boolean
  sepa_mandat_date?: string
  sepa_mandat_url?: string
}

interface Draft {
  id: number
  field_name: string
  old_value: { account_holder?: string; iban?: string; verarbeitung?: boolean; weitergabe?: boolean; [k: string]: unknown } | null
  new_value: { account_holder?: string; iban?: string; verarbeitung?: boolean; weitergabe?: boolean; [k: string]: unknown } | null
}

interface Props {
  memberId?: number
  form: Member
  isNew: boolean
  drafts: Draft[]
  onFormChange: (updates: Partial<Member>) => void
  onDraftAccept: (draftId: number) => Promise<void>
  onDraftReject: (draftId: number) => Promise<void>
  onSave: () => Promise<void>
  saving: boolean
  saved: boolean
  error: string
}

export default function MemberKontaktTab({ memberId, form, isNew, drafts, onFormChange, onDraftAccept, onDraftReject, onSave, saving, saved, error }: Props) {
  const { user, hasCapability } = useAuth()
  const { privateKey } = useVault()
  const bankdatenDraft = drafts.find(d => d.field_name === 'bankdaten') ?? null
  const sepaDraft = drafts.find(d => d.field_name === 'sepa_mandat') ?? null

  const [decryptedDraft, setDecryptedDraft] = useState<{ iban: string; account_holder: string } | null>(null)

  useEffect(() => {
    if (!bankdatenDraft || !privateKey) { setDecryptedDraft(null); return }
    const env = bankdatenDraft.new_value as unknown as BankEnvelope
    if (!env?.bank_ciphertext || !env?.bank_dek_enc) { setDecryptedDraft(null); return }
    let cancelled = false
    decryptBankData(env, privateKey)
      .then(d => { if (!cancelled) setDecryptedDraft(d) })
      .catch(() => { if (!cancelled) setDecryptedDraft(null) })
    return () => { cancelled = true }
  }, [privateKey, bankdatenDraft])

  const [ibanDisplay, setIbanDisplay] = useState(formatIBAN(form.iban || ''))
  const [ibanError, setIbanError] = useState('')

  const sepaInputRef = useRef<HTMLInputElement>(null)
  const [sepaUploading, setSepaUploading] = useState(false)
  const [sepaUploadError, setSepaUploadError] = useState('')
  const [openError, setOpenError] = useState('')
  const [deleteError, setDeleteError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)

  const canDeleteSepa = hasCapability('manage_members') || user?.isParent === true
  const MAX_SEPA_BYTES = 2 * 1024 * 1024

  useEffect(() => {
    setIbanDisplay(formatIBAN(form.iban || ''))
  }, [form.iban])

  const handleIbanChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
    if (raw.length > 22) return
    setIbanDisplay(raw.match(/.{1,4}/g)?.join(' ') ?? raw)
    setIbanError('')
    onFormChange({ iban: raw })
  }

  const handleIbanBlur = () => {
    const raw = ibanDisplay.replace(/\s/g, '')
    if (raw && !validateIBAN(raw)) setIbanError('Ungültige IBAN')
    else setIbanError('')
  }

  const handleSepaUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || isNew || !memberId) return
    if (file.size > MAX_SEPA_BYTES) {
      setSepaUploadError('Die Datei ist zu groß. Maximal erlaubt sind 2 MB.')
      if (sepaInputRef.current) sepaInputRef.current.value = ''
      return
    }
    setSepaUploading(true)
    setSepaUploadError('')
    try {
      // Zero-Knowledge: PDF clientseitig an den Gruppen-Schlüssel verschlüsseln und als
      // Blob + gewrappten DEK hochladen. Der Server sieht das Dokument nie im Klartext.
      const bytes = new Uint8Array(await file.arrayBuffer())
      const { blob, dekEnc } = await encryptFile(bytes)
      const formData = new FormData()
      formData.append('file', new Blob([blob as BlobPart], { type: 'application/octet-stream' }), 'mandat.bin')
      formData.append('dek_enc', dekEnc)
      const { data } = await api.post<{ sepa_mandat_url: string }>(
        `/upload/sepa-mandat/${memberId}`,
        formData,
        { headers: { 'Content-Type': 'multipart/form-data' } }
      )
      onFormChange({ sepa_mandat_url: data.sepa_mandat_url })
    } catch (e) {
      const msg = errorMessage(e, '')
      setSepaUploadError(
        msg.includes('too_large')
          ? 'Die Datei ist zu groß. Maximal erlaubt sind 2 MB.'
          : msg.includes('eingerichtet')
            ? 'Bankdaten-Tresor ist noch nicht eingerichtet.'
            : 'Hochladen fehlgeschlagen.'
      )
    } finally {
      setSepaUploading(false)
      if (sepaInputRef.current) sepaInputRef.current.value = ''
    }
  }

  // Mandat clientseitig entschlüsseln und anzeigen — braucht den entsperrten Tresor.
  const openSepaMandat = async () => {
    if (!memberId) return
    setOpenError('')
    if (!privateKey) {
      setOpenError('Zum Öffnen den Bankdaten-Tresor entsperren (Menü „Tresor").')
      return
    }
    try {
      const { data } = await api.get<{ token: string; dek_enc: string }>(
        `/members/${memberId}/sepa-mandat/download-token`,
      )
      const res = await api.get<ArrayBuffer>(
        `/members/${memberId}/sepa-mandat/download?token=${data.token}`,
        { responseType: 'arraybuffer' },
      )
      const plain = await decryptFile(new Uint8Array(res.data), data.dek_enc, privateKey)
      const url = URL.createObjectURL(new Blob([plain as BlobPart], { type: 'application/pdf' }))
      window.open(url, '_blank')
      setTimeout(() => URL.revokeObjectURL(url), 60_000)
    } catch {
      setOpenError('Dokument konnte nicht geöffnet/entschlüsselt werden.')
    }
  }

  const handleDeleteSepa = async () => {
    if (!memberId) return
    setDeleteError('')
    try {
      await api.delete(`/members/${memberId}/sepa-mandat`)
      onFormChange({ sepa_mandat_url: undefined, sepa_mandat: false, sepa_mandat_date: '' })
    } catch {
      setDeleteError('Löschen fehlgeschlagen.')
    } finally {
      setConfirmDelete(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Bankdaten */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Bankdaten</h2>

        {bankdatenDraft && !privateKey && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            <div className="flex items-center justify-between gap-2 flex-wrap">
              <p>Bankdaten-Antrag liegt vor — Tresor entsperren um einzusehen und anzunehmen (Menü „Tresor").</p>
              <button
                onClick={() => onDraftReject(bankdatenDraft.id)}
                className="px-2 py-1 bg-brand-danger-light text-brand-danger rounded hover:bg-red-200 text-xs font-medium shrink-0"
              >
                Ablehnen
              </button>
            </div>
          </div>
        )}

        {bankdatenDraft && privateKey && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            <div className="flex items-start justify-between gap-2 flex-wrap">
              <div className="space-y-1">
                <p className="font-medium mb-1">Angeforderte Bankdaten:</p>
                {decryptedDraft
                  ? <>
                      {decryptedDraft.account_holder && (
                        <p className="text-xs">
                          <span className="text-brand-text-muted">Kontoinhaber:</span>{' '}
                          {decryptedDraft.account_holder}
                        </p>
                      )}
                      {decryptedDraft.iban && (
                        <p className="text-xs font-mono">
                          <span className="text-brand-text-muted not-italic">IBAN:</span>{' '}
                          {formatIBAN(decryptedDraft.iban)}
                        </p>
                      )}
                    </>
                  : <p className="text-xs text-brand-text-subtle">Wird entschlüsselt…</p>
                }
              </div>
              <div className="flex gap-2 shrink-0">
                <button
                  onClick={() => onDraftAccept(bankdatenDraft.id)}
                  className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 text-xs font-medium"
                >
                  Annehmen
                </button>
                <button
                  onClick={() => onDraftReject(bankdatenDraft.id)}
                  className="px-2 py-1 bg-brand-danger-light text-brand-danger rounded hover:bg-red-200 text-xs font-medium"
                >
                  Ablehnen
                </button>
              </div>
            </div>
          </div>
        )}

        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Kontoinhaber</label>
            <input
              type="text"
              value={form.account_holder || ''}
              onChange={e => onFormChange({ account_holder: e.target.value })}
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">IBAN</label>
            <input
              type="text"
              value={ibanDisplay}
              onChange={handleIbanChange}
              onBlur={handleIbanBlur}
              placeholder="DE89 3704 0044 0532 0130 00"
              maxLength={42}
              className={`w-full border rounded-md px-3 py-2 text-sm font-mono tracking-wider focus:outline-none focus:ring-2 focus:ring-brand-yellow ${
                ibanError ? 'border-brand-danger bg-brand-danger-light' : 'border-brand-border'
              }`}
            />
            {ibanError && <p className="text-xs text-brand-danger mt-1">{ibanError}</p>}
          </div>
          <label className="flex items-center gap-2 cursor-pointer mt-2">
            <input
              type="checkbox"
              checked={form.beitragsfrei || false}
              onChange={e => onFormChange(
                e.target.checked
                  ? { beitragsfrei: true }
                  : { beitragsfrei: false, beitragsfrei_grund: '' },
              )}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Beitragsfrei</span>
          </label>
          {form.beitragsfrei && (
            <div className="mt-2">
              <label className="block text-sm font-medium text-brand-text-muted mb-1">
                Grund für Beitragsfreiheit
              </label>
              <input
                type="text"
                value={form.beitragsfrei_grund || ''}
                onChange={e => onFormChange({ beitragsfrei_grund: e.target.value })}
                placeholder="z. B. kein aktiver Sportler mehr"
                maxLength={200}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              />
            </div>
          )}
        </div>
      </div>

      {/* SEPA-Mandat */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-4">SEPA-Mandat</h2>
        <div className="space-y-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={form.sepa_mandat || false}
              onChange={e => onFormChange({ sepa_mandat: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Mandat erteilt</span>
            {sepaDraft && <span className="text-sm text-brand-text-muted">(Änderung ausstehend)</span>}
          </label>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">
              Datum der Unterschrift
            </label>
            <input
              type="date"
              value={form.sepa_mandat_date || ''}
              onChange={e => onFormChange({ sepa_mandat_date: e.target.value })}
              className="w-full sm:w-auto border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
            <p className="text-xs text-brand-text-subtle mt-1">
              Tag, an dem das Mitglied das SEPA-Mandat unterzeichnet hat (Pflichtfeld für die XML-Erzeugung im Beitragslauf).
            </p>
          </div>

          {!isNew && (
            <div className="mt-4 space-y-2">
              <label className="block text-sm font-medium text-brand-text mb-1">Mandat-Dokument</label>

              {form.sepa_mandat_url && (
                <div className="flex items-center gap-2 flex-wrap">
                  <button
                    onClick={openSepaMandat}
                    className="flex items-center gap-1.5 bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                  >
                    <ExternalLink className="w-4 h-4" />
                    Dokument öffnen
                  </button>
                  {canDeleteSepa && (
                    <button
                      onClick={() => setConfirmDelete(true)}
                      className="flex items-center gap-1.5 bg-brand-danger text-white rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-danger/90 transition-colors"
                    >
                      <Trash2 className="w-4 h-4" />
                      Dokument löschen
                    </button>
                  )}
                </div>
              )}

              <div>
                <input ref={sepaInputRef} type="file" accept=".pdf,image/*" className="hidden" onChange={handleSepaUpload} />
                <button
                  onClick={() => sepaInputRef.current?.click()}
                  disabled={sepaUploading}
                  className="bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow disabled:opacity-40 transition-colors"
                >
                  {sepaUploading ? 'Hochladen…' : form.sepa_mandat_url ? 'Dokument ersetzen' : 'Dokument hochladen'}
                </button>
              </div>

              {sepaUploadError && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 flex-shrink-0" />{sepaUploadError}
                </div>
              )}
              {openError && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 flex-shrink-0" />{openError}
                </div>
              )}
              {deleteError && (
                <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 flex-shrink-0" />{deleteError}
                </div>
              )}
            </div>
          )}

          {sepaDraft && (
            <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span>
                  <span className="font-medium">Angeforderte SEPA-Mandat:</span>{' '}
                  {sepaDraft.new_value ? 'Erteilt' : 'Nicht erteilt'}
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => onDraftAccept(sepaDraft.id)}
                    className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 font-medium text-xs"
                  >
                    Annehmen
                  </button>
                  <button
                    onClick={() => onDraftReject(sepaDraft.id)}
                    className="px-2 py-1 bg-brand-danger-light text-brand-danger rounded hover:opacity-80 font-medium text-xs"
                  >
                    Ablehnen
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {!isNew && (
        <div className="flex items-center gap-3">
          <button
            onClick={onSave}
            disabled={saving}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-green-600">Gespeichert</span>}
          {error && <span className="text-sm text-brand-danger">{error}</span>}
        </div>
      )}

      {/* Delete confirmation modal */}
      {confirmDelete && createPortal(
        <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
            <h2 className="font-semibold text-brand-text mb-2">Dokument löschen</h2>
            <p className="text-sm text-brand-text-muted mb-4">Das SEPA-Mandat-Dokument wirklich löschen?</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setConfirmDelete(false)} className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text">Abbrechen</button>
              <button
                onClick={handleDeleteSepa}
                className="bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors"
              >
                Löschen
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  )
}
