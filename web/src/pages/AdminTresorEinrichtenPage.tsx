import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ShieldCheck, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { generateVaultSetup } from '../lib/crypto'

export default function AdminTresorEinrichtenPage() {
  const navigate = useNavigate()
  const [passphrase, setPassphrase] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (passphrase !== confirm) {
      setError('Passphrasen stimmen nicht überein')
      return
    }
    if (passphrase.length < 12) {
      setError('Passphrase muss mindestens 12 Zeichen lang sein')
      return
    }
    setLoading(true)
    try {
      const { saltB64, keyCheckB64 } = await generateVaultSetup(passphrase)
      await api.put('/admin/encryption-config', {
        vorstand_kdf_salt: saltB64,
        vorstand_key_check: keyCheckB64,
      })
      navigate('/admin/tresor-verwaltung')
    } catch (err: any) {
      if (err?.response?.status === 409) {
        setError('Tresor ist bereits eingerichtet. Verwende die Rotationsfunktion.')
      } else {
        setError('Fehler beim Einrichten des Tresors')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="p-8 max-w-lg mx-auto">
      <div className="flex items-center gap-3 mb-6">
        <ShieldCheck className="w-6 h-6" />
        <h1 className="text-2xl font-bold text-brand-text">Tresor einrichten</h1>
      </div>

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-6">
        <div className="flex gap-3 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg mb-4">
          <AlertTriangle className="w-5 h-5 text-brand-danger shrink-0 mt-0.5" />
          <p className="text-sm text-brand-text">
            <strong>Wichtig:</strong> Die Passphrase wird nicht gespeichert. Wenn sie verloren geht, sind
            alle verschlüsselten Mitgliedsdaten unwiderruflich verloren. Zwei-Personen-Regel empfohlen.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">Passphrase</label>
            <input
              type="password"
              value={passphrase}
              onChange={e => setPassphrase(e.target.value)}
              placeholder="Mindestens 12 Zeichen"
              required
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">Bestätigen</label>
            <input
              type="password"
              value={confirm}
              onChange={e => setConfirm(e.target.value)}
              placeholder="Passphrase wiederholen"
              required
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
          <button
            type="submit"
            disabled={loading || !passphrase || !confirm}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {loading ? 'Richte ein…' : 'Tresor einrichten'}
          </button>
        </form>
      </div>
    </div>
  )
}
