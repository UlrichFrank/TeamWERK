import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import { Member, ChangeDraft } from '../../pages/ProfilePage'

const formatIBAN = (raw: string) =>
  raw.replace(/\s/g, '').toUpperCase().match(/.{1,4}/g)?.join(' ') ?? ''

interface Props {
  ownMember: Member | null
}

export default function ProfileMemberTab({ ownMember }: Props) {
  const [drafts, setDrafts] = useState<ChangeDraft[]>([])

  // Name-Änderung
  const [nameFirst, setNameFirst] = useState('')
  const [nameLast, setNameLast] = useState('')
  const [nameSaving, setNameSaving] = useState(false)
  const [nameError, setNameError] = useState('')
  const [nameSaved, setNameSaved] = useState(false)

  // IBAN-Änderung
  const [iban, setIban] = useState('')
  const [ibanDisplay, setIbanDisplay] = useState('')
  const [ibanSaving, setIbanSaving] = useState(false)
  const [ibanError, setIbanError] = useState('')
  const [ibanSaved, setIbanSaved] = useState(false)

  const [cancelError, setCancelError] = useState('')

  useEffect(() => {
    if (ownMember) {
      setNameFirst(ownMember.first_name)
      setNameLast(ownMember.last_name)
      const rawIban = ownMember.iban ?? ''
      setIban(rawIban)
      setIbanDisplay(formatIBAN(rawIban))
      loadDrafts()
    }
  }, [ownMember?.id])

  const loadDrafts = async () => {
    if (!ownMember) return
    try {
      const r = await api.get(`/members/${ownMember.id}/change-drafts`)
      setDrafts(r.data?.drafts ?? [])
    } catch {}
  }

  const getDraft = (field: string) => drafts.find(d => d.field_name === field)

  const handleCancelDraft = async (draftId: number) => {
    if (!ownMember) return
    setCancelError('')
    try {
      await api.delete(`/members/${ownMember.id}/change-drafts/${draftId}`)
      setDrafts(prev => prev.filter(d => d.id !== draftId))
    } catch {
      setCancelError('Fehler beim Abbrechen')
    }
  }

  const handleNameRequest = async () => {
    if (!ownMember) return
    if (!nameFirst.trim() || !nameLast.trim()) {
      setNameError('Vor- und Nachname dürfen nicht leer sein.')
      return
    }
    setNameSaving(true)
    setNameError('')
    try {
      await api.post(`/members/${ownMember.id}/change-request`, {
        field_name: 'name',
        new_value: { first_name: nameFirst.trim(), last_name: nameLast.trim() },
      })
      await loadDrafts()
      setNameSaved(true)
      setTimeout(() => setNameSaved(false), 2000)
    } catch {
      setNameError('Fehler beim Senden der Änderungsanfrage.')
    } finally {
      setNameSaving(false)
    }
  }

  const handleIbanChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
    if (raw.length > 22) return
    setIban(raw)
    setIbanDisplay(raw.match(/.{1,4}/g)?.join(' ') ?? raw)
    setIbanError('')
  }

  const handleIbanRequest = async () => {
    if (!ownMember) return
    if (!iban.trim()) {
      setIbanError('IBAN darf nicht leer sein.')
      return
    }
    setIbanSaving(true)
    setIbanError('')
    try {
      await api.post(`/members/${ownMember.id}/change-request`, {
        field_name: 'iban',
        new_value: iban,
      })
      await loadDrafts()
      setIbanSaved(true)
      setTimeout(() => setIbanSaved(false), 2000)
    } catch {
      setIbanError('Fehler beim Senden der Änderungsanfrage.')
    } finally {
      setIbanSaving(false)
    }
  }

  if (!ownMember) {
    return <div className="text-gray-500">Keine Mitgliedsdaten verfügbar.</div>
  }

  const formatDate = (s: string) => {
    if (!s) return '–'
    return new Date(s).toLocaleDateString('de-DE')
  }

  const nameDraft = getDraft('name')
  const ibanDraft = getDraft('iban')

  const nameChanged = nameFirst !== ownMember.first_name || nameLast !== ownMember.last_name
  const ibanChanged = iban !== (ownMember.iban ?? '')

  return (
    <div className="space-y-6">
      {/* Stammdaten — read-only */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Stammdaten</h2>
        <div className="space-y-3 text-sm">
          <Row label="Vorname" value={ownMember.first_name} />
          <Row label="Nachname" value={ownMember.last_name} />
          <Row label="Geburtsdatum" value={formatDate(ownMember.date_of_birth)} />
          <Row label="Passnummer" value={ownMember.pass_number || '–'} />
          <Row label="Rückennummer" value={ownMember.jersey_number?.toString() ?? '–'} />
          <Row label="Position" value={ownMember.position || '–'} />
          <Row label="Status" value={ownMember.status || '–'} />
        </div>
      </div>

      {/* Name ändern */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-1">Name ändern</h2>
        <p className="text-xs text-gray-500 mb-4">Namensänderungen müssen vom Verein übernommen werden.</p>

        {nameDraft && (
          <div className="mb-4 text-xs text-gray-600 p-3 bg-blue-50 rounded-lg border border-blue-100">
            <span className="font-medium text-blue-700">Ausstehend:</span>{' '}
            {nameDraft.new_value?.first_name} {nameDraft.new_value?.last_name}
            <button
              onClick={() => handleCancelDraft(nameDraft.id)}
              className="ml-3 text-red-600 hover:text-red-800 underline"
            >
              Abbrechen
            </button>
          </div>
        )}

        <div className="grid grid-cols-2 gap-3 mb-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
            <input
              type="text"
              value={nameFirst}
              onChange={e => setNameFirst(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
            <input
              type="text"
              value={nameLast}
              onChange={e => setNameLast(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={handleNameRequest}
            disabled={nameSaving || !nameChanged}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
          >
            {nameSaving ? 'Senden…' : 'Änderung anfordern'}
          </button>
          {nameSaved && <span className="text-sm text-green-600">Anfrage gesendet</span>}
          {nameError && <span className="text-sm text-red-600">{nameError}</span>}
        </div>
      </div>

      {/* IBAN ändern */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-1">IBAN</h2>
        <p className="text-xs text-gray-500 mb-4">IBAN-Änderungen müssen vom Verein übernommen werden.</p>

        {ibanDraft && (
          <div className="mb-4 text-xs text-gray-600 p-3 bg-blue-50 rounded-lg border border-blue-100">
            <span className="font-medium text-blue-700">Ausstehend:</span>{' '}
            <span className="font-mono">{ibanDraft.new_value}</span>
            <button
              onClick={() => handleCancelDraft(ibanDraft.id)}
              className="ml-3 text-red-600 hover:text-red-800 underline"
            >
              Abbrechen
            </button>
          </div>
        )}

        <div className="mb-3 text-sm">
          <span className="font-medium text-gray-700">Kontoinhaber: </span>
          <span className="text-gray-900">{ownMember.account_holder || '–'}</span>
        </div>
        <div className="mb-3">
          <label className="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
          <input
            type="text"
            value={ibanDisplay}
            onChange={handleIbanChange}
            placeholder="DE89 3704 0044 0532 0130 00"
            maxLength={27}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono tracking-wider"
          />
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={handleIbanRequest}
            disabled={ibanSaving || !ibanChanged}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
          >
            {ibanSaving ? 'Senden…' : 'Änderung anfordern'}
          </button>
          {ibanSaved && <span className="text-sm text-green-600">Anfrage gesendet</span>}
          {ibanError && <span className="text-sm text-red-600">{ibanError}</span>}
        </div>
      </div>

      {cancelError && <p className="text-sm text-red-600">{cancelError}</p>}
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex gap-2">
      <span className="text-gray-500 w-36 shrink-0">{label}:</span>
      <span className="text-gray-900">{value}</span>
    </div>
  )
}
