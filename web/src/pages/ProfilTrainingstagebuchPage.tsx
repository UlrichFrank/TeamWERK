import { useCallback, useEffect, useState } from 'react'
import { Plus, ImageOff } from 'lucide-react'
import { api } from '../lib/api'
import ActionMenu from '../components/ActionMenu'
import TrainingDiaryEntryForm, { type DiarySubmitPayload } from '../components/TrainingDiaryEntryForm'
import TrainingDiaryProofView from '../components/TrainingDiaryProofView'
import { deleteProof, uploadProof } from '../lib/trainingDiaryProof'
import {
  RETENTION_HINT,
  fmtDate,
  kindLabel,
  type DiaryEntry,
} from '../lib/trainingDiary'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { BTN_PRIMARY } from '../lib/buttonStyles'

// Nachweis-Darstellung. Der 'purged'-Zweig löst bewusst KEINEN Bildabruf aus —
// er würde zwangsläufig 410 liefern und im UI als kaputtes Bild erscheinen.
function ProofBlock({
  entry,
  readOnly,
  onDelete,
}: {
  entry: DiaryEntry
  readOnly?: boolean
  onDelete: () => void
}) {
  if (entry.proof_status === 'purged') {
    return (
      <p className="mt-2 inline-flex items-center gap-1 text-xs text-brand-text-muted italic">
        <ImageOff className="w-4 h-4" />
        Nachweis gelöscht (90 Tage nach Saisonende)
      </p>
    )
  }
  if (entry.proof_status !== 'present') return null

  return (
    <div className="mt-2">
      <TrainingDiaryProofView
        entry={entry}
        thumbClassName="max-h-48 rounded-md border border-brand-border-subtle"
      />
      {!readOnly && (
        <button
          type="button"
          onClick={onDelete}
          className="mt-1 block text-xs text-brand-danger hover:underline"
        >
          Nachweis entfernen
        </button>
      )}
    </div>
  )
}

// forcedMemberId schaltet auf die Fremdsicht um (Eltern auf der Kind-Seite):
// gelesen wird dann /members/{id}/training-diary und die Ansicht ist
// schreibgeschützt — Schreibrecht hat serverseitig ausschließlich der
// Eigentümer, die UI soll das nicht anders suggerieren.
export function ProfilTrainingstagebuchContent({ forcedMemberId }: { forcedMemberId?: number } = {}) {
  const readOnly = forcedMemberId != null
  const [entries, setEntries] = useState<DiaryEntry[]>([])
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<DiaryEntry | null>(null)
  const [pendingFile, setPendingFile] = useState<File | null>(null)
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => {
    const url = forcedMemberId != null
      ? `/members/${forcedMemberId}/training-diary`
      : '/training-diary'
    api
      .get<{ items: DiaryEntry[] }>(url)
      .then(r => setEntries(r.data.items ?? []))
      .catch(() => setError('Das Trainingstagebuch konnte nicht geladen werden.'))
      .finally(() => setLoaded(true))
  }, [forcedMemberId])

  useEffect(load, [load])
  useLiveUpdates(event => {
    if (event === 'training-diary-changed') load()
  })

  async function handleSubmit(payload: DiarySubmitPayload) {
    setBusy(true)
    setError('')
    try {
      const entry = editing
        ? (await api.put<DiaryEntry>(`/training-diary/${editing.id}`, payload)).data
        : (await api.post<DiaryEntry>('/training-diary', payload)).data
      if (pendingFile) await uploadProof(entry.id, pendingFile)
      setFormOpen(false)
      setEditing(null)
      setPendingFile(null)
      load()
    } catch {
      setError('Speichern fehlgeschlagen. Bitte prüf deine Eingaben.')
    } finally {
      setBusy(false)
    }
  }

  async function handleDelete(entry: DiaryEntry) {
    if (!window.confirm('Diesen Eintrag wirklich löschen?')) return
    try {
      await api.delete(`/training-diary/${entry.id}`)
      load()
    } catch {
      setError('Löschen fehlgeschlagen.')
    }
  }

  async function handleProofUpload(entry: DiaryEntry, file: File) {
    try {
      await uploadProof(entry.id, file)
      load()
    } catch {
      setError('Der Nachweis konnte nicht hochgeladen werden. Erlaubt sind JPG, PNG, WebP und PDF (max. 1 MB).')
    }
  }

  async function handleProofDelete(entry: DiaryEntry) {
    try {
      await deleteProof(entry.id)
      load()
    } catch {
      setError('Der Nachweis konnte nicht entfernt werden.')
    }
  }

  return (
    <div className="space-y-4">
      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
          {error}
        </div>
      )}

      {!formOpen && !readOnly && (
        <button
          type="button"
          onClick={() => {
            setEditing(null)
            setPendingFile(null)
            setFormOpen(true)
          }}
          className={`${BTN_PRIMARY} inline-flex items-center gap-1`}
        >
          <Plus className="w-5 h-5" />
          Einheit erfassen
        </button>
      )}

      {formOpen && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <TrainingDiaryEntryForm
            initial={editing ?? undefined}
            busy={busy}
            onSubmit={handleSubmit}
            onFileSelect={setPendingFile}
            onCancel={() => {
              setFormOpen(false)
              setEditing(null)
              setPendingFile(null)
            }}
          />
        </div>
      )}

      {loaded && entries.length === 0 && (
        <p className="text-sm text-brand-text-muted">
          Noch keine Einheiten erfasst. {readOnly ? '' : RETENTION_HINT}
        </p>
      )}

      <div className="space-y-2">
        {entries.map(entry => (
          <div
            key={entry.id}
            className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4"
          >
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-sm font-medium text-brand-text">
                  {fmtDate(entry.trained_on)} · {kindLabel(entry)}
                </p>
                <p className="text-sm text-brand-text-muted">
                  {entry.duration_min} min · RPE {entry.rpe}
                </p>
                {entry.note && (
                  <p className="mt-1 text-sm text-brand-text italic">{entry.note}</p>
                )}
              </div>
              {!readOnly && (
                <ActionMenu
                  actions={[
                    {
                      label: 'Bearbeiten',
                      onClick: () => {
                        setEditing(entry)
                        setPendingFile(null)
                        setFormOpen(true)
                      },
                    },
                    { label: 'Löschen', onClick: () => handleDelete(entry), variant: 'danger' },
                  ]}
                />
              )}
            </div>

            <ProofBlock
              entry={entry}
              readOnly={readOnly}
              onDelete={() => handleProofDelete(entry)}
            />

            {entry.proof_status === 'none' && !readOnly && (
              <label className="mt-2 block text-xs text-brand-text-muted">
                Nachweis nachreichen
                <input
                  type="file"
                  accept="image/jpeg,image/png,image/webp,application/pdf"
                  aria-label="Nachweis nachreichen"
                  onChange={e => {
                    const file = e.target.files?.[0]
                    if (file) handleProofUpload(entry, file)
                  }}
                  className="mt-1 block w-full text-sm text-brand-text"
                />
              </label>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

export default function ProfilTrainingstagebuchPage() {
  return (
    <div>
      <h1 className="text-2xl font-semibold text-brand-text mb-4">Trainingstagebuch</h1>
      <p className="mb-4 text-sm text-brand-text-muted">{RETENTION_HINT}</p>
      <ProfilTrainingstagebuchContent />
    </div>
  )
}
