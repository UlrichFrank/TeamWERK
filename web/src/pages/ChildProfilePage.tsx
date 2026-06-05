import { useState, useEffect, FormEvent } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

interface ChildMember {
  id: number
  first_name: string
  last_name: string
  date_of_birth: string
  pass_number: string
  jersey_number?: number
  position: string
  status: string
  user_id?: number
  street?: string
  zip?: string
  city?: string
  iban?: string
  account_holder?: string
}

type TabName = 'kontakt' | 'mitglied' | 'bank'

export default function ChildProfilePage() {
  const { memberId } = useParams<{ memberId: string }>()
  const navigate = useNavigate()
  const [member, setMember] = useState<ChildMember | null>(null)
  const [activeTab, setActiveTab] = useState<TabName>('kontakt')

  const load = () => {
    api.get(`/profile/kind/${memberId}`)
      .then(r => setMember(r.data))
      .catch(err => {
        if (err.response?.status === 403) navigate('/')
      })
  }

  useEffect(() => { load() }, [memberId])
  useLiveUpdates(event => { if (event === 'members') load() })

  if (!member) return null

  const title = `${member.first_name}s Profil`

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-bold mb-6">{title}</h1>

      <div className="flex gap-1 mb-6 border-b border-brand-border-subtle flex-wrap">
        {(['kontakt', 'mitglied', 'bank'] as TabName[]).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab
                ? 'border-brand-yellow text-brand-text'
                : 'border-transparent text-brand-text-muted hover:text-brand-text'
            }`}
          >
            {tab === 'kontakt' ? 'Kontakt' : tab === 'mitglied' ? 'Mitgliedsdaten' : 'Bankdaten'}
          </button>
        ))}
      </div>

      {activeTab === 'kontakt' && (
        <KontaktTab member={member} onSaved={load} />
      )}
      {activeTab === 'mitglied' && (
        <MitgliedTab member={member} onSaved={load} />
      )}
      {activeTab === 'bank' && (
        <BankTab member={member} onSaved={load} />
      )}
    </div>
  )
}

const inputCls = `w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`

function KontaktTab({ member, onSaved }: { member: ChildMember; onSaved: () => void }) {
  const [street, setStreet] = useState(member.street ?? '')
  const [zip, setZip] = useState(member.zip ?? '')
  const [city, setCity] = useState(member.city ?? '')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setStreet(member.street ?? '')
    setZip(member.zip ?? '')
    setCity(member.city ?? '')
  }, [member.id])

  const changed =
    street !== (member.street ?? '') ||
    zip !== (member.zip ?? '') ||
    city !== (member.city ?? '')

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.put(`/profile/kind/${member.id}/member`, {
        first_name: member.first_name,
        last_name: member.last_name,
        date_of_birth: member.date_of_birth,
        jersey_number: member.jersey_number ?? null,
        position: member.position,
        street,
        zip,
        city,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      onSaved()
    } catch {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Adresse</h2>
        <form onSubmit={handleSave} className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Straße</label>
            <input type="text" value={street} onChange={e => setStreet(e.target.value)} className={inputCls} />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">PLZ</label>
              <input type="text" value={zip} onChange={e => setZip(e.target.value)} className={inputCls} />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Ort</label>
              <input type="text" value={city} onChange={e => setCity(e.target.value)} className={inputCls} />
            </div>
          </div>
          {changed && (
            <div className="flex items-center gap-3 pt-1">
              <button
                type="submit"
                disabled={saving}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {saving ? 'Speichern…' : 'Speichern'}
              </button>
              {saved && <span className="text-sm text-green-600">Gespeichert</span>}
              {error && <span className="text-sm text-brand-danger">{error}</span>}
            </div>
          )}
        </form>
      </div>
    </div>
  )
}

function MitgliedTab({ member, onSaved }: { member: ChildMember; onSaved: () => void }) {
  const [firstName, setFirstName] = useState(member.first_name)
  const [lastName, setLastName] = useState(member.last_name)
  const [dob, setDob] = useState(member.date_of_birth ? member.date_of_birth.slice(0, 10) : '')
  const [jerseyNumber, setJerseyNumber] = useState(member.jersey_number?.toString() ?? '')
  const [position, setPosition] = useState(member.position)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setFirstName(member.first_name)
    setLastName(member.last_name)
    setDob(member.date_of_birth ? member.date_of_birth.slice(0, 10) : '')
    setJerseyNumber(member.jersey_number?.toString() ?? '')
    setPosition(member.position)
  }, [member.id])

  const changed =
    firstName !== member.first_name ||
    lastName !== member.last_name ||
    dob !== (member.date_of_birth ? member.date_of_birth.slice(0, 10) : '') ||
    jerseyNumber !== (member.jersey_number?.toString() ?? '') ||
    position !== member.position

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      const jerseyNum = jerseyNumber !== '' ? parseInt(jerseyNumber, 10) : null
      await api.put(`/profile/kind/${member.id}/member`, {
        first_name: firstName,
        last_name: lastName,
        date_of_birth: dob,
        jersey_number: jerseyNum,
        position,
        street: member.street ?? '',
        zip: member.zip ?? '',
        city: member.city ?? '',
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      onSaved()
    } catch {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  const formatDate = (s: string) => {
    if (!s) return '–'
    return new Date(s + 'T12:00:00').toLocaleDateString('de-DE')
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Stammdaten</h2>
        <div className="space-y-3 text-sm mb-4">
          <Row label="Passnummer" value={member.pass_number || '–'} />
          <Row label="Status" value={member.status || '–'} />
        </div>
        <form onSubmit={handleSave} className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Vorname</label>
              <input type="text" value={firstName} onChange={e => setFirstName(e.target.value)} className={inputCls} />
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Nachname</label>
              <input type="text" value={lastName} onChange={e => setLastName(e.target.value)} className={inputCls} />
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">
                Geburtsdatum
                {dob && <span className="ml-2 font-normal text-brand-text-subtle">({formatDate(dob)})</span>}
              </label>
              <input type="date" value={dob} onChange={e => setDob(e.target.value)} className={inputCls} />
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Rückennummer</label>
              <input
                type="number"
                min="1"
                max="99"
                value={jerseyNumber}
                onChange={e => setJerseyNumber(e.target.value)}
                placeholder="–"
                className={inputCls}
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Position</label>
            <input type="text" value={position} onChange={e => setPosition(e.target.value)} placeholder="–" className={inputCls} />
          </div>
          {changed && (
            <div className="flex items-center gap-3 pt-1">
              <button
                type="submit"
                disabled={saving}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {saving ? 'Speichern…' : 'Speichern'}
              </button>
              {saved && <span className="text-sm text-green-600">Gespeichert</span>}
              {error && <span className="text-sm text-brand-danger">{error}</span>}
            </div>
          )}
        </form>
      </div>
    </div>
  )
}

function BankTab({ member, onSaved }: { member: ChildMember; onSaved: () => void }) {
  const [iban, setIban] = useState(member.iban ?? '')
  const [accountHolder, setAccountHolder] = useState(member.account_holder ?? '')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setIban(member.iban ?? '')
    setAccountHolder(member.account_holder ?? '')
  }, [member.id])

  const ibanChanged = iban.replace(/\s/g, '') !== (member.iban ?? '').replace(/\s/g, '')
  const ahChanged = accountHolder !== (member.account_holder ?? '')
  const changed = ibanChanged || ahChanged

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      const raw = iban.replace(/\s/g, '').toUpperCase()
      await api.put(`/profile/kind/${member.id}/bank`, {
        iban: raw,
        account_holder: accountHolder,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      onSaved()
    } catch {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Bankdaten</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Kontoinhaber</label>
            <input
              type="text"
              value={accountHolder}
              onChange={e => setAccountHolder(e.target.value)}
              placeholder="Nicht hinterlegt"
              className={inputCls}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">IBAN</label>
            <input
              type="text"
              value={iban}
              onChange={e => {
                const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
                if (raw.length <= 22) setIban(raw)
              }}
              placeholder="DE89 3704 0044 0532 0130 00"
              className={`${inputCls} font-mono tracking-wider`}
            />
          </div>
        </div>
        {changed && (
          <div className="flex items-center gap-3 mt-4">
            <button
              onClick={handleSave}
              disabled={saving}
              className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {saving ? 'Speichern…' : 'Speichern'}
            </button>
            {saved && <span className="text-sm text-green-600">Gespeichert</span>}
            {error && <span className="text-sm text-brand-danger">{error}</span>}
          </div>
        )}
      </div>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex gap-2">
      <span className="text-brand-text-muted w-36 shrink-0">{label}:</span>
      <span className="text-brand-text">{value}</span>
    </div>
  )
}
