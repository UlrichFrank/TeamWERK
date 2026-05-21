import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
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
  heim: 'bg-blue-100 text-blue-700',
  'auswärts': 'bg-orange-100 text-orange-700',
  generisch: 'bg-gray-100 text-gray-600',
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
        game_duration_minutes: 90,
      })
      navigate(`/admin/dienstplan-vorlagen/${r.data.id}`)
    } catch {
      setCreateError('Erstellen fehlgeschlagen.')
    }
  }

  if (loading) return <div className="text-gray-400 text-sm">Laden…</div>

  return (
    <div className="max-w-4xl">
      <div className="flex items-center justify-between mb-6 flex-wrap gap-3">
        <h1 className="text-2xl font-bold">Dienstplan-Vorlagen</h1>
      </div>

      {/* Warnung bei doppeltem Typ */}
      {Object.entries(typeCounts).some(([, count]) => count > 1) && (
        <div className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-700">
          Achtung: Es gibt mehrere Vorlagen des gleichen Typs. Bei der Slot-Generierung wird immer die erste verwendet (niedrigste ID).
        </div>
      )}

      {/* Tabelle */}
      <div className="bg-white rounded-xl shadow border border-gray-200 overflow-hidden mb-6">
        {templates.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-10 italic">
            Keine Vorlagen vorhanden — lege eine neue an.
          </p>
        ) : (
          <>
            {/* Desktop */}
            <table className="hidden sm:table w-full text-sm">
              <thead>
                <tr className="bg-gray-50 text-left text-xs text-gray-500 uppercase tracking-wide border-b">
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Typ</th>
                  <th className="px-4 py-3">Dauer</th>
                  <th className="px-4 py-3">Einträge</th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {templates.map(t => (
                  <tr key={t.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/dienstplan-vorlagen/${t.id}`}
                        className="font-medium text-gray-900 hover:text-brand-blue hover:underline"
                      >
                        {t.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${typeBadge[t.template_type] ?? ''}`}>
                        {typeLabel[t.template_type] ?? t.template_type}
                      </span>
                      {typeCounts[t.template_type] > 1 && (
                        <span className="ml-1 text-amber-500 text-xs" title="Doppelter Typ">⚠</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-gray-600">{t.game_duration_minutes} min</td>
                    <td className="px-4 py-3 text-gray-600">{t.item_count}</td>
                    <td className="px-4 py-3 text-right">
                      {deleteId === t.id ? (
                        <span className="flex items-center gap-2 justify-end">
                          <span className="text-xs text-gray-500">Wirklich löschen?</span>
                          <button
                            onClick={() => handleDelete(t.id)}
                            className="text-xs text-red-600 hover:text-red-800 font-medium"
                          >Ja</button>
                          <button
                            onClick={() => setDeleteId(null)}
                            className="text-xs text-gray-500 hover:text-gray-700"
                          >Abbrechen</button>
                        </span>
                      ) : (
                        <button
                          onClick={() => setDeleteId(t.id)}
                          className="text-xs text-red-500 hover:text-red-700"
                        >Löschen</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Mobile */}
            <div className="sm:hidden divide-y divide-gray-100">
              {templates.map(t => (
                <div key={t.id} className="p-4">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <Link
                        to={`/admin/dienstplan-vorlagen/${t.id}`}
                        className="font-medium text-gray-900 hover:underline"
                      >
                        {t.name}
                      </Link>
                      <div className="flex items-center gap-2 mt-1">
                        <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${typeBadge[t.template_type] ?? ''}`}>
                          {typeLabel[t.template_type] ?? t.template_type}
                        </span>
                        <span className="text-xs text-gray-500">{t.item_count} Einträge · {t.game_duration_minutes} min</span>
                      </div>
                    </div>
                    <button
                      onClick={() => setDeleteId(deleteId === t.id ? null : t.id)}
                      className="text-xs text-red-500 hover:text-red-700 shrink-0"
                    >Löschen</button>
                  </div>
                  {deleteId === t.id && (
                    <div className="mt-2 flex items-center gap-2 text-xs">
                      <span className="text-gray-500">Wirklich löschen?</span>
                      <button onClick={() => handleDelete(t.id)} className="text-red-600 font-medium">Ja</button>
                      <button onClick={() => setDeleteId(null)} className="text-gray-500">Abbrechen</button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {deleteError && <p className="text-sm text-red-600 mb-4">{deleteError}</p>}

      {/* Neue Vorlage anlegen */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-5">
        <h2 className="font-semibold text-gray-700 mb-4">Neue Vorlage anlegen</h2>
        <div className="grid grid-cols-1 sm:grid-cols-[1fr_auto_auto] gap-3 items-end">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              type="text"
              value={newName}
              onChange={e => { setNewName(e.target.value); setCreateError('') }}
              onKeyDown={e => e.key === 'Enter' && handleCreate()}
              placeholder="z.B. Heimspiel Standard"
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Typ</label>
            <select
              value={newType}
              onChange={e => setNewType(e.target.value as typeof newType)}
              className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
            >
              <option value="heim">Heim</option>
              <option value="auswärts">Auswärts</option>
              <option value="generisch">Generisch</option>
            </select>
          </div>
          <button
            onClick={handleCreate}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
          >
            Anlegen
          </button>
        </div>
        {createError && <p className="text-sm text-red-600 mt-2">{createError}</p>}
      </div>
    </div>
  )
}
