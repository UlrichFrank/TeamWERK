import { useState } from 'react'
import { Lock } from 'lucide-react'
import { useVault } from '../contexts/VaultContext'

interface Props {
  onUnlocked?: () => void
}

export default function VaultPassphraseDialog({ onUnlocked }: Props) {
  const { unlockVault } = useVault()
  const [passphrase, setPassphrase] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const ok = await unlockVault(passphrase)
      if (ok) {
        setPassphrase('')
        onUnlocked?.()
      } else {
        setError('Falsche Passphrase')
      }
    } catch {
      setError('Fehler beim Entsperren')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
        <div className="flex items-center gap-3 mb-4">
          <Lock className="w-5 h-5 text-brand-text" />
          <h2 className="text-lg font-semibold text-brand-text">Tresor entsperren</h2>
        </div>
        <p className="text-sm text-brand-text-muted mb-4">
          Gib die Tresor-Passphrase ein, um auf verschlüsselte Mitgliedsdaten zuzugreifen.
        </p>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input
            type="password"
            value={passphrase}
            onChange={e => setPassphrase(e.target.value)}
            placeholder="Passphrase"
            autoFocus
            required
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
          <button
            type="submit"
            disabled={loading || !passphrase}
            className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {loading ? 'Entsperre…' : 'Entsperren'}
          </button>
        </form>
      </div>
    </div>
  )
}
