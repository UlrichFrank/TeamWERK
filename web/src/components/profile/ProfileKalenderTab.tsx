import { useEffect, useState } from 'react'
import { Copy, Check, Trash2, Smartphone } from 'lucide-react'
import { api } from '../../lib/api'
import Toggle from '../Toggle'

type Toggles = {
  include_heim: boolean
  include_auswaerts: boolean
  include_training: boolean
  include_generisch: boolean
  include_duty: boolean
}

type TokenResponse = Toggles & { token: string }

const ALL_ON: Toggles = {
  include_heim: true,
  include_auswaerts: true,
  include_training: true,
  include_generisch: true,
  include_duty: true,
}

const labels: Record<keyof Toggles, string> = {
  include_heim: 'Heim-Spiele',
  include_auswaerts: 'Auswärts-Spiele',
  include_training: 'Trainings',
  include_generisch: 'Sonstige Events',
  include_duty: 'Dienste',
}

const BTN_PRIMARY = 'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_DANGER = 'bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed'

export default function ProfileKalenderTab() {
  const [token, setToken] = useState<string | null>(null)
  const [toggles, setToggles] = useState<Toggles>(ALL_ON)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')
  const [dirty, setDirty] = useState(false)

  useEffect(() => {
    api.get<TokenResponse>('/calendar/token')
      .then(r => {
        setToken(r.data.token)
        setToggles({
          include_heim: r.data.include_heim,
          include_auswaerts: r.data.include_auswaerts,
          include_training: r.data.include_training,
          include_generisch: r.data.include_generisch,
          include_duty: r.data.include_duty,
        })
      })
      .catch(() => {
        setToken(null)
      })
      .finally(() => setLoading(false))
  }, [])

  const flip = (key: keyof Toggles) => {
    setToggles(t => ({ ...t, [key]: !t[key] }))
    setDirty(true)
  }

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      const r = await api.post<TokenResponse>('/calendar/token', toggles)
      setToken(r.data.token)
      setDirty(false)
    } catch {
      setError('Speichern fehlgeschlagen')
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!confirm('Kalender-Link wirklich löschen? Bestehende Abonnements werden danach leer angezeigt.')) return
    setDeleting(true)
    setError('')
    try {
      await api.delete('/calendar/token')
      setToken(null)
      setToggles(ALL_ON)
      setDirty(false)
    } catch {
      setError('Löschen fehlgeschlagen')
    } finally {
      setDeleting(false)
    }
  }

  const copyToClipboard = async () => {
    if (!token) return
    const url = `${window.location.origin}/api/calendar/feed/${token}`
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      setError('Kopieren fehlgeschlagen')
    }
  }

  if (loading) {
    return <p className="text-sm text-brand-text-muted">Lädt…</p>
  }

  const feedUrl = token ? `${window.location.origin}/api/calendar/feed/${token}` : ''

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <div className="p-6 pb-2">
          <h2 className="font-semibold text-brand-text-muted mb-1">Kalender-Abo</h2>
          <p className="text-xs text-brand-text-subtle mb-3">
            Abonniere diesen Link in Google Calendar, Apple Kalender oder Outlook. Der Link ist privat — teile ihn nicht.
          </p>
        </div>
        <div className="divide-y divide-brand-border-subtle">
          {(Object.keys(labels) as (keyof Toggles)[]).map(key => (
            <div key={key} className="flex items-center justify-between px-6 py-3">
              <p className="text-sm font-medium text-brand-text">{labels[key]}</p>
              <Toggle
                enabled={toggles[key]}
                onToggle={() => flip(key)}
                label={labels[key]}
              />
            </div>
          ))}
        </div>
      </div>

      {token && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h3 className="font-semibold text-brand-text-muted mb-2">Feed-URL</h3>
          <div className="flex items-center gap-2 mb-4">
            <input
              type="text"
              readOnly
              value={feedUrl}
              className="flex-1 border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              onFocus={e => e.target.select()}
            />
            <button
              onClick={copyToClipboard}
              className={BTN_PRIMARY}
              aria-label="Link kopieren"
            >
              {copied
                ? <span className="inline-flex items-center gap-1"><Check className="w-4 h-4" /> Kopiert</span>
                : <span className="inline-flex items-center gap-1"><Copy className="w-4 h-4" /> Kopieren</span>}
            </button>
          </div>
        </div>
      )}

      {token && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 space-y-5">
          <h3 className="font-semibold text-brand-text-muted">Anleitung: Kalender abonnieren</h3>

          <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            <strong>Hinweis:</strong> Dieser Link enthält ausschließlich deine eigenen Termine.
            Termine deiner Kinder oder anderer Familienmitglieder sind nicht enthalten — jede
            Person braucht ihren eigenen Link aus ihrem jeweiligen Profil.
          </div>

          <div>
            <div className="flex items-center gap-2 mb-2">
              <Smartphone className="w-5 h-5 text-brand-text" aria-hidden="true" />
              <h4 className="font-medium text-brand-text">iPhone / iPad (Apple Kalender)</h4>
            </div>
            <ol className="list-decimal pl-5 text-sm text-brand-text space-y-1">
              <li>Oben auf <em>Kopieren</em> tippen, um die Feed-URL in die Zwischenablage zu legen.</li>
              <li>Einstellungen-App öffnen → <em>Apps</em> → <em>Kalender</em> → <em>Kalender-Accounts</em> → <em>Account hinzufügen</em> → <em>Andere</em>.</li>
              <li><em>Kalenderabo hinzufügen</em> wählen und den kopierten Link als Server einfügen.</li>
              <li>Auf <em>Weiter</em> und dann <em>Sichern</em> tippen. Die Termine erscheinen in der Kalender-App.</li>
            </ol>
          </div>

          <div>
            <div className="flex items-center gap-2 mb-2">
              <Smartphone className="w-5 h-5 text-brand-text" aria-hidden="true" />
              <h4 className="font-medium text-brand-text">Android (Google Kalender)</h4>
            </div>
            <ol className="list-decimal pl-5 text-sm text-brand-text space-y-1">
              <li>Oben auf <em>Kopieren</em> tippen, um die Feed-URL in die Zwischenablage zu legen.</li>
              <li>Im Browser <code className="text-xs">calendar.google.com</code> öffnen (am einfachsten am Computer; auf dem Handy in der Desktop-Ansicht).</li>
              <li>Links neben <em>Weitere Kalender</em> auf das <em>+</em> tippen und <em>Per URL</em> wählen.</li>
              <li>Den kopierten Link einfügen und auf <em>Kalender hinzufügen</em> tippen. Nach wenigen Minuten erscheinen die Termine auch in der Google-Kalender-App auf dem Handy.</li>
            </ol>
          </div>
        </div>
      )}

      <div className="flex items-center gap-3 flex-wrap">
        <button
          onClick={save}
          disabled={saving || (!dirty && token !== null)}
          className={BTN_PRIMARY}
        >
          {saving ? 'Speichern…' : token ? 'Änderungen speichern' : 'Link aktivieren'}
        </button>
        {token && (
          <button
            onClick={remove}
            disabled={deleting}
            className={BTN_DANGER}
            aria-label="Kalender-Link löschen"
          >
            <span className="inline-flex items-center gap-1">
              <Trash2 className="w-4 h-4" /> {deleting ? 'Löschen…' : 'Link löschen'}
            </span>
          </button>
        )}
        {error && <span className="text-sm text-brand-danger">{error}</span>}
      </div>
    </div>
  )
}
