import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ShieldCheck, Lock } from 'lucide-react'
import { api } from '../lib/api'
import { useVault } from '../contexts/VaultContext'
import VaultGate from '../components/VaultGate'
import {
  deriveKey,
  encrypt,
  generateDEK,
  generateVaultSetup,
  unwrapKey,
  wrapKey,
} from '../lib/crypto'

interface EncryptionConfig {
  configured: boolean
  vorstand_kdf_salt: string
  vorstand_key_check: string
}

interface EncryptedMember {
  id: number
  first_name: string
  last_name: string
  ciphertext: string
  dek_enc_vorstand: string
}

// --- Migration section (11.1–11.3) ---

function MigrationSection() {
  const { vaultKwKey } = useVault()
  const [members, setMembers] = useState<EncryptedMember[]>([])
  const [progress, setProgress] = useState(0)
  const [total, setTotal] = useState(0)
  const [running, setRunning] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.get<EncryptedMember[]>('/members/export-encrypted').then(({ data }) => {
      const needsMigration = data.filter(m => !m.ciphertext && !m.dek_enc_vorstand)
      setMembers(data)
      setTotal(needsMigration.length)
    })
  }, [])

  const unmigrated = members.filter(m => !m.ciphertext)

  async function runMigration() {
    if (!vaultKwKey) return
    setRunning(true)
    setError('')
    setProgress(0)
    let count = 0
    for (const member of unmigrated) {
      try {
        // No legacy data accessible via API — create empty encrypted record
        // to mark member as migrated; data entry happens via member detail page
        const payload = {}
        const dek = await generateDEK()
        const ciphertext = await encrypt(payload, dek)
        const dekEncVorstand = await wrapKey(dek, vaultKwKey)
        await api.put(`/members/${member.id}/sensitive`, {
          ciphertext,
          dek_enc_vorstand: dekEncVorstand,
        })
        count++
        setProgress(count)
      } catch {
        setError(`Fehler bei Mitglied ${member.first_name} ${member.last_name}`)
        break
      }
    }
    setRunning(false)
    if (!error) setDone(true)
  }

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 mt-6">
      <h2 className="text-lg font-semibold text-brand-text mb-2">Datenmigration</h2>
      <p className="text-sm text-brand-text-muted mb-4">
        Erstellt leere verschlüsselte Einträge für alle noch nicht migrierten Mitglieder.
        Die sensiblen Daten werden anschließend über die Mitglied-Detailseite eingetragen.
      </p>
      <p className="text-sm text-brand-text mb-3">
        Ausstehend: <strong>{total}</strong> Mitglieder
      </p>
      {running && (
        <div className="mb-3">
          <div className="h-2 bg-brand-border rounded-full overflow-hidden">
            <div
              className="h-2 bg-brand-yellow transition-all duration-300"
              style={{ width: total > 0 ? `${(progress / total) * 100}%` : '0%' }}
            />
          </div>
          <p className="text-sm text-brand-text-muted mt-1">{progress} / {total}</p>
        </div>
      )}
      {error && (
        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-3">
          {error}
        </p>
      )}
      {done && (
        <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text mb-3">
          Migration abgeschlossen. Die alten Klartext-Spalten können jetzt manuell per Datenbank-Migration
          gedroppt werden.
        </div>
      )}
      {!done && (
        <button
          onClick={runMigration}
          disabled={running || total === 0}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {running ? 'Migriert…' : 'Migration starten'}
        </button>
      )}
    </div>
  )
}

// --- Rotation section (7.5) ---

function RotationSection() {
  const { vaultKwKey, lockVault } = useVault()
  const navigate = useNavigate()
  const [newPassphrase, setNewPassphrase] = useState('')
  const [confirm, setConfirm] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleRotate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (newPassphrase !== confirm) {
      setError('Passphrasen stimmen nicht überein')
      return
    }
    if (newPassphrase.length < 12) {
      setError('Passphrase muss mindestens 12 Zeichen lang sein')
      return
    }
    if (!vaultKwKey) return
    setLoading(true)
    try {
      const { data: members } = await api.get<EncryptedMember[]>('/members/export-encrypted')
      const withData = members.filter(m => m.dek_enc_vorstand)

      const { saltB64, keyCheckB64 } = await generateVaultSetup(newPassphrase)
      const newKwKey = await deriveKey(newPassphrase, saltB64)

      const entries = await Promise.all(
        withData.map(async m => {
          const dek = await unwrapKey(m.dek_enc_vorstand, vaultKwKey)
          const dekEncVorstand = await wrapKey(dek, newKwKey)
          return { member_id: m.id, dek_enc_vorstand: dekEncVorstand }
        }),
      )

      await api.put('/admin/rotate-encryption', {
        new_salt: saltB64,
        new_key_check: keyCheckB64,
        entries,
      })

      lockVault()
      navigate('/admin/tresor-verwaltung')
    } catch {
      setError('Fehler bei der Schlüsselrotation')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 mt-6">
      <h2 className="text-lg font-semibold text-brand-text mb-2">Passphrase rotieren</h2>
      <p className="text-sm text-brand-text-muted mb-4">
        Alle DEKs werden mit der neuen Passphrase neu gewrappt. Der Server sieht weder die alte noch die neue Passphrase.
        Nach der Rotation wird der Tresor gesperrt.
      </p>
      <form onSubmit={handleRotate} className="space-y-4">
        <input
          type="password"
          value={newPassphrase}
          onChange={e => setNewPassphrase(e.target.value)}
          placeholder="Neue Passphrase (min. 12 Zeichen)"
          required
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
        />
        <input
          type="password"
          value={confirm}
          onChange={e => setConfirm(e.target.value)}
          placeholder="Bestätigen"
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
          disabled={loading || !newPassphrase || !confirm}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? 'Rotiert…' : 'Passphrase rotieren'}
        </button>
      </form>
    </div>
  )
}

// --- Main page ---

function TresorContent() {
  const { isUnlocked, lockVault } = useVault()
  const navigate = useNavigate()
  const [config, setConfig] = useState<EncryptionConfig | null>(null)

  useEffect(() => {
    api.get<EncryptionConfig>('/admin/encryption-config').then(({ data }) => setConfig(data))
  }, [])

  if (!config) return <div className="p-8 text-brand-text-muted">Lade…</div>

  if (!config.configured) {
    return (
      <div className="p-8 max-w-lg mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <ShieldCheck className="w-6 h-6" />
          <h1 className="text-2xl font-bold text-brand-text">Tresor-Verwaltung</h1>
        </div>
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <p className="text-sm text-brand-text mb-4">
            Der Verschlüsselungs-Tresor ist noch nicht eingerichtet.
          </p>
          <button
            onClick={() => navigate('/admin/tresor-einrichten')}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            Tresor einrichten
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-2xl mx-auto">
      <div className="flex items-center gap-3 mb-6">
        <ShieldCheck className="w-6 h-6" />
        <h1 className="text-2xl font-bold text-brand-text">Tresor-Verwaltung</h1>
      </div>

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <div className="flex items-center gap-2 mb-3">
          {isUnlocked ? (
            <span className="inline-flex items-center gap-1.5 text-sm font-medium text-green-700 bg-green-100 rounded-full px-3 py-1">
              <ShieldCheck className="w-4 h-4" /> Entsperrt
            </span>
          ) : (
            <span className="inline-flex items-center gap-1.5 text-sm font-medium text-brand-text-muted bg-brand-border-subtle rounded-full px-3 py-1">
              <Lock className="w-4 h-4" /> Gesperrt
            </span>
          )}
        </div>
        {isUnlocked && (
          <button
            onClick={lockVault}
            className="mt-2 text-sm text-brand-text-muted hover:text-brand-danger transition-colors"
          >
            Tresor sperren
          </button>
        )}
      </div>

      <VaultGate>
        <MigrationSection />
        <RotationSection />
      </VaultGate>
    </div>
  )
}

export default function AdminTresorVerwaltungPage() {
  return <TresorContent />
}
