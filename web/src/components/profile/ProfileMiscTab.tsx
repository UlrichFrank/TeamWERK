import { useState, useEffect, FormEvent } from 'react'
import { api } from '../../lib/api'

type Category = 'games' | 'trainings' | 'duties' | 'carpooling'

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
}

const categoryLabels: Record<Category, string> = {
  games: 'Spiele',
  trainings: 'Trainings',
  duties: 'Dienste',
  carpooling: 'Fahrgemeinschaften',
}

export default function ProfileMiscTab() {
  const [prefs, setPrefs] = useState<Prefs>(defaults)
  const [changed, setChanged] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.get('/profile/notification-preferences').then(r => {
      const loaded: Prefs = { ...defaults }
      for (const cat of Object.keys(defaults) as Category[]) {
        if (r.data?.[cat]) {
          loaded[cat] = { push: r.data[cat].push ?? true, email: r.data[cat].email ?? false }
        }
      }
      setPrefs(loaded)
    }).catch(() => {})
  }, [])

  const togglePush = (cat: Category) => {
    setPrefs(p => ({ ...p, [cat]: { ...p[cat], push: !p[cat].push } }))
    setChanged(true)
  }

  const toggleEmail = (cat: Category) => {
    setPrefs(p => ({ ...p, [cat]: { ...p[cat], email: !p[cat].email } }))
    setChanged(true)
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
              <p className="text-sm font-medium text-brand-text">{categoryLabels[cat]}</p>
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

function Toggle({ enabled, onToggle, label }: { enabled: boolean; onToggle: () => void; label: string }) {
  return (
    <button
      onClick={onToggle}
      aria-label={label}
      className={`relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors ${
        enabled ? 'bg-brand-yellow' : 'bg-brand-border'
      }`}
    >
      <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
        enabled ? 'translate-x-6' : 'translate-x-1'
      }`} />
    </button>
  )
}
