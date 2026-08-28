import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import ActionMenu from '../components/ActionMenu'
import EditModal from '../components/EditModal'
import OffsetInput from '../components/OffsetInput'
import DurationInput from '../components/DurationInput'
import { AUDIENCE_OPTIONS } from '../lib/constants'
import { buildTeamShortNames, type TeamForName } from '../lib/teamName'
import { TeamScopeField, RotationEnabledField, AusrichterField, SlotsCountField } from '../components/DutyTemplateItemFields'
import HoursInput from '../components/HoursInput'
import { errorData } from '../lib/errors'
import { toggleTeamID, refreshItemsFromDutyTypes } from '../lib/dutyTemplateItems'
import { dynamicSpanImpossible, IMPOSSIBLE_SPAN_MESSAGE } from '../lib/duration'
import { ChevronDown, RefreshCw } from 'lucide-react'

interface DutyType {
  id: number
  name: string
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
  /** Dauer in Stunden — Vorbelegung der Vorlagen-Zeile (dienst-dauer). */
  hours_value: number
  /** Dauer-Modus des Diensttyps (dienst-dauer-dynamisch) — Vorbelegung beim Copy-on-pick. */
  duration_mode?: 'absolut' | 'dynamisch'
  end_anchor?: 'start' | 'end'
  end_offset_minutes?: number
  audiences?: string[] | null
}

interface Ausrichter {
  id: number
  name: string
  aktiv: boolean
  is_default: boolean
  sort_order: number
}

interface TemplateItem {
  duty_type_id: number
  anchor: 'start' | 'end'
  offset_minutes: number
  slots_count: number
  /** Dauer des Dienstes in Stunden (dienst-dauer). Copy-on-pick vom Diensttyp. Gilt nur
   * im Modus 'absolut'; im Modus 'dynamisch' wird sie nicht gelesen, aber weiter
   * mitgeführt, damit ein Moduswechsel hin und zurück den Wert nicht verliert
   * (dienst-zeitmodus-strikt). */
  hours_value: number
  /** Zeit-Modus: 'absolut' (Maske: „Startzeit + Dauer") = hours_value gilt fest,
   * 'dynamisch' (Maske: „Startzeit + Endzeit") = das Ende folgt
   * end_anchor/end_offset_minutes gegen den Termin. */
  duration_mode: 'absolut' | 'dynamisch'
  end_anchor: 'start' | 'end'
  end_offset_minutes: number
  audiences: string[]
  /** Leer/fehlend = Eintrag gilt für ALLE Kaderteams eines Spiels. */
  team_ids?: number[] | null
  /** Bewirtungsrotation: false/fehlend = deaktiviert. Siehe DutyTemplateItemFields. */
  rotation_enabled?: boolean
  /** Ausrichter-Einschränkung (heimspieltag-ausrichter): null/fehlend = für alle Ausrichter. */
  ausrichter_id?: number | null
}

interface DutyTemplate {
  id: number
  name: string
  template_type: 'heim' | 'auswärts' | 'generisch'
  duration_minutes: number
  item_count: number
}

interface TemplateFormState {
  name: string
  template_type: 'heim' | 'auswärts' | 'generisch'
  duration_minutes: number
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
    duration_minutes: 60,
    items: [],
  }
}

/**
 * Unmögliche Zeitspanne einer Vorlagen-Zeile — dieselbe Regel wie im Server
 * (`dynamicSpanImpossible` in internal/games/handler.go) und in der
 * Diensttyp-Maske. Blockiert das Speichern, statt den 400 abzuwarten.
 * (openspec/changes/dienst-zeitmodus-strikt)
 */
function itemSpanImpossible(item: TemplateItem): boolean {
  return dynamicSpanImpossible(
    item.duration_mode, item.anchor, item.offset_minutes,
    item.end_anchor, item.end_offset_minutes,
  )
}

function newItem(): TemplateItem {
  return {
    duty_type_id: 0, anchor: 'start', offset_minutes: 0, slots_count: 1, hours_value: 1,
    duration_mode: 'absolut', end_anchor: 'end', end_offset_minutes: 0,
    audiences: [], team_ids: [],
  }
}

function TemplateForm({ template, onChange, dutyTypes, teams, ausrichter }: {
  template: TemplateFormState
  onChange: (template: TemplateFormState) => void
  dutyTypes: DutyType[]
  teams: TeamForName[]
  ausrichter: Ausrichter[]
}) {
  // Wie auf der Detailseite: generische Vorlagen erzeugen Slots ohne team_id,
  // die Regeneration ignoriert team_ids dort — die Auswahl wird deshalb gar
  // nicht erst angeboten. Gespeicherte Werte bleiben unangetastet.
  const scopeTeams = template.template_type === 'generisch' ? [] : teams
  const [showItemMenu, setShowItemMenu] = useState(false)
  const [refreshNote, setRefreshNote] = useState('')
  // Ausrichter-Auswahl nur für Heim-Vorlagen; für andere template_type wird nichts angeboten.
  const ausrichterOptions = template.template_type === 'heim' ? ausrichter : []
  const teamShortNames = buildTeamShortNames(teams)
  const updateItem = (index: number, patch: Partial<TemplateItem>) => {
    onChange({
      ...template,
      items: template.items.map((item, idx) => idx === index ? { ...item, ...patch } : item),
    })
  }

  const addItem = () => onChange({ ...template, items: [...template.items, newItem()] })

  // Auffrischen aus den Diensttypen: ändert nur den Formularzustand, persistiert
  // wird wie sonst auch erst beim Speichern — deshalb kein Bestätigungsdialog,
  // der Speichern-Knopf ist bereits das Tor. Die Meldung sagt, was passiert ist.
  const refreshFromDutyTypes = () => {
    setShowItemMenu(false)
    const { items, changed } = refreshItemsFromDutyTypes(template.items, dutyTypes)
    setRefreshNote(
      changed === 0
        ? 'Alle Einträge stimmen bereits mit ihrem Diensttyp überein.'
        : `${changed} ${changed === 1 ? 'Eintrag' : 'Einträge'} aus den Diensttypen aufgefrischt — zum Übernehmen speichern.`,
    )
    if (changed > 0) onChange({ ...template, items })
  }
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
          {template.template_type === 'generisch' && (
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Dauer</label>
              <DurationInput
                value={template.duration_minutes}
                onChange={v => onChange({ ...template, duration_minutes: v })}
                className={INPUT}
              />
            </div>
          )}
        </div>
      </div>

      <div className="bg-brand-surface-card rounded-xl border border-brand-border-subtle p-4">
        <div className="flex items-center justify-between mb-4 gap-3">
          <h3 className="font-semibold text-brand-text">Dienst-Einträge</h3>
          <div className="relative shrink-0">
            <div className="flex">
              <button
                type="button"
                onClick={addItem}
                className="bg-brand-yellow text-brand-black rounded-l-md px-3 py-1.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
              >
                + Eintrag hinzufügen
              </button>
              <button
                type="button"
                onClick={() => setShowItemMenu(v => !v)}
                aria-label="Weitere Aktionen"
                aria-expanded={showItemMenu}
                aria-haspopup="menu"
                className="flex items-center bg-brand-yellow text-brand-black rounded-r-md border-l border-l-brand-black/20 px-2 py-1.5 hover:bg-brand-black hover:text-brand-yellow transition-colors"
              >
                <ChevronDown className="w-3.5 h-3.5" />
              </button>
            </div>
            {showItemMenu && (
              <div role="menu" className="absolute right-0 mt-1 w-72 bg-brand-white border border-brand-border rounded-md shadow-lg z-20 overflow-hidden">
                <button
                  type="button"
                  role="menuitem"
                  onClick={refreshFromDutyTypes}
                  className="w-full flex items-start gap-2 text-left px-4 py-2.5 text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
                >
                  <RefreshCw className="w-4 h-4 shrink-0 mt-0.5" />
                  <span>
                    Aus Diensttypen auffrischen
                    <span className="block text-xs text-brand-text-muted">
                      Holt Dauer, Anker, Versatz und Zielgruppe zurück in alle Einträge.
                    </span>
                  </span>
                </button>
              </div>
            )}
          </div>
        </div>

        {refreshNote && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            {refreshNote}
          </div>
        )}

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
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
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
                            hours_value: dutyType?.hours_value ?? item.hours_value,
                            duration_mode: dutyType?.duration_mode ?? item.duration_mode,
                            end_anchor: dutyType?.end_anchor ?? item.end_anchor,
                            end_offset_minutes: dutyType?.end_offset_minutes ?? item.end_offset_minutes,
                            audiences: dutyType?.audiences ?? [],
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
                    <SlotsCountField
                      id={`slots-count-${index}`}
                      value={item.slots_count}
                      onChange={v => updateItem(index, { slots_count: v })}
                      rotationEnabled={item.rotation_enabled}
                      inputClassName={INPUT_SM}
                    />
                  </div>

                  <div>
                    <label className="block text-xs text-brand-text-muted mb-1">Zielgruppe <span className="text-brand-text-subtle font-normal">(leer = keine Einschränkung)</span></label>
                    <div className="grid grid-cols-2 sm:grid-cols-3 gap-1.5 mt-1">
                      {AUDIENCE_OPTIONS.map(o => (
                        <label key={o.value} className="flex items-center gap-1.5 text-xs cursor-pointer">
                          <input
                            type="checkbox"
                            checked={item.audiences.includes(o.value)}
                            onChange={e => updateItem(index, {
                              audiences: e.target.checked
                                ? [...item.audiences, o.value]
                                : item.audiences.filter(a => a !== o.value),
                            })}
                            className="accent-brand-yellow"
                          />
                          {o.label}
                        </label>
                      ))}
                    </div>
                  </div>

                  <TeamScopeField
                    teams={scopeTeams}
                    shortNames={teamShortNames}
                    selected={item.team_ids ?? []}
                    onToggle={(teamID, checked) => updateItem(index, {
                      team_ids: toggleTeamID(item.team_ids, teamID, checked),
                    })}
                  />
                  <RotationEnabledField
                    id={`rotation-enabled-modal-${index}`}
                    value={item.rotation_enabled}
                    onChange={v => updateItem(index, { rotation_enabled: v })}
                  />
                  {ausrichterOptions.length > 0 && (
                    <AusrichterField
                      id={`ausrichter-modal-${index}`}
                      value={item.ausrichter_id}
                      options={ausrichterOptions}
                      onChange={v => updateItem(index, { ausrichter_id: v })}
                    />
                  )}

                  <div>
                    <label className="block text-xs text-brand-text-muted mb-1">Zeit-Modus</label>
                    <div className="flex flex-col gap-1.5 sm:flex-row sm:gap-4">
                      <label className="flex items-center gap-1.5 text-xs cursor-pointer">
                        <input
                          type="radio"
                          name={`duration-mode-${index}`}
                          checked={item.duration_mode === 'absolut'}
                          onChange={() => updateItem(index, { duration_mode: 'absolut' })}
                          className="accent-brand-yellow"
                        />
                        Startzeit + Dauer
                      </label>
                      <label className="flex items-center gap-1.5 text-xs cursor-pointer">
                        <input
                          type="radio"
                          name={`duration-mode-${index}`}
                          checked={item.duration_mode === 'dynamisch'}
                          onChange={() => updateItem(index, { duration_mode: 'dynamisch' })}
                          className="accent-brand-yellow"
                        />
                        Startzeit + Endzeit
                      </label>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs text-brand-text-muted mb-1">Start-Anker</label>
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
                      <label className="block text-xs text-brand-text-muted mb-1">Start-Versatz</label>
                      <OffsetInput
                        value={item.offset_minutes}
                        onChange={v => updateItem(index, { offset_minutes: v })}
                        className={INPUT_SM}
                      />
                    </div>
                  </div>

                  {item.duration_mode === 'absolut' ? (
                    <div className="sm:w-1/2 sm:pr-1.5">
                      <label className="block text-xs text-brand-text-muted mb-1">Dauer</label>
                      <HoursInput
                        value={item.hours_value}
                        onChange={v => updateItem(index, { hours_value: v })}
                        className={INPUT_SM}
                      />
                    </div>
                  ) : (
                    <>
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div>
                          <label className="block text-xs text-brand-text-muted mb-1">End-Anker</label>
                          <select
                            value={item.end_anchor}
                            onChange={e => updateItem(index, { end_anchor: e.target.value as TemplateItem['end_anchor'] })}
                            className={INPUT_SM}
                          >
                            <option value="start">Anpfiff</option>
                            <option value="end">Spielende</option>
                          </select>
                        </div>
                        <div>
                          <label className="block text-xs text-brand-text-muted mb-1">End-Versatz</label>
                          <OffsetInput
                            value={item.end_offset_minutes}
                            onChange={v => updateItem(index, { end_offset_minutes: v })}
                            className={INPUT_SM}
                          />
                        </div>
                      </div>
                      {itemSpanImpossible(item) && (
                        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                          {IMPOSSIBLE_SPAN_MESSAGE}
                        </p>
                      )}
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <p className="text-xs text-brand-text-subtle">
        Versatz-Format: <code>-1h 30min</code> (vor Anker) · <code>+30min</code> (nach Anker) · <code>0</code>
      </p>
    </div>
  )
}

export default function AdminDutyTemplatesPage() {
  const [templates, setTemplates] = useState<DutyTemplate[]>([])
  const [dutyTypes, setDutyTypes] = useState<DutyType[]>([])
  const [teams, setTeams] = useState<TeamForName[]>([])
  const [ausrichter, setAusrichter] = useState<Ausrichter[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteError, setDeleteError] = useState('')
  const [modalTemplate, setModalTemplate] = useState<TemplateFormState | null>(null)
  const [editingTemplateId, setEditingTemplateId] = useState<number | null>(null)
  const [modalSaving, setModalSaving] = useState(false)
  const [modalError, setModalError] = useState('')

  const loadTemplates = async () => {
    const r = await api.get('/duty-templates')
    setTemplates(r.data ?? [])
  }

  useEffect(() => {
    Promise.all([
      api.get('/duty-templates').then(r => setTemplates(r.data ?? [])),
      api.get('/duty-types').then(r => setDutyTypes(r.data ?? [])),
      // Kaderteams der aktiven Saison für die Team-Einschränkung pro Eintrag.
      api.get<TeamForName[]>('/teams/names').then(r => setTeams(r.data ?? [])),
      // Ausrichter für die Ausrichter-Auswahl pro Eintrag (nur aktive).
      api.get<{ items: Ausrichter[] }>('/ausrichter').then(r => setAusrichter((r.data?.items ?? []).filter(a => a.aktiv))),
    ]).finally(() => setLoading(false))
  }, [])

  useLiveUpdates(event => { if (event === 'games') loadTemplates() })


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
      const r = await api.get(`/duty-templates/${id}`)
      setModalTemplate({
        name: r.data.name,
        template_type: r.data.template_type,
        duration_minutes: r.data.duration_minutes,
        items: (r.data.items ?? []).map((it: Omit<TemplateItem, 'audiences' | 'duration_mode' | 'end_anchor' | 'end_offset_minutes'> & {
          audiences?: string[] | null
          duration_mode?: 'absolut' | 'dynamisch'
          end_anchor?: 'start' | 'end'
          end_offset_minutes?: number
        }) => ({
          ...it,
          audiences: it.audiences ?? [],
          duration_mode: it.duration_mode ?? 'absolut',
          end_anchor: it.end_anchor ?? 'end',
          end_offset_minutes: it.end_offset_minutes ?? 0,
        })),
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

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Vorlage „${name}" wirklich löschen?`)) return
    setDeleteError('')
    try {
      await api.delete(`/duty-templates/${id}`)
      setTemplates(prev => prev.filter(t => t.id !== id))
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
    // Unmögliche Spanne: der Server antwortet hier mit 400 und verwürfe die ganze
    // Vorlage. Die Nummer im Text ist die Zeile, an der die Meldung schon steht —
    // der Modal-Fehler oben soll nur sagen, wo man hinschauen muss.
    const badSpan = modalTemplate.items.findIndex(itemSpanImpossible)
    if (badSpan >= 0) {
      setModalError(`Eintrag ${badSpan + 1}: ${IMPOSSIBLE_SPAN_MESSAGE}`)
      return
    }

    setModalError('')
    setModalSaving(true)

    try {
      if (editingTemplateId == null) {
        const createResponse = await api.post('/duty-templates', {
          name: modalTemplate.name.trim(),
          template_type: modalTemplate.template_type,
          duration_minutes: modalTemplate.duration_minutes,
        })
        const createdId = createResponse.data.id
        if (modalTemplate.items.length > 0) {
          await api.put(`/duty-templates/${createdId}`, {
            name: modalTemplate.name.trim(),
            template_type: modalTemplate.template_type,
            duration_minutes: modalTemplate.duration_minutes,
            items: modalTemplate.items,
          })
        }
      } else {
        await api.put(`/duty-templates/${editingTemplateId}`, {
          name: modalTemplate.name.trim(),
          template_type: modalTemplate.template_type,
          duration_minutes: modalTemplate.duration_minutes,
          items: modalTemplate.items,
        })
      }
      await loadTemplates()
      closeModal()
    } catch (e) {
      // Die Backend-Fehlercodes von PUT /duty-templates in lesbare Sätze
      // übersetzen. Ohne diese Zuordnung sagt das Modal bei einer abgelehnten
      // Rotations- oder Ausrichter-Kombination nur "Speichern fehlgeschlagen"
      // und der Vorstand rät, welche Zeile gemeint ist.
      const code = errorData<{ error?: string }>(e)?.error
      const fallback = editingTemplateId == null ? 'Erstellen fehlgeschlagen.' : 'Speichern fehlgeschlagen.'
      setModalError(
        code === 'rotation_requires_normal_behavior'
          ? 'Bewirtungsrotation erfordert „Normal (immer)" bei „Mehrere Spiele am gleichen Tag" und „Spiele am Vortag / Folgetag" des zugehörigen Diensttyps.'
          : code === 'ausrichter_requires_heim_template'
            ? 'Ausrichter-Auswahl ist nur bei Heim-Vorlagen erlaubt.'
            : code === 'invalid_hours_value'
              ? 'Die Dauer eines Eintrags muss größer als 0 sein.'
              : code === 'impossible_duration_span'
                ? IMPOSSIBLE_SPAN_MESSAGE
                : code === 'invalid_team'
                  ? 'Ein ausgewähltes Team existiert nicht mehr.'
                  : fallback,
      )
    } finally {
      setModalSaving(false)
    }
  }

  if (loading) return <div className="text-brand-text-muted text-sm">Laden…</div>

  return (
    <div className="max-w-4xl">
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Dienstplan-Vorlagen</h1>
          <button
            onClick={openCreateModal}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-1.5 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            + Vorlage
          </button>
        </div>
      </div>

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-6">
        {templates.length === 0 ? (
          <p className="text-sm text-brand-text-subtle text-center py-10 italic">
            Keine Vorlagen vorhanden — lege eine neue an.
          </p>
        ) : (
          <>
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Typ</th>
                  <th className="hidden sm:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Dauer</th>
                  <th className="hidden sm:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Einträge</th>
                  <th className="bg-brand-surface-card px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-subtle">
                {templates.map(t => (
                  <tr key={t.id} className="hover:bg-brand-table-select transition-colors">
                    <td className="px-4 py-3">
                      <button
                        type="button"
                        onClick={() => openEditModal(t.id)}
                        className="font-medium text-brand-text hover:text-brand-blue hover:underline text-left"
                      >
                        {t.name}
                      </button>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${typeBadge[t.template_type] ?? ''}`}>
                        {typeLabel[t.template_type] ?? t.template_type}
                      </span>
                    </td>
                    <td className="hidden sm:table-cell px-4 py-3 text-brand-text-muted">
                      {t.template_type === 'generisch' ? `${t.duration_minutes} min` : '–'}
                    </td>
                    <td className="hidden sm:table-cell px-4 py-3 text-brand-text-muted">{t.item_count}</td>
                    <td className="px-3 py-3 text-right">
                      <ActionMenu actions={[
                        { label: 'Bearbeiten', onClick: () => openEditModal(t.id) },
                        { label: 'Löschen', onClick: () => handleDelete(t.id, t.name), variant: 'danger' },
                      ]} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

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
            <TemplateForm template={modalTemplate} onChange={setModalTemplate} dutyTypes={dutyTypes} teams={teams} ausrichter={ausrichter} />
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
