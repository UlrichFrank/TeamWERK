import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Trash2, Check } from 'lucide-react'
import { api } from '../lib/api'
import ActionMenu from '../components/ActionMenu'
import OffsetInput from '../components/OffsetInput'
import DurationInput from '../components/DurationInput'
import { AUDIENCE_OPTIONS } from '../lib/constants'

interface DutyType {
  id: number
  name: string
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
  audiences?: string[] | null
}

interface TemplateItem {
  duty_type_id: number
  anchor: 'start' | 'end'
  offset_minutes: number
  slots_count: number
  audiences?: string[] | null
}

interface Template {
  id: number
  name: string
  template_type: 'heim' | 'auswärts' | 'generisch'
  duration_minutes: number
  items: TemplateItem[]
}

function newItem(): TemplateItem {
  return { duty_type_id: 0, anchor: 'start', offset_minutes: 0, slots_count: 1, audiences: [] }
}

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const INPUT_SM = 'w-full border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow'

export default function AdminDutyTemplateDetailPage() {
  const { id } = useParams<{ id: string }>()

  const [template, setTemplate] = useState<Template | null>(null)
  const [dutyTypes, setDutyTypes] = useState<DutyType[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [saveError, setSaveError] = useState('')

  useEffect(() => {
    Promise.all([
      api.get(`/duty-templates/${id}`).then(r => setTemplate(r.data)),
      api.get('/duty-types').then(r => setDutyTypes(r.data ?? [])),
    ]).finally(() => setLoading(false))
  }, [id])

  const updateItem = (i: number, patch: Partial<TemplateItem>) => {
    setTemplate(t => t ? { ...t, items: t.items.map((item, idx) => idx === i ? { ...item, ...patch } : item) } : t)
  }

  const addItem = () => {
    setTemplate(t => t ? { ...t, items: [...t.items, newItem()] } : t)
  }

  const removeItem = (i: number) => {
    setTemplate(t => t ? { ...t, items: t.items.filter((_, idx) => idx !== i) } : t)
  }

  const handleSave = async () => {
    if (!template) return
    const invalid = template.items.filter(it => it.duty_type_id === 0)
    if (invalid.length > 0) {
      setSaveError('Bitte für alle Einträge einen Diensttyp auswählen.')
      return
    }
    setSaveError('')
    setSaving(true)
    setSaved(false)
    try {
      await api.put(`/duty-templates/${id}`, template)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch {
      setSaveError('Speichern fehlgeschlagen.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="text-brand-text-muted text-sm">Laden…</div>
  if (!template) return <div className="text-brand-danger text-sm">Vorlage nicht gefunden.</div>

  return (
    <div className="max-w-2xl">
      <div className="flex items-center gap-2 text-sm text-brand-text-muted mb-4">
        <Link to="/admin/dienstplan-vorlagen" className="hover:underline">Dienstplan-Vorlagen</Link>
        <span>/</span>
        <span className="text-brand-text">{template.name}</span>
      </div>

      <h1 className="text-2xl font-bold mb-6">{template.name}</h1>

      {/* Name + Typ + Dauer */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-5 mb-5">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Name der Vorlage</label>
            <input
              type="text"
              value={template.name}
              onChange={e => setTemplate(t => t ? { ...t, name: e.target.value } : t)}
              className={INPUT}
            />
          </div>
          <div className={`grid gap-4 ${template.template_type === 'generisch' ? 'grid-cols-2' : 'grid-cols-1'}`}>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Typ</label>
              <select
                value={template.template_type}
                onChange={e => setTemplate(t => t ? { ...t, template_type: e.target.value as Template['template_type'] } : t)}
                className={INPUT}
              >
                <option value="heim">Heim</option>
                <option value="auswärts">Auswärts</option>
                <option value="generisch">Generisch</option>
              </select>
            </div>
            {template.template_type === 'generisch' && (
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Dauer</label>
                <DurationInput
                  value={template.duration_minutes}
                  onChange={v => setTemplate(t => t ? { ...t, duration_minutes: v } : t)}
                  className={INPUT}
                />
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Items */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-5">
        <div className="flex items-center justify-between px-5 py-3 border-b border-brand-border-subtle">
          <h2 className="font-semibold text-brand-text">Dienst-Einträge</h2>
          <button
            onClick={addItem}
            className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            + Eintrag hinzufügen
          </button>
        </div>

        {template.items.length === 0 ? (
          <p className="text-sm text-brand-text-subtle text-center py-8 italic">
            Keine Einträge — klicke auf „+ Eintrag hinzufügen"
          </p>
        ) : (
          <>
            {/* Mobile */}
            <div className="sm:hidden space-y-3 p-3">
              {template.items.map((item, i) => {
                const dutyType = dutyTypes.find(d => d.id === item.duty_type_id)
                return (
                  <div key={i} className="bg-white border border-brand-border-subtle rounded p-4">
                    <div className="flex items-start justify-between gap-2 mb-3">
                      <h3 className="font-medium text-sm flex-1 text-brand-text">{dutyType?.name || 'Diensttyp auswählen'}</h3>
                      <ActionMenu
                        actions={[{ label: 'Löschen', onClick: () => removeItem(i), variant: 'danger' }]}
                      />
                    </div>
                    <div className="space-y-3">
                      <div>
                        <label className="block text-xs text-brand-text-muted mb-1">Diensttyp</label>
                        <select
                          value={item.duty_type_id}
                          onChange={e => {
                            const dtId = Number(e.target.value)
                            const dt = dutyTypes.find(d => d.id === dtId)
                            updateItem(i, { duty_type_id: dtId, ...(dt ? { anchor: dt.default_anchor, offset_minutes: dt.default_offset_minutes, audiences: dt.audiences ?? [] } : {}) })
                          }}
                          className={INPUT_SM}
                        >
                          <option value={0}>Auswählen…</option>
                          {dutyTypes.map(dt => <option key={dt.id} value={dt.id}>{dt.name}</option>)}
                        </select>
                      </div>
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <label className="block text-xs text-brand-text-muted mb-1">Anker</label>
                          <select
                            value={item.anchor}
                            onChange={e => updateItem(i, { anchor: e.target.value as 'start' | 'end' })}
                            className={INPUT_SM}
                          >
                            <option value="start">Anpfiff/Beginn</option>
                            <option value="end">Abpfiff/Ende</option>
                          </select>
                        </div>
                        <div>
                          <label className="block text-xs text-brand-text-muted mb-1">Versatz</label>
                          <OffsetInput
                            value={item.offset_minutes}
                            onChange={v => updateItem(i, { offset_minutes: v })}
                            className={INPUT_SM}
                          />
                        </div>
                      </div>
                      <div>
                        <label className="block text-xs text-brand-text-muted mb-1">Personen</label>
                        <input
                          type="number"
                          min={1}
                          value={item.slots_count}
                          onChange={e => updateItem(i, { slots_count: Number(e.target.value) })}
                          className={INPUT_SM}
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-brand-text-muted mb-1">Zielgruppe <span className="text-brand-text-subtle">(leer = keine)</span></label>
                        <div className="grid grid-cols-2 gap-1">
                          {AUDIENCE_OPTIONS.map(o => (
                            <label key={o.value} className="flex items-center gap-1.5 text-xs cursor-pointer">
                              <input
                                type="checkbox"
                                checked={(item.audiences ?? []).includes(o.value)}
                                onChange={e => updateItem(i, {
                                  audiences: e.target.checked
                                    ? [...(item.audiences ?? []), o.value]
                                    : (item.audiences ?? []).filter(a => a !== o.value),
                                })}
                                className="accent-brand-yellow"
                              />
                              {o.label}
                            </label>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Desktop */}
            <div className="hidden sm:block divide-y divide-brand-border-subtle">
              {template.items.map((item, i) => (
                <div key={i} className="p-4 grid grid-cols-[1fr_auto_auto_auto_auto_auto] gap-3 items-center">
                  <div>
                    <label className="block text-xs text-brand-text-muted mb-1">Diensttyp</label>
                    <select
                      value={item.duty_type_id}
                      onChange={e => {
                        const dtId = Number(e.target.value)
                        const dt = dutyTypes.find(d => d.id === dtId)
                        updateItem(i, { duty_type_id: dtId, ...(dt ? { anchor: dt.default_anchor, offset_minutes: dt.default_offset_minutes, audiences: dt.audiences ?? [] } : {}) })
                      }}
                      className="w-full border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow"
                    >
                      <option value={0}>Auswählen…</option>
                      {dutyTypes.map(dt => <option key={dt.id} value={dt.id}>{dt.name}</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs text-brand-text-muted mb-1">Anker</label>
                    <select
                      value={item.anchor}
                      onChange={e => updateItem(i, { anchor: e.target.value as 'start' | 'end' })}
                      className="border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow"
                    >
                      <option value="start">Anpfiff</option>
                      <option value="end">Spielende</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs text-brand-text-muted mb-1">Versatz</label>
                    <OffsetInput
                      value={item.offset_minutes}
                      onChange={v => updateItem(i, { offset_minutes: v })}
                      className="w-28 border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-brand-text-muted mb-1">Personen</label>
                    <input
                      type="number"
                      min={1}
                      value={item.slots_count}
                      onChange={e => updateItem(i, { slots_count: Number(e.target.value) })}
                      className="w-16 border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-brand-text-muted mb-1">Zielgruppe <span className="text-brand-text-subtle text-xs">(leer = keine)</span></label>
                    <div className="space-y-1">
                      {AUDIENCE_OPTIONS.map(o => (
                        <label key={o.value} className="flex items-center gap-1.5 text-xs cursor-pointer whitespace-nowrap">
                          <input
                            type="checkbox"
                            checked={(item.audiences ?? []).includes(o.value)}
                            onChange={e => updateItem(i, {
                              audiences: e.target.checked
                                ? [...(item.audiences ?? []), o.value]
                                : (item.audiences ?? []).filter(a => a !== o.value),
                            })}
                            className="accent-brand-yellow"
                          />
                          {o.label}
                        </label>
                      ))}
                    </div>
                  </div>
                  <button
                    onClick={() => removeItem(i)}
                    aria-label="Eintrag entfernen"
                    className="text-brand-text-muted hover:text-brand-danger transition-colors mt-4 p-1"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-xs text-brand-text mb-5">
        <strong>Versatz:</strong> Format <code>-1h 30min</code> (vor Anker) · <code>+15min</code> (nach Anker) · <code>0</code>. Tage, Stunden und Minuten können kombiniert werden, z.B. <code>-2d 3h</code>.
      </div>

      {saveError && (
        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-3">
          {saveError}
        </p>
      )}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={saving}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {saving ? 'Speichern…' : 'Vorlage speichern'}
        </button>
        {saved && (
          <span className="text-brand-success text-sm flex items-center gap-1">
            <Check className="w-4 h-4" /> Gespeichert
          </span>
        )}
      </div>
    </div>
  )
}
