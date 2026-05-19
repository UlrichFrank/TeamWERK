import { useEffect, useState } from 'react'
import { api } from '../lib/api'

interface DutyType {
  id: number
  name: string
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
}

interface TemplateItem {
  duty_type_id: number
  anchor: 'start' | 'end'
  offset_minutes: number
  slots_count: number
  role_desc: string
}

interface Template {
  id?: number
  name: string
  game_duration_minutes: number
  items: TemplateItem[]
}

function newItem(): TemplateItem {
  return { duty_type_id: 0, anchor: 'start', offset_minutes: 0, slots_count: 1, role_desc: '' }
}

export default function AdminGameTemplatePage() {
  const [template, setTemplate] = useState<Template>({
    name: 'Heimspiel Standard',
    game_duration_minutes: 90,
    items: [],
  })
  const [dutyTypes, setDutyTypes] = useState<DutyType[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([
      api.get('/admin/game-template').then(r => setTemplate(r.data)),
      api.get('/admin/duty-types').then(r => setDutyTypes(r.data ?? [])),
    ]).finally(() => setLoading(false))
  }, [])

  const updateItem = (i: number, patch: Partial<TemplateItem>) => {
    setTemplate(t => ({
      ...t,
      items: t.items.map((item, idx) => idx === i ? { ...item, ...patch } : item),
    }))
  }

  const addItem = () => {
    setTemplate(t => ({ ...t, items: [...t.items, newItem()] }))
  }

  const removeItem = (i: number) => {
    setTemplate(t => ({ ...t, items: t.items.filter((_, idx) => idx !== i) }))
  }

  const handleSave = async () => {
    setSaveError(null)
    const invalid = template.items.filter(it => it.duty_type_id === 0)
    if (invalid.length > 0) {
      setSaveError('Bitte für alle Einträge einen Diensttyp auswählen.')
      return
    }
    setSaving(true)
    setSaved(false)
    try {
      await api.put('/admin/game-template', template)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch {
      setSaveError('Speichern fehlgeschlagen. Bitte Seite neu laden und erneut versuchen.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="text-gray-400 text-sm">Laden…</div>

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">Spiel-Vorlage (Template)</h1>
      <p className="text-sm text-gray-500 mb-6">
        Diese Vorlage bestimmt, welche Dienste beim Anlegen eines Heimspiels automatisch generiert werden.
        Zeitangaben sind relativ zum Anpfiff (Anker: Start) oder Spielende (Anker: Ende).
      </p>

      {/* Template name */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-5 mb-5">
        <label className="block text-sm font-medium mb-1">Name der Vorlage</label>
        <input
          type="text"
          value={template.name}
          onChange={e => setTemplate(t => ({ ...t, name: e.target.value }))}
          className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
        />
      </div>

      {/* Items */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-5">
        <div className="flex items-center justify-between px-5 py-3 border-b">
          <h2 className="font-semibold">Dienst-Einträge</h2>
          <button
            onClick={addItem}
            className="text-sm bg-brand-yellow text-black px-3 py-1.5 rounded-md font-medium hover:bg-black hover:text-brand-yellow transition-colors"
          >
            + Eintrag hinzufügen
          </button>
        </div>

        {template.items.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-8 italic">
            Keine Einträge — klicke auf „+ Eintrag hinzufügen"
          </p>
        ) : (
          <div className="divide-y">
            {template.items.map((item, i) => (
              <div key={i} className="p-4 grid grid-cols-[1fr_auto_auto_auto_auto_auto] gap-3 items-center">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Diensttyp</label>
                  <select
                    value={item.duty_type_id}
                    onChange={e => {
                      const id = Number(e.target.value)
                      const dt = dutyTypes.find(d => d.id === id)
                      updateItem(i, {
                        duty_type_id: id,
                        ...(dt ? { anchor: dt.default_anchor, offset_minutes: dt.default_offset_minutes } : {}),
                      })
                    }}
                    className="w-full border rounded-md px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
                  >
                    <option value={0}>Auswählen…</option>
                    {dutyTypes.map(dt => <option key={dt.id} value={dt.id}>{dt.name}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Anker</label>
                  <select
                    value={item.anchor}
                    onChange={e => updateItem(i, { anchor: e.target.value as 'start' | 'end' })}
                    className="border rounded-md px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
                  >
                    <option value="start">Anpfiff</option>
                    <option value="end">Spielende</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Versatz (min)</label>
                  <input
                    type="number"
                    value={item.offset_minutes}
                    onChange={e => updateItem(i, { offset_minutes: Number(e.target.value) })}
                    className="w-20 border rounded-md px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Personen</label>
                  <input
                    type="number"
                    min={1}
                    value={item.slots_count}
                    onChange={e => updateItem(i, { slots_count: Number(e.target.value) })}
                    className="w-16 border rounded-md px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Rollenbezeichnung</label>
                  <input
                    type="text"
                    value={item.role_desc}
                    onChange={e => updateItem(i, { role_desc: e.target.value })}
                    placeholder="Optional"
                    className="w-36 border rounded-md px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
                  />
                </div>
                <button
                  onClick={() => removeItem(i)}
                  className="text-gray-400 hover:text-brand-error transition-colors mt-4 px-1"
                  title="Eintrag entfernen"
                >✕</button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Versatz info */}
      <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-blue-700 mb-5">
        <strong>Versatz:</strong> Negative Werte = vor dem Anker (z.B. −60 = 60 min vor Anpfiff).
        Positive Werte = nach dem Anker (z.B. +15 = 15 min nach Spielende).
      </div>

      {saveError && (
        <p className="text-brand-error text-sm mb-3">{saveError}</p>
      )}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={saving}
          className="bg-brand-yellow text-black px-6 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-50"
        >
          {saving ? 'Speichern…' : 'Vorlage speichern'}
        </button>
        {saved && <span className="text-brand-success text-sm">✓ Gespeichert</span>}
      </div>
    </div>
  )
}
