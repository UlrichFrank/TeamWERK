import { useCallback, useEffect, useState } from 'react'
import { Plus, X } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import KaderMemberSearch from '../components/KaderMemberSearch'
import KaderTrainerSearch from '../components/KaderTrainerSearch'
import PersonChip from '../components/PersonChip'
import { useEscapeKey } from '../lib/useEscapeKey'
import { errorData } from '../lib/errors'
import { BTN_DANGER, BTN_PRIMARY, BTN_SECONDARY, HEADER_CTRL, HEADER_PRIMARY } from '../lib/buttonStyles'

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

interface Person {
  id: number
  name: string
  user_id?: number
  status?: string
}

interface PracticeGroup {
  id: number
  season_id: number
  name: string
  members: Person[]
  trainers: Person[]
}

// Übungsgruppen sind Kader ohne Altersklasse, Geschlecht, Jahrgang und ohne
// erweiterten Kader — die Maske zeigt deshalb nur Name, Trainer und Mitglieder.
export default function UebungsgruppenPage() {
  const [groups, setGroups] = useState<PracticeGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [createName, setCreateName] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [renaming, setRenaming] = useState<PracticeGroup | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<PracticeGroup | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [busy, setBusy] = useState<Record<string, boolean>>({})

  useEscapeKey(
    deleteConfirm ? () => setDeleteConfirm(null) :
    renaming ? () => setRenaming(null) :
    showCreate ? () => setShowCreate(false) :
    null
  )

  const load = useCallback(async () => {
    try {
      const res = await api.get('/practice-groups')
      setGroups(res.data?.items ?? [])
      setError(null)
    } catch {
      setError('Übungsgruppen konnten nicht geladen werden.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])
  useLiveUpdates(event => { if (event === 'practice-groups') load() })

  const handleCreate = async () => {
    const name = createName.trim()
    if (!name) return
    setCreating(true)
    try {
      await api.post('/practice-groups', { name })
      setCreateName('')
      setShowCreate(false)
      setError(null)
      await load()
    } catch (e) {
      setError(errorData<{ error?: string }>(e)?.error ?? 'Anlegen fehlgeschlagen.')
    } finally {
      setCreating(false)
    }
  }

  const handleRename = async () => {
    if (!renaming) return
    const name = renameValue.trim()
    if (!name) return
    try {
      await api.put(`/practice-groups/${renaming.id}`, { name })
      setRenaming(null)
      setError(null)
      await load()
    } catch (e) {
      setError(errorData<{ error?: string }>(e)?.error ?? 'Umbenennen fehlgeschlagen.')
    }
  }

  const handleDelete = async () => {
    if (!deleteConfirm) return
    setDeleting(true)
    try {
      await api.delete(`/practice-groups/${deleteConfirm.id}`)
      setDeleteConfirm(null)
      setError(null)
      await load()
    } catch (e) {
      const data = errorData(e) as { training_count?: number; error?: string } | undefined
      setError(data?.training_count
        ? `Die Gruppe hat noch ${data.training_count} Trainingstermin(e) — erst die Termine löschen.`
        : data?.error ?? 'Löschen fehlgeschlagen.')
      setDeleteConfirm(null)
    } finally {
      setDeleting(false)
    }
  }

  const mutate = async (key: string, body: Record<string, number[]>) => {
    setBusy(prev => ({ ...prev, [key]: true }))
    try {
      await api.put(`/practice-groups/${key.split('-')[0]}`, body)
      await load()
    } catch {
      setError('Änderung fehlgeschlagen.')
    } finally {
      setBusy(prev => ({ ...prev, [key]: false }))
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-semibold text-brand-text">Übungsgruppen</h1>
        <button
          onClick={() => { setShowCreate(true); setCreateName('') }}
          className={`${HEADER_CTRL} ${HEADER_PRIMARY} whitespace-nowrap`}
        >
          <Plus className="w-4 h-4" />
          Neue Gruppe
        </button>
      </div>

      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4">
          {error}
        </div>
      )}

      {loading && <p className="text-sm text-brand-text-muted">Laden…</p>}

      {!loading && groups.length === 0 && (
        <div className="bg-brand-surface-card rounded-xl border-t-4 border-brand-yellow p-8 text-center">
          <p className="text-brand-text-muted text-sm">
            Noch keine Übungsgruppen in der aktiven Saison. Eine Übungsgruppe ist ein
            benanntes Trainingsgefäß ohne Altersklasse und Jahrgang — für Torwart-,
            Athletik- oder Sichtungstraining.
          </p>
        </div>
      )}

      {groups.map(g => (
        <div key={g.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow mb-3">
          <div className="px-5 py-3 border-b border-brand-border-subtle flex items-center justify-between gap-2">
            <button
              onClick={() => { setRenaming(g); setRenameValue(g.name) }}
              className="font-semibold text-sm truncate text-brand-text hover:text-brand-text-muted transition-colors text-left"
              title="Umbenennen"
            >
              {g.name}
            </button>
            <div className="flex items-center gap-2 shrink-0">
              <span className="text-xs text-brand-text-muted">{g.members.length} Mitgl.</span>
              <button
                onClick={() => setDeleteConfirm(g)}
                aria-label="Übungsgruppe löschen"
                title="Übungsgruppe löschen"
                className="text-brand-text-subtle hover:text-brand-danger transition-colors p-0.5 rounded"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          <div className="px-5 py-2 border-t border-brand-border-subtle">
            <KaderTrainerSearch
              assignedTrainers={g.trainers}
              onAdd={memberId => mutate(`${g.id}-trainer-add-${memberId}`, { trainers_add: [memberId] })}
              onRemove={memberId => mutate(`${g.id}-trainer-remove-${memberId}`, { trainers_remove: [memberId] })}
            />
          </div>

          <div className="px-5 pt-2 pb-2 border-t border-brand-border-subtle">
            <KaderMemberSearch
              kaderId={g.id}
              basePath="/practice-groups"
              showAgeFilter={false}
              onMemberAdded={load}
            />
          </div>

          {g.members.length === 0 ? (
            <p className="text-xs text-brand-text-subtle italic px-5 py-3">Keine Mitglieder</p>
          ) : (
            <ul className="divide-y divide-brand-border-subtle px-5 pb-4">
              {g.members.map(m => (
                <li key={m.id} className="flex items-center justify-between py-2 gap-2">
                  <span className="text-sm text-brand-text flex items-center gap-1.5 flex-wrap">
                    <PersonChip userId={m.user_id} name={m.name} />
                  </span>
                  <button
                    onClick={() => mutate(`${g.id}-member-remove-${m.id}`, { members_remove: [m.id] })}
                    disabled={busy[`${g.id}-member-remove-${m.id}`]}
                    className="text-brand-text-muted hover:text-brand-danger transition-colors disabled:opacity-40 p-1 rounded"
                    aria-label="Mitglied entfernen"
                    title="Mitglied entfernen"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      ))}

      {showCreate && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
            <h2 className="text-lg font-semibold text-brand-text mb-4">Neue Übungsgruppe</h2>
            <label className="block text-xs font-medium text-brand-text-muted mb-1">Name</label>
            <input
              autoFocus
              value={createName}
              onChange={e => setCreateName(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') handleCreate() }}
              placeholder="z. B. Torwarttraining"
              className={INPUT}
            />
            <div className="flex justify-end gap-2 mt-5">
              <button
                onClick={() => setShowCreate(false)}
                className={BTN_SECONDARY}
              >
                Abbrechen
              </button>
              <button onClick={handleCreate} disabled={creating || !createName.trim()} className={BTN_PRIMARY}>
                {creating ? 'Anlegen…' : 'Anlegen'}
              </button>
            </div>
          </div>
        </div>
      )}

      {renaming && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
            <h2 className="text-lg font-semibold text-brand-text mb-4">Übungsgruppe umbenennen</h2>
            <input
              autoFocus
              value={renameValue}
              onChange={e => setRenameValue(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') handleRename() }}
              className={INPUT}
            />
            <div className="flex justify-end gap-2 mt-5">
              <button
                onClick={() => setRenaming(null)}
                className={BTN_SECONDARY}
              >
                Abbrechen
              </button>
              <button onClick={handleRename} disabled={!renameValue.trim()} className={BTN_PRIMARY}>
                Speichern
              </button>
            </div>
          </div>
        </div>
      )}

      {deleteConfirm && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
            <h2 className="text-lg font-semibold text-brand-text mb-2">Übungsgruppe löschen</h2>
            <p className="text-sm text-brand-text-muted mb-5">
              „{deleteConfirm.name}" wird gelöscht. Solange Trainingstermine an der Gruppe
              hängen, ist das nicht möglich.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setDeleteConfirm(null)}
                className={BTN_SECONDARY}
              >
                Abbrechen
              </button>
              <button onClick={handleDelete} disabled={deleting} className={BTN_DANGER}>
                {deleting ? 'Löschen…' : 'Löschen'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
