import { useEffect, useState, FormEvent } from 'react'
import { ShieldCheck, Lock, LockOpen, AlertTriangle } from 'lucide-react'
import { api } from '../../lib/api'
import { useVault } from '../../contexts/VaultContext'
import { useLiveUpdates } from '../../hooks/useLiveUpdates'
import { generateVaultSetup } from '../../lib/crypto'

const INPUT =
  'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_PRIMARY =
  'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6'
const ALERT_ERR = 'p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger'
const ALERT_INFO = 'p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text'

export default function TresorPage() {
  const { isUnlocked, unlock, lock } = useVault()
  const [configured, setConfigured] = useState<boolean | null>(null)
  const [pass, setPass] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const load = () => {
    api.get<{ configured: boolean }>('/admin/encryption-config').then(r => setConfigured(r.data.configured))
  }
  useEffect(load, [])
  useLiveUpdates(event => {
    if (event === 'settings') load()
  })

  async function handleSetup(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (pass.length < 12) {
      setError('Die Passphrase muss mindestens 12 Zeichen lang sein.')
      return
    }
    if (pass !== confirm) {
      setError('Die Passphrasen stimmen nicht überein.')
      return
    }
    setBusy(true)
    try {
      const { saltB64, keyCheckB64 } = await generateVaultSetup(pass)
      await api.put('/admin/encryption-config', {
        vorstand_kdf_salt: saltB64,
        vorstand_key_check: keyCheckB64,
      })
      await unlock(pass)
      setPass('')
      setConfirm('')
      setConfigured(true)
    } catch {
      setError('Einrichtung fehlgeschlagen.')
    } finally {
      setBusy(false)
    }
  }

  async function handleUnlock(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      const ok = await unlock(pass)
      if (!ok) setError('Falsche Passphrase.')
      else setPass('')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="max-w-xl space-y-6">
      <div className="flex items-center gap-2">
        <ShieldCheck className="w-6 h-6 text-brand-text" />
        <h1 className="text-xl font-semibold text-brand-text">Bankdaten-Tresor</h1>
      </div>

      {configured === null && <p className="text-sm text-brand-text-muted">Lädt…</p>}

      {/* Eingerichtet → Entsperren / Status */}
      {configured === true && (
        <div className={CARD}>
          {isUnlocked ? (
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-brand-text">
                <LockOpen className="w-5 h-5" />
                <span className="text-sm font-medium">Tresor entsperrt</span>
              </div>
              <p className="text-sm text-brand-text-muted">
                Bankdaten können in dieser Sitzung gelesen und geschrieben werden. Der Schlüssel wird nach
                30 Minuten Inaktivität automatisch gesperrt.
              </p>
              <button onClick={lock} className={BTN_PRIMARY}>
                <span className="inline-flex items-center gap-2">
                  <Lock className="w-4 h-4" /> Jetzt sperren
                </span>
              </button>
            </div>
          ) : (
            <form onSubmit={handleUnlock} className="space-y-4">
              <div className="flex items-center gap-2 text-brand-text">
                <Lock className="w-5 h-5" />
                <span className="text-sm font-medium">Tresor gesperrt</span>
              </div>
              <input
                type="password"
                className={INPUT}
                placeholder="Tresor-Passphrase"
                value={pass}
                onChange={e => setPass(e.target.value)}
                autoFocus
              />
              {error && <div className={ALERT_ERR}>{error}</div>}
              <button type="submit" disabled={busy || !pass} className={BTN_PRIMARY}>
                Entsperren
              </button>
            </form>
          )}
        </div>
      )}

      {/* Nicht eingerichtet → Einrichtung */}
      {configured === false && (
        <div className={CARD}>
          <form onSubmit={handleSetup} className="space-y-4">
            <p className="text-sm text-brand-text">
              Lege eine gemeinsame Tresor-Passphrase für Vorstand und Kassierer fest. Mit ihr werden die
              Bankdaten clientseitig ver- und entschlüsselt.
            </p>
            <div className="flex items-start gap-2 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg">
              <AlertTriangle className="w-5 h-5 text-brand-danger shrink-0 mt-0.5" />
              <div className="text-sm text-brand-danger">
                <strong>Kein Zurücksetzen möglich.</strong> Die Passphrase wird nirgends gespeichert. Geht sie
                verloren, sind <strong>alle Bankdaten unwiederbringlich verloren</strong>. Sie sollte mindestens
                zwei verantwortlichen Personen bekannt sein.
              </div>
            </div>
            <input
              type="password"
              className={INPUT}
              placeholder="Neue Passphrase (min. 12 Zeichen)"
              value={pass}
              onChange={e => setPass(e.target.value)}
            />
            <input
              type="password"
              className={INPUT}
              placeholder="Passphrase bestätigen"
              value={confirm}
              onChange={e => setConfirm(e.target.value)}
            />
            {error && <div className={ALERT_ERR}>{error}</div>}
            <div className={ALERT_INFO}>
              Bewahre die Passphrase sicher auf (z. B. Passwort-Manager des Vorstands).
            </div>
            <button type="submit" disabled={busy || !pass || !confirm} className={BTN_PRIMARY}>
              Tresor einrichten
            </button>
          </form>
        </div>
      )}
    </div>
  )
}
