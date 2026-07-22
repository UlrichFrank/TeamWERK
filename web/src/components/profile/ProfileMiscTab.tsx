import { useState, useEffect, FormEvent } from 'react'
import { api } from '../../lib/api'
import Toggle from '../Toggle'
import { useAuth, MapsProvider } from '../../contexts/AuthContext'

type Category = 'games' | 'trainings' | 'duties' | 'carpooling' | 'chat' | 'operativ' | 'sonstiges'

interface Pref {
  push: boolean
  email: boolean
}

type Prefs = Record<Category, Pref>

const defaults: Prefs = {
  games: { push: true, email: false },
  trainings: { push: true, email: false },
  duties: { push: true, email: false },
  carpooling: { push: true, email: false },
  chat: { push: true, email: false },
  operativ: { push: true, email: false },
  sonstiges: { push: true, email: false },
}

const categoryLabels: Record<Category, string> = {
  games: 'Spiele',
  trainings: 'Trainings',
  duties: 'Dienste',
  carpooling: 'Fahrgemeinschaften',
  chat: 'Nachrichten',
  operativ: 'Vereinsaufgaben',
  sonstiges: 'Sonstige Events',
}

// Kurzbeschreibung unter dem Label (nur wo hilfreich).
const categoryDescriptions: Record<Category, string> = {
  games: 'Erinnerungen 24 h und 3 h vor Spielen, Turnieren und sonstigen Terminen deines Teams.',
  trainings: 'Erinnerungen 24 h und 3 h vor Trainingseinheiten deines Teams.',
  duties: 'Neue offene Dienste in der Dienstbörse sowie Benachrichtigung bei einer direkten Dienstzuweisung.',
  carpooling: 'Anfragen und Zusagen in Fahrgemeinschaften zu Spielen und Terminen.',
  chat: 'Neue Nachrichten in Mannschafts- und Direktchats.',
  operativ: 'Erinnerungen an deine Vereinsaufgaben, z. B. Anwesenheiten nachtragen oder Spielberichte freigeben.',
  sonstiges: 'Technische Ereignisse rund um Vereinsinhalte, z. B. wenn ein hochgeladenes Video fertig verarbeitet ist oder ein Video bald gelöscht wird.',
}

export default function ProfileMiscTab() {
  const { user, mapsProvider: ctxMapsProvider, setMapsProvider: setCtxMapsProvider } = useAuth()
  const isSpieler = user?.clubFunctions?.includes('spieler') === true
  const [prefs, setPrefs] = useState<Prefs>(defaults)
  const [changed, setChanged] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [absencesPublic, setAbsencesPublic] = useState(false)
  const [absenceSaving, setAbsenceSaving] = useState(false)
  const [mapsProvider, setMapsProvider] = useState<MapsProvider>(ctxMapsProvider)
  const [mapsSaving, setMapsSaving] = useState(false)
  const [mapsSaved, setMapsSaved] = useState(false)
  const [mapsError, setMapsError] = useState('')

  useEffect(() => {
    if (isSpieler) {
      api.get('/profile/me').then(r => {
        setAbsencesPublic(r.data?.own_member?.absences_public === 1 || r.data?.own_member?.absences_public === true)
      }).catch(() => {})
    }
    api.get('/profile/notification-preferences').then(r => {
      const loaded: Prefs = { ...defaults }
      for (const cat of Object.keys(defaults) as Category[]) {
        if (r.data?.[cat]) {
          loaded[cat] = { push: r.data[cat].push ?? true, email: r.data[cat].email ?? false }
        }
      }
      setPrefs(loaded)
    }).catch(() => {})
    // Einmaliger Initial-Load (nur Mount); isSpieler ist für die Komponenten-Lebensdauer stabil
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const toggleAbsenceVisibility = async () => {
    const newValue = !absencesPublic
    setAbsencesPublic(newValue)
    setAbsenceSaving(true)
    try {
      await api.put('/profile/absence-visibility', { public: newValue })
    } catch {
      setAbsencesPublic(!newValue)
    } finally {
      setAbsenceSaving(false)
    }
  }

  const togglePush = (cat: Category) => {
    setPrefs(p => ({ ...p, [cat]: { ...p[cat], push: !p[cat].push } }))
    setChanged(true)
  }

  const toggleEmail = (cat: Category) => {
    setPrefs(p => ({ ...p, [cat]: { ...p[cat], email: !p[cat].email } }))
    setChanged(true)
  }

  const handleMapsProviderChange = async (p: MapsProvider) => {
    setMapsProvider(p)
    setMapsSaving(true)
    setMapsError('')
    setMapsSaved(false)
    try {
      await api.put('/profile/me', { maps_provider: p })
      setCtxMapsProvider(p)
      setMapsSaved(true)
      setTimeout(() => setMapsSaved(false), 2000)
    } catch {
      setMapsError('Speichern fehlgeschlagen')
      setMapsProvider(ctxMapsProvider)
    } finally {
      setMapsSaving(false)
    }
  }

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.put('/profile/notification-preferences', prefs)
      setSaved(true)
      setChanged(false)
      setTimeout(() => setSaved(false), 2000)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <div className="p-6 pb-2">
          <h2 className="font-semibold text-brand-text-muted mb-4">Benachrichtigungen</h2>
          <div className="grid grid-cols-[1fr_auto_auto] items-center gap-x-6 gap-y-0 text-xs text-brand-text-muted uppercase mb-2 px-0">
            <span />
            <span className="text-center w-11">Push</span>
            <span className="text-center w-11">E-Mail</span>
          </div>
        </div>
        <div className="divide-y divide-brand-border-subtle">
          {(Object.keys(defaults) as Category[]).map(cat => (
            <div key={cat} className="grid grid-cols-[1fr_auto_auto] items-center gap-x-6 px-6 py-3">
              <div>
                <p className="text-sm font-medium text-brand-text">{categoryLabels[cat]}</p>
                {categoryDescriptions[cat] && (
                  <p className="text-xs text-brand-text-subtle mt-0.5">{categoryDescriptions[cat]}</p>
                )}
              </div>
              <Toggle
                enabled={prefs[cat].push}
                onToggle={() => togglePush(cat)}
                label={`Push ${categoryLabels[cat]}`}
              />
              <Toggle
                enabled={prefs[cat].email}
                onToggle={() => toggleEmail(cat)}
                label={`E-Mail ${categoryLabels[cat]}`}
              />
            </div>
          ))}
        </div>
      </div>

      {isSpieler && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
          <div className="p-6 pb-2">
            <h2 className="font-semibold text-brand-text-muted mb-1">Sichtbarkeit für Mitglieder</h2>
            <p className="text-xs text-brand-text-subtle mb-3">Wenn aktiv, sehen Trainer deine Abwesenheiten im Kalender.</p>
          </div>
          <div className="divide-y divide-brand-border-subtle">
            <div className="flex items-center justify-between px-6 py-3">
              <p className="text-sm font-medium text-brand-text">Abwesenheiten für Trainer sichtbar</p>
              <Toggle
                enabled={absencesPublic}
                onToggle={absenceSaving ? () => {} : toggleAbsenceVisibility}
                label="Abwesenheiten für Trainer sichtbar"
              />
            </div>
          </div>
        </div>
      )}

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <div className="p-6 pb-2">
          <h2 className="font-semibold text-brand-text-muted mb-1">Kartendienst</h2>
          <p className="text-xs text-brand-text-subtle mb-3">Welche Karten-App beim Klick auf einen Ort geöffnet werden soll.</p>
        </div>
        <div className="divide-y divide-brand-border-subtle">
          <div className="flex items-center justify-between px-6 py-3 gap-4">
            <label htmlFor="maps-provider" className="text-sm font-medium text-brand-text shrink-0">Karten-App</label>
            <div className="flex items-center gap-3">
              <select
                id="maps-provider"
                value={mapsProvider}
                disabled={mapsSaving}
                onChange={e => handleMapsProviderChange(e.target.value as MapsProvider)}
                className="border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow disabled:opacity-50"
              >
                <option value="auto">Automatisch</option>
                <option value="google">Google Maps</option>
                <option value="apple">Apple Maps</option>
              </select>
              {mapsSaved && <span className="text-sm text-brand-text-muted">Gespeichert</span>}
              {mapsError && <span className="text-sm text-brand-danger">{mapsError}</span>}
            </div>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={!changed || saving}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
        {saved && <span className="text-sm text-brand-text-muted">Gespeichert</span>}
        {error && <span className="text-sm text-brand-danger">{error}</span>}
      </div>
    </div>
  )
}

