import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'

interface DutyTemplate {
  id: number
  name: string
  template_type: 'heim' | 'auswärts' | 'generisch'
  game_duration_minutes: number
  item_count: number
}

const typeLabel: Record<string, string> = {
  heim: 'Heim',
  'auswärts': 'Auswärts',
  generisch: 'Generisch',
}

const typeBadge: Record<string, string> = {
  heim: 'bg-brand-blue/10 text-brand-blue',
  'auswärts': 'bg-brand-warning-light text-brand-text',
  generisch: 'bg-brand-border-subtle text-brand-text-muted',
}

export default function AdminDutyTemplatesPage() {
  const navigate = useNavigate()
  const [templates, setTemplates] = useState<DutyTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [deleteError, setDeleteError] = useState('')
  const [newName, setNewName] = useState('')
  const [newType, setNewType] = useState<'heim' | 'auswärts' | 'generisch'>('heim')
  const [createError, setCreateError] = useState('')

  useEffect(() => {
    api.get('/admin/duty-templates').then(r => {
      setTemplates(r.data ?? [])
    }).finally(() => setLoading(false))
  }, [])

  const typeCounts = templates.reduce<Record<string, number>>((acc, t) => {
    acc[t.template_type] = (acc[t.template_type] || 0) + 1
    return acc
  }, {})

  const handleDelete = async (id: number) => {
    setDeleteError('')
    try {
      await api.delete(`/admin/duty-templates/${id}`)
      setTemplates(prev => prev.filter(t => t.id !== id))
      setDeleteId(null)
    } catch {
      setDeleteError('Löschen fehlgeschlagen.')
    }
  }

  const handleCreate = async () => {
    if (!newName.trim()) {
      setCreateError('Name darf nicht leer sein.')
      return
    }
    setCreateError('')
    try {
      const r = await api.post('/admin/duty-templates', {
        name: newName.trim(),
        template_type: newType,
        game_duration_minutes: 60,
      })
      navigate(`/admin/dienstplan-vorlagen/${r.data.id}`)
    } catch {
      setCreateError('Erstellen fehlgeschlagen.')
    }
  }

  if (loading) return <div className="text-brand-text-muted text-sm">Laden…</div>

  return (
    <div className="max-w-4xl">
      <div className="flex items-center justify-between mb-6 flex-wrap gap-3">
        <h1 className="text-2xl font-bold">Dienstplan-Vorlagen</h1>
      </div>

      {/* Warnung bei doppeltem Typ */}
      {Object.entries(typeCounts).some(([, count]) => count > 1) && (
        <div className="mb-4 p-3 bg-brand-warning-light border border-brand-warning/40 rounded-lg text-sm text-brand-text flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 text-brand-warning mt-0.5 flex-shrink-0" />
          Achtung: Es gibt mehrere Vorlagen des gleichen Typs. Bei der Slot-Generierung wird immer die erste verwendet (niedrigste ID).
        </div>
      )}

      {/* Tabelle */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-6">
        {templates.length === 0 ? (
          <p className="text-sm text-brand-text-subtle text-center py-10 italic">
            Keine Vorlagen vorhanden — lege eine neue an.
          </p>
        ) : (
          <>
            {/* Desktop */}
            <table className="hidden sm:table w-full text-sm">
              <thead>
                <tr>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Typ</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Dauer</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Einträge</th>
                  <th className="bg-brand-surface-card px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-subtle">
                {templates.map(t => (
                  <tr key={t.id} className="hover:bg-brand-table-select transition-colors">
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/dienstplan-vorlagen/${t.id}`}
                        className="font-medium text-brand-text hover:text-brand-blue hover:underline"
                      >
                        {t.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${typeBadge[t.template_type] ?? ''}`}>
                        {typeLabel[t.template_type] ?? t.template_type}
                      </span>
                      {typeCounts[t.template_type] > 1 && (
                        <AlertTriangle className="inline w-3.5 h-3.5 ml-1 text-brand-warning" aria-label="Doppelter Typ" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-brand-text-muted">{t.game_duration_minutes} min</td>
                    <td className="px-4 py-3 text-brand-text-muted">{t.item_count}</td>
                    <td className="px-4 py-3 text-right">
                      {deleteId === t.id ? (
                        <span className="flex items-center gap-2 justify-end">
                          <span className="text-xs text-brand-text-muted">Wirklich löschen?</span>
                          <button onClick={() => handleDelete(t.id)} className="text-xs text-brand-danger hover:text-brand-danger/80 font-medium">Ja</button>
                          <button onClick={() => setDeleteId(null)} className="text-xs text-brand-text-muted hover:text-brand-text">Abbrechen</button>
                        </span>
                      ) : (
                        <button onClick={() => setDeleteId(t.id)} className="text-xs text-brand-danger hover:text-brand-danger/80">Löschen</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Mobile */}
            <div className="sm:hidden divide-y divide-brand-border-subtle">
              {templates.map(t => (
                <div key={t.id} className="p-4">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <Link
                        to={`/admin/dienstplan-vorlagen/${t.id}`}
                        className="font-medium text-brand-text hover:underline"
                      >
                        {t.name}
                      </Link>
                      <div className="flex items-center gap-2 mt-1">
                        <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${typeBadge[t.template_type] ?? ''}`}>
                          {typeLabel[t.template_type] ?? t.template_type}
                        </span>
                        <span className="text-xs text-brand-text-muted">{t.item_count} Einträge · {t.game_duration_minutes} min</span>
                      </div>
                    </div>
                    <button
                      onClick={() => setDeleteId(deleteId === t.id ? null : t.id)}
                      className="text-xs text-brand-danger hover:text-brand-danger/80 shrink-0"
                    >Löschen</button>
                  </div>
                  {deleteId === t.id && (
                    <div className="mt-2 flex items-center gap-2 text-xs">
                      <span className="text-brand-text-muted">Wirklich löschen?</span>
                      <button onClick={() => handleDelete(t.id)} className="text-brand-danger font-medium">Ja</button>
                      <button onClick={() => setDeleteId(null)} className="text-brand-text-muted">Abbrechen</button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {deleteError && (
        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4">
          {deleteError}
        </p>
      )}

      {/* Neue Vorlage anlegen */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-5">
        <h2 className="font-semibold text-brand-text mb-4">Neue Vorlage anlegen</h2>
        <div className="grid grid-cols-1 sm:grid-cols-[1fr_auto_auto] gap-3 items-end">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Name</label>
            <input
              type="text"
              value={newName}
              onChange={e => { setNewName(e.target.value); setCreateError('') }}
              onKeyDown={e => e.key === 'Enter' && handleCreate()}
              placeholder="z.B. Heimspiel Standard"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Typ</label>
            <select
              value={newType}
              onChange={e => setNewType(e.target.value as typeof newType)}
              className="border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            >
              <option value="heim">Heim</option>
              <option value="auswärts">Auswärts</option>
              <option value="generisch">Generisch</option>
            </select>
          </div>
          <button
            onClick={handleCreate}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            Anlegen
          </button>
        </div>
        {createError && (
          <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mt-3">
            {createError}
          </p>
        )}
      </div>
    </div>
  )
}
