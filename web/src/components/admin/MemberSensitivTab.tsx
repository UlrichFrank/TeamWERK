import { useEffect, useState } from 'react'
import { Lock } from 'lucide-react'
import { api } from '../../lib/api'
import { useVault } from '../../contexts/VaultContext'
import VaultGate from '../VaultGate'
import { decrypt, encrypt, generateDEK, unwrapKey, wrapKey } from '../../lib/crypto'

interface SensitivePayload {
  date_of_birth?: string
  street?: string
  zip?: string
  city?: string
  iban?: string
  account_holder?: string
}

interface SensitiveResponse {
  ciphertext: string
  dek_enc_vorstand: string
  dek_enc_member?: string
  member_salt?: string
}

interface Props {
  memberId: number
  memberUserId?: number
}

function SensitiveForm({ memberId }: Props) {
  const { vaultKwKey } = useVault()
  const [raw, setRaw] = useState<SensitiveResponse | null>(null)
  const [payload, setPayload] = useState<SensitivePayload>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!vaultKwKey) return
    setLoading(true)
    api.get<SensitiveResponse>(`/members/${memberId}/sensitive`)
      .then(async ({ data, status }) => {
        if ((status as number) === 204 || !data?.ciphertext) {
          setPayload({})
          setRaw(null)
          return
        }
        setRaw(data)
        const dek = await unwrapKey(data.dek_enc_vorstand, vaultKwKey)
        const decrypted = await decrypt(data.ciphertext, dek)
        setPayload(decrypted as SensitivePayload)
      })
      .catch(() => setError('Fehler beim Laden der sensitiven Daten'))
      .finally(() => setLoading(false))
  }, [memberId, vaultKwKey])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!vaultKwKey) return
    setSaving(true)
    setError('')
    try {
      let dek
      if (raw?.dek_enc_vorstand) {
        dek = await unwrapKey(raw.dek_enc_vorstand, vaultKwKey)
      } else {
        dek = await generateDEK()
      }
      const ciphertext = await encrypt(payload, dek)
      const dekEncVorstand = await wrapKey(dek, vaultKwKey)

      await api.put(`/members/${memberId}/sensitive`, {
        ciphertext,
        dek_enc_vorstand: dekEncVorstand,
        dek_enc_member: raw?.dek_enc_member ?? null,
        member_salt: raw?.member_salt ?? null,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  function field(key: keyof SensitivePayload) {
    return {
      value: payload[key] ?? '',
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        setPayload(p => ({ ...p, [key]: e.target.value })),
    }
  }

  if (loading) return <p className="text-sm text-brand-text-muted">Lade…</p>

  return (
    <form onSubmit={handleSave} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 space-y-4">
      <p className="flex items-center gap-2 text-xs text-brand-text-muted">
        <Lock className="w-3.5 h-3.5" /> Verschlüsselt – nur im Browser entschlüsselt
      </p>

      <div>
        <label className="block text-sm font-medium text-brand-text mb-1">Geburtsdatum</label>
        <input type="date" {...field('date_of_birth')}
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
      </div>

      <fieldset>
        <legend className="text-sm font-medium text-brand-text mb-2">Adresse</legend>
        <div className="space-y-2">
          <input type="text" placeholder="Straße" {...field('street')}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
          <div className="flex gap-2">
            <input type="text" placeholder="PLZ" {...field('zip')}
              className="w-28 border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
            <input type="text" placeholder="Ort" {...field('city')}
              className="flex-1 border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
          </div>
        </div>
      </fieldset>

      <div>
        <label className="block text-sm font-medium text-brand-text mb-1">IBAN</label>
        <input type="text" placeholder="DE00 0000 0000 0000 0000 00" {...field('iban')}
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text font-mono placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
      </div>

      <div>
        <label className="block text-sm font-medium text-brand-text mb-1">Kontoinhaber</label>
        <input type="text" placeholder="Name des Kontoinhabers" {...field('account_holder')}
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
      </div>

      {error && (
        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
          {error}
        </p>
      )}
      {saved && (
        <p className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
          Gespeichert
        </p>
      )}

      <button type="submit" disabled={saving}
        className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
        {saving ? 'Speichere…' : 'Speichern'}
      </button>
    </form>
  )
}

export default function MemberSensitivTab({ memberId, memberUserId }: Props) {
  return (
    <VaultGate>
      <SensitiveForm memberId={memberId} memberUserId={memberUserId} />
    </VaultGate>
  )
}
