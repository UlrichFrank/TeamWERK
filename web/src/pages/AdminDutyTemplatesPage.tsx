import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import MobileCard from '../components/MobileCard'
import EditModal from '../components/EditModal'

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

interface DutyTemplate {
  id: number
  name: string
  template_type: 'heim' | 'auswärts' | 'generisch'
  game_duration_minutes: number
  item_count: number
}

interface TemplateFormState {
  name: string
  template_type: 'heim' | 'auswärts' | 'generisch'
  game_duration_minutes: number
  items: TemplateItem[]
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

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const INPUT_SM = 'w-full border border-brand-border rounded px-2 py-1 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow'

function newTemplate(): TemplateFormState {
  return {
    name: '',
    template_type: 'heim',
    game_duration_minutes: 60,
    items: [],
  }
}

const ROLE_OPTIONS = [
  { value: 'elternteil', label: 'Elternteil' },
  { value: 'spieler', label: 'Spieler' },
  { value: 'trainer', label: 'Trainer' },
  { value: 'vorstand', label: 'Vorstand' },
  { value: 'admin', label: 'Admin' },
]

function newItem(): TemplateItem {
  return { duty_type_id: 0, anchor: 'start', offset_minutes: 0, slots_count: 1, role_desc: 'spieler' }
}

function TemplateForm({ template, onChange, dutyTypes }: {
  template: TemplateFormState
  onChange: (template: TemplateFormState) => void
  dutyTypes: DutyType[]
}) {
  const updateItem = (index: number, patch: Partial<TemplateItem>) => {
    onChange({
      ...template,
      items: template.items.map((item, idx) => idx === index ? { ...item, ...patch } : item),
    })
  }

  const addItem = () => onChange({ ...template, items: [...template.items, newItem()] })
  const removeItem = (index: number) => onChange({ ...template, items: template.items.filter((_, idx) => idx !== index) })

  return (
    <div className="space-y-4">
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Name der Vorlage</label>
          <input
            value={template.name}
            onChange={e => onChange({ ...template, name: e.target.value })}
            placeholder="z.B. Heimspiel Standard"
            className={INPUT}
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Typ</label>
            <select
              value={template.template_type}
              onChange={e => onChange({ ...template, template_type: e.target.value as TemplateFormState['template_type'] })}
              className={INPUT}
            >
              <option value="heim">Heim</option>
              <option value="auswärts">Auswärts</option>
              <option value="generisch">Generisch</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Spieldauer (min)</label>
            <input
              type="number"
              min={1}
              value={template.game_duration_minutes}
              onChange={e => onChange({ ...template, game_duration_minutes: Number(e.target.value) })}
              className={INPUT}
            />
          </div>
        </div>
      </div>

      <div className="bg-brand-surface-card rounded-xl border border-brand-border-subtle p-4">
        <div className="flex items-center justify-between mb-4 gap-3">
          <h3 className="font-semibold text-brand-text">Dienst-Einträge</h3>
          <button
            type="button"
            onClick={addItem}
            className="bg-brand-yellow text-brand-black rounded-md px-3 py-1.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            + Eintrag hinzufügen
          </button>
        </div>

        {template.items.length === 0 ? (
          <p className="text-sm text-brand-text-subtle italic">Keine Einträge — klicke auf „+ Eintrag hinzufügen“.</p>
        ) : (
          <div className="space-y-4">
            {template.items.map((item, index) => (
              <div key={index} className="border border-brand-border-subtle rounded-xl p-3">
                <div className="flex items-center justify-between gap-2 mb-3">
                  <div className="text-sm font-medium text-brand-text">Eintrag {index + 1}</div>
                  <button
                    type="button"
                    onClick={() => removeItem(index)}
                    className="text-xs text-brand-danger hover:text-brand-danger/80"
                  >
                    Entfernen
                  </button>
                </div>
                <div className="space-y-2">
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <div>
                      <label className="block text-xs text-brand-text-muted mb-1">Diensttyp</label>
                      <select
                        value={item.duty_type_id}
                        onChange={e => {
                          const dutyTypeId = Number(e.target.value)
                          const dutyType = dutyTypes.find(dt => dt.id === dutyTypeId)
                          updateItem(index, {
                            duty_type_id: dutyTypeId,
                            anchor: dutyType?.default_anchor ?? item.anchor,
                            offset_minutes: dutyType?.default_offset_minutes ?? item.offset_minutes,
                          })
                        }}
                        className={INPUT_SM}
                      >
                        <option value={0}>Auswählen…</option>
                        {dutyTypes.map(dt => (
                          <option key={dt.id} value={dt.id}>{dt.name}</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs text-brand-text-muted mb-1">Rolle</label>
                      <select
                        value={item.role_desc}
                        onChange={e => updateItem(index, { role_desc: e.target.value })}
                        className={INPUT_SM}
                      >
                        {ROLE_OPTIONS.map(role => (
                          <option key={role.value} value={role.value}>{role.label}</option>
                        ))}
                        {item.role_desc && !ROLE_OPTIONS.some(role => role.value === item.role_desc) && (
                          <option value={item.role_desc}>{item.role_desc}</option>
                        )}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs text-brand-text-muted mb-1">Personen</label>
                      <input
                        type="number"
                        min={1}
                        value={item.slots_count}
                        onChange={e => updateItem(index, { slots_count: Number(e.target.value) })}
                        className={INPUT_SM}
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs text-brand-text-muted mb-1">Anker</label>
                      <select
                        value={item.anchor}
                        onChange={e => updateItem(index, { anchor: e.target.value as TemplateItem['anchor'] })}
                        className={INPUT_SM}
                      >
                        <option value="start">Anpfiff</option>
                        <option value="end">Spielende</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs text-brand-text-muted mb-1">Versatz (min)</label>
                      <input
                        type="number"
                        value={item.offset_minutes}
                        onChange={e => updateItem(index, { offset_minutes: Number(e.target.value) })}
                        className={INPUT_SM}
                      />
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <p className="text-xs text-brand-text-subtle">
        Negative Werte = vor dem Anker (z.B. −60 = 60 min vor Anpfiff). Positive Werte = nach dem Anker.
      </p>
    </div>
  )
}

export default function AdminDutyTemplatesPage() {
  const [templates, setTemplates] = useState<DutyTemplate[]>([])
  const [dutyTypes, setDutyTypes] = useState<DutyType[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [deleteError, setDeleteError] = useState('')
  const [modalTemplate, setModalTemplate] = useState<TemplateFormState | null>(null)
  const [editingTemplateId, setEditingTemplateId] = useState<number | null>(null)
  const [modalSaving, setModalSaving] = useState(false)
  const [modalError, setModalError] = useState('')

  const loadTemplates = async () => {
    const r = await api.get('/admin/duty-templates')
    setTemplates(r.data ?? [])
  }

  useEffect(() => {
    Promise.all([
      api.get('/admin/duty-templates').then(r => setTemplates(r.data ?? [])),
      api.get('/admin/duty-types').then(r => setDutyTypes(r.data ?? [])),
    ]).finally(() => setLoading(false))
  }, [])

  const typeCounts = templates.reduce<Record<string, number>>((acc, t) => {
    acc[t.template_type] = (acc[t.template_type] || 0) + 1
    return acc
  }, {})

  const openCreateModal = () => {
    setModalTemplate(newTemplate())
    setEditingTemplateId(null)
    setModalError('')
  }

  const openEditModal = async (id: number) => {
    setModalError('')
    setEditingTemplateId(id)
    setModalTemplate(null)
    try {
      const r = await api.get(`/admin/duty-templates/${id}`)
      setModalTemplate({
        name: r.data.name,
        template_type: r.data.template_type,
        game_duration_minutes: r.data.game_duration_minutes,
        items: r.data.items,
      })
    } catch {
      setModalError('Vorlage konnte nicht geladen werden.')
    }
  }

  const closeModal = () => {
    setModalTemplate(null)
    setEditingTemplateId(null)
    setModalError('')
  }

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

  const handleSave = async () => {
    if (!modalTemplate) return
    if (!modalTemplate.name.trim()) {
      setModalError('Name darf nicht leer sein.')
      return
    }
    if (modalTemplate.items.some(item => item.duty_type_id === 0)) {
      setModalError('Bitte wähle für alle Einträge einen Diensttyp.')
      return
    }

    setModalError('')
    setModalSaving(true)

    try {
      if (editingTemplateId == null) {
        const createResponse = await api.post('/admin/duty-templates', {
          name: modalTemplate.name.trim(),
          template_type: modalTemplate.template_type,
          game_duration_minutes: modalTemplate.game_duration_minutes,
        })
        const createdId = createResponse.data.id
        if (modalTemplate.items.length > 0) {
          await api.put(`/admin/duty-templates/${createdId}`, {
            name: modalTemplate.name.trim(),
            template_type: modalTemplate.template_type,
            game_duration_minutes: modalTemplate.game_duration_minutes,
            items: modalTemplate.items,
          })
        }
      } else {
        await api.put(`/admin/duty-templates/${editingTemplateId}`, {
          name: modalTemplate.name.trim(),
          template_type: modalTemplate.template_type,
          game_duration_minutes: modalTemplate.game_duration_minutes,
          items: modalTemplate.items,
        })
      }
      await loadTemplates()
      closeModal()
    } catch {
      setModalError(editingTemplateId == null ? 'Erstellen fehlgeschlagen.' : 'Speichern fehlgeschlagen.')
    } finally {
      setModalSaving(false)
    }
  }

  if (loading) return <div className="text-brand-text-muted text-sm">Laden…</div>

  return (
    <div className="max-w-4xl">
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Dienstplan-Vorlagen</h1>
          <button
            onClick={openCreateModal}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            + Neue Vorlage
          </button>
        </div>
      </div>

      {Object.entries(typeCounts).some(([, count]) => count > 1) && (
        <div className="mb-4 p-3 bg-brand-warning-light border border-brand-warning/40 rounded-lg text-sm text-brand-text flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 text-brand-warning mt-0.5 flex-shrink-0" />
          Achtung: Es gibt mehrere Vorlagen des gleichen Typs. Bei der Slot-Generierung wird immer die erste verwendet (niedrigste ID).
        </div>
      )}

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-6">
        {templates.length === 0 ? (
          <p className="text-sm text-brand-text-subtle text-center py-10 italic">
            Keine Vorlagen vorhanden — lege eine neue an.
          </p>
        ) : (
          <>
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
                        <div className="flex gap-1 justify-end">
                          <button
                            onClick={() => openEditModal(t.id)}
                            className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                          >
                            Bearbeiten
                          </button>
                          <button
                            onClick={() => setDeleteId(t.id)}
                            className="bg-brand-danger text-white rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-danger/90 transition-colors"
                          >
                            Löschen
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            <div className="sm:hidden divide-y divide-brand-border-subtle">
              {templates.map(t => (
                <MobileCard
                  key={t.id}
                  title={t.name}
                  subtitle={`${typeLabel[t.template_type] ?? t.template_type} · ${t.item_count} Einträge · ${t.game_duration_minutes} min`}
                  badge={{ label: typeLabel[t.template_type] ?? t.template_type, variant: t.template_type === 'heim' ? 'blue' : t.template_type === 'auswärts' ? 'yellow' : 'red' }}
                  actions={[
                    { label: 'Bearbeiten', onClick: () => openEditModal(t.id) },
                    { label: 'Löschen', onClick: () => setDeleteId(t.id), variant: 'danger' },
                  ]}
                >
                  {typeCounts[t.template_type] > 1 && (
                    <div className="mt-2 text-xs text-brand-warning flex items-center gap-1">
                      <AlertTriangle className="w-3.5 h-3.5" />
                      Mehrere Vorlagen dieses Typs vorhanden
                    </div>
                  )}
                </MobileCard>
              ))}
              {deleteId !== null && (
                <div className="p-4 flex items-center gap-2 text-xs text-brand-text-muted">
                  <span>Wirklich löschen?</span>
                  <button onClick={() => handleDelete(deleteId)} className="text-brand-danger font-medium">Ja</button>
                  <button onClick={() => setDeleteId(null)} className="text-brand-text-muted">Abbrechen</button>
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {deleteError && (
        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4">
          {deleteError}
        </p>
      )}

      <EditModal
        isOpen={modalTemplate !== null}
        title={editingTemplateId == null ? 'Neue Dienstplan-Vorlage' : 'Dienstplan-Vorlage bearbeiten'}
        onClose={closeModal}
        onSave={handleSave}
        isSaving={modalSaving}
        maxWidthClass="max-w-3xl"
      >
        {modalTemplate ? (
          <>
            <TemplateForm template={modalTemplate} onChange={setModalTemplate} dutyTypes={dutyTypes} />
            {modalError && (
              <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                {modalError}
              </p>
            )}
          </>
        ) : (
          <div className="text-brand-text-muted text-sm">Lade Vorlage…</div>
        )}
      </EditModal>
    </div>
  )
}
