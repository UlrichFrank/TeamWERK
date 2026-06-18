import { useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { ExternalLink, Trash2, AlertTriangle } from 'lucide-react'
import { api } from '../../lib/api'
import { useAuth } from '../../contexts/AuthContext'

interface Member {
  dsgvo_verarbeitung?: boolean
  dsgvo_verarbeitung_date?: string
  dsgvo_weitergabe?: boolean
  dsgvo_weitergabe_date?: string
  sepa_mandat?: boolean
  sepa_mandat_date?: string
  sepa_mandat_url?: string
}

interface Draft {
  id: number
  field_name: string
  old_value: any
  new_value: any
}

interface Props {
  memberId: number
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

export default function MemberDatenschutzTab({ memberId, form, isNew, drafts, onFormChange, onDraftAccept, onDraftReject, onSave, saving, saved, error }: Props) {
  const { user, hasCapability } = useAuth()
  const sepaInputRef = useRef<HTMLInputElement>(null)
  const [sepaUploading, setSepaUploading] = useState(false)
  const [sepaUploadError, setSepaUploadError] = useState('')
  const [openError, setOpenError] = useState('')
  const [deleteError, setDeleteError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)

  const dsgvoDraft = drafts.find(d => d.field_name === 'dsgvo')
  const sepaDraft = drafts.find(d => d.field_name === 'sepa_mandat')

  const canDeleteSepa = hasCapability('manage_members') || user?.isParent === true

  const MAX_SEPA_BYTES = 2 * 1024 * 1024

  const handleSepaUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || isNew) return
    if (file.size > MAX_SEPA_BYTES) {
      setSepaUploadError('Die Datei ist zu groß. Maximal erlaubt sind 2 MB.')
      if (sepaInputRef.current) sepaInputRef.current.value = ''
      return
    }
    setSepaUploading(true)
    setSepaUploadError('')
    try {
      const formData = new FormData()
      formData.append('file', file)
      const { data } = await api.post<{ sepa_mandat_url: string }>(
        `/upload/sepa-mandat/${memberId}`,
        formData,
        { headers: { 'Content-Type': 'multipart/form-data' } }
      )
      onFormChange({ sepa_mandat_url: data.sepa_mandat_url })
    } catch (err: any) {
      const msg: string = err?.response?.data ?? ''
      setSepaUploadError(
        msg.includes('too_large')
          ? 'Die Datei ist zu groß. Maximal erlaubt sind 2 MB.'
          : 'Hochladen fehlgeschlagen.'
      )
    } finally {
      setSepaUploading(false)
      if (sepaInputRef.current) sepaInputRef.current.value = ''
    }
  }

  const openSepaMandat = async () => {
    setOpenError('')
    const tab = window.open('about:blank', '_blank')
    try {
      const { data } = await api.get<{ token: string }>(`/members/${memberId}/sepa-mandat/download-token`)
      if (tab) tab.location.href = `/api/members/${memberId}/sepa-mandat/download?token=${data.token}`
    } catch {
      if (tab) tab.close()
      setOpenError('Dokument konnte nicht geöffnet werden.')
    }
  }

  const handleDeleteSepa = async () => {
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
      {/* DSGVO */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-4">Datenschutz (DSGVO)</h2>
        <div className="space-y-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={form.dsgvo_verarbeitung || false}
              onChange={e => onFormChange({ dsgvo_verarbeitung: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Datenverarbeitung eingewilligt</span>
            {dsgvoDraft && <span className="text-sm text-brand-text-muted">(Änderung ausstehend)</span>}
          </label>
          {form.dsgvo_verarbeitung_date && (
            <p className="text-xs text-brand-text-muted">seit {form.dsgvo_verarbeitung_date}</p>
          )}

          <label className="flex items-center gap-2 cursor-pointer mt-4">
            <input
              type="checkbox"
              checked={form.dsgvo_weitergabe || false}
              onChange={e => onFormChange({ dsgvo_weitergabe: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Datenweitergabe eingewilligt</span>
            {dsgvoDraft && <span className="text-sm text-brand-text-muted">(Änderung ausstehend)</span>}
          </label>
          {form.dsgvo_weitergabe_date && (
            <p className="text-xs text-brand-text-muted">seit {form.dsgvo_weitergabe_date}</p>
          )}

          {dsgvoDraft && (
            <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span>
                  <span className="font-medium">Angeforderte DSGVO-Änderung:</span>{' '}
                  Verarbeitung: {dsgvoDraft.new_value?.verarbeitung ? 'Ja' : 'Nein'}, Weitergabe: {dsgvoDraft.new_value?.weitergabe ? 'Ja' : 'Nein'}
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => onDraftAccept(dsgvoDraft.id)}
                    className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 font-medium text-xs"
                  >
                    Annehmen
                  </button>
                  <button
                    onClick={() => onDraftReject(dsgvoDraft.id)}
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
          {form.sepa_mandat_date && (
            <p className="text-xs text-brand-text-muted">seit {form.sepa_mandat_date}</p>
          )}

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

      {/* Save Button */}
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
