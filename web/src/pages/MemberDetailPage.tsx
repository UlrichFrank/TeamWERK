import { useEffect, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; member_number: string; pass_number: string
  jersey_number?: number; position: string; status: string; user_id?: number
}

interface User { id: number; name: string; email: string; role: string }

const STATUS_OPTIONS = ['aktiv', 'verletzt', 'pausiert', 'passiv', 'ausgetreten']
const HANDBALL_POSITIONS = ['Torwart', 'Linksaußen', 'Rechtsaußen', 'Rückraum Links', 'Rückraum Mitte', 'Rückraum Rechts', 'Kreisläufer']

export default function MemberDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isNew = id === 'neu'
  const isAdmin = user?.role === 'admin'

  const [form, setForm] = useState<Omit<Member, 'id'>>({
    first_name: '', last_name: '', date_of_birth: '', member_number: '', pass_number: '',
    jersey_number: undefined, position: '', status: 'aktiv',
  })
  const [users, setUsers] = useState<User[]>([])
  const [selectedParentUser, setSelectedParentUser] = useState('')
  const [linkedParents, setLinkedParents] = useState<User[]>([])
  const [selectedLinkedUser, setSelectedLinkedUser] = useState('')
  const [currentUserID, setCurrentUserID] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [removingParent, setRemovingParent] = useState<Record<number, boolean>>({})

  const loadLinkedParents = () => {
    if (isAdmin && !isNew && id) {
      api.get(`/admin/members/${id}/parents`).then(r => setLinkedParents(r.data ?? []))
    }
  }

  useEffect(() => {
    if (isAdmin) api.get('/admin/users').then(r => setUsers(r.data.items ?? []))
    if (!isNew && id) {
      api.get(`/members/${id}`).then(r => {
        const m: Member = r.data
        setForm({
          first_name: m.first_name, last_name: m.last_name,
          date_of_birth: m.date_of_birth ?? '', member_number: m.member_number ?? '',
          pass_number: m.pass_number ?? '',
          jersey_number: m.jersey_number, position: m.position ?? '', status: m.status,
        })
        setCurrentUserID(m.user_id ?? null)
        setSelectedLinkedUser(m.user_id ? String(m.user_id) : '')
      })
      loadLinkedParents()
    }
  }, [id, isNew, isAdmin])

  const handleSave = async () => {
    setSaving(true); setError('')
    try {
      const body = { ...form, jersey_number: form.jersey_number ? Number(form.jersey_number) : null }
      if (isNew) {
        const r = await api.post('/members', body)
        navigate(`/mitglieder/${r.data.id}`, { replace: true })
      } else {
        await api.put(`/members/${id}`, body)
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      }
    } catch {
      setError('Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  const handleStatusChange = async (status: string) => {
    setForm(f => ({ ...f, status }))
    if (!isNew && id) await api.put(`/members/${id}/status`, { status })
  }

  const handleFamilyLink = async () => {
    if (!selectedParentUser || !id) return
    try {
      await api.post('/admin/family-links', { parent_user_id: Number(selectedParentUser), member_id: Number(id) })
      setSelectedParentUser('')
      loadLinkedParents()
      setSaved(true); setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Verknüpfen.')
    }
  }

  const handleRemoveParent = async (parentUserId: number) => {
    if (!id) return
    setRemovingParent(prev => ({ ...prev, [parentUserId]: true }))
    try {
      await api.delete('/admin/family-links', { data: { parent_user_id: parentUserId, member_id: Number(id) } })
      loadLinkedParents()
    } catch {
      setError('Fehler beim Entfernen.')
    } finally {
      setRemovingParent(prev => ({ ...prev, [parentUserId]: false }))
    }
  }

  const handleLinkUser = async () => {
    if (!id) return
    await api.put(`/admin/members/${id}/user`, { user_id: selectedLinkedUser ? Number(selectedLinkedUser) : null })
    setCurrentUserID(selectedLinkedUser ? Number(selectedLinkedUser) : null)
    setSaved(true); setTimeout(() => setSaved(false), 2000)
  }

  const togglePosition = (pos: string) => {
    const current = form.position ? form.position.split(',').filter(Boolean) : []
    const next = current.includes(pos) ? current.filter(p => p !== pos) : [...current, pos]
    setForm(f => ({ ...f, position: next.join(',') }))
  }

  const selectedPositions = form.position ? form.position.split(',').filter(Boolean) : []
  const isPassive = form.status === 'passiv'
  const canAddParent = linkedParents.length < 2

  const field = (label: string, key: keyof typeof form, type = 'text') => (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">{label}</label>
      <input
        type={type}
        value={String(form[key] ?? '')}
        onChange={e => setForm(f => ({ ...f, [key]: e.target.value }))}
        className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
      />
    </div>
  )

  const disabledField = (label: string) => (
    <div>
      <label className="block text-sm font-medium text-gray-400 mb-1">{label} <span className="font-normal">(passiv)</span></label>
      <input disabled className="w-full border border-gray-200 rounded-md px-3 py-2 text-sm bg-gray-50 text-gray-400" />
    </div>
  )

  return (
    <div className="max-w-2xl">
      <div className="flex items-center gap-3 mb-6">
        <Link to="/mitglieder" className="text-sm text-gray-500 hover:text-gray-700">← Mitglieder</Link>
        <h1 className="text-2xl font-bold">{isNew ? 'Mitglied anlegen' : 'Mitglied bearbeiten'}</h1>
      </div>

      {/* Stammdaten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Stammdaten</h2>
        <div className="grid grid-cols-2 gap-4">
          {field('Vorname', 'first_name')}
          {field('Nachname', 'last_name')}
          {field('Geburtsdatum', 'date_of_birth', 'date')}
          {field('Mitgliedsnummer', 'member_number')}
          {isPassive ? disabledField('Passnummer') : field('Passnummer', 'pass_number')}
          {isPassive ? disabledField('Rückennummer') : field('Rückennummer', 'jersey_number', 'number')}
        </div>

        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">Positionen</label>
          {isPassive ? (
            <p className="text-sm text-gray-400 italic">Nicht zutreffend (passiv)</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {HANDBALL_POSITIONS.map(pos => (
                <button
                  key={pos}
                  type="button"
                  onClick={() => togglePosition(pos)}
                  className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                    selectedPositions.includes(pos)
                      ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                      : 'text-gray-600 border-gray-300 hover:border-brand-black hover:text-brand-black'
                  }`}
                >
                  {pos}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
          <div className="flex gap-2 flex-wrap">
            {STATUS_OPTIONS.map(s => (
              <button
                key={s}
                onClick={() => handleStatusChange(s)}
                className={`px-3 py-1 rounded-full text-sm border ${
                  form.status === s
                    ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                    : 'text-gray-600 border-gray-300 hover:border-brand-black hover:text-brand-black'
                }`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>

        {error && <p className="mt-3 text-sm text-brand-error">{error}</p>}
        <div className="mt-4 flex items-center gap-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-brand-success">Gespeichert</span>}
        </div>
      </div>

      {/* Nutzer verknüpfen (Admin only, existierendes Mitglied) */}
      {isAdmin && !isNew && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-1">Nutzer verknüpfen</h2>
          <p className="text-xs text-gray-500 mb-4">Das Mitglied erhält damit Zugang zu seinem eigenen Profil.</p>
          {currentUserID && (
            <p className="text-xs text-brand-black mb-3">
              Aktuell verknüpft: {users.find(u => u.id === currentUserID)?.name ?? `User #${currentUserID}`}
            </p>
          )}
          <div className="flex gap-3 items-end">
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">Nutzer</label>
              <select
                value={selectedLinkedUser}
                onChange={e => setSelectedLinkedUser(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                <option value="">– keine Verknüpfung –</option>
                {users.map(u => (
                  <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                ))}
              </select>
            </div>
            <button
              onClick={handleLinkUser}
              className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
            >
              Speichern
            </button>
          </div>
          {saved && <p className="mt-2 text-sm text-brand-success">Gespeichert</p>}
        </div>
      )}

      {/* Erziehungsberechtigte (Admin only, existierendes Mitglied) */}
      {isAdmin && !isNew && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-gray-700 mb-4">Erziehungsberechtigte</h2>

          {linkedParents.length > 0 && (
            <div className="mb-4 space-y-2">
              {linkedParents.map(p => (
                <div key={p.id} className="flex items-center justify-between border border-gray-100 rounded-lg px-4 py-2 text-sm">
                  <div>
                    <span className="font-medium">{p.name}</span>
                    <span className="ml-2 text-gray-400">{p.email}</span>
                  </div>
                  <button
                    onClick={() => handleRemoveParent(p.id)}
                    disabled={removingParent[p.id]}
                    className="text-xs text-gray-400 hover:text-red-600 transition-colors disabled:opacity-40 px-2 py-1 rounded"
                    title="Verknüpfung entfernen"
                  >
                    {removingParent[p.id] ? '…' : 'Entfernen'}
                  </button>
                </div>
              ))}
            </div>
          )}

          {canAddParent ? (
            <div className="flex gap-3 items-end">
              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-700 mb-1">Erziehungsberechtigten hinzufügen</label>
                <select
                  value={selectedParentUser}
                  onChange={e => setSelectedParentUser(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                >
                  <option value="">– auswählen –</option>
                  {users.filter(u => !linkedParents.some(p => p.id === u.id)).map(u => (
                    <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                  ))}
                </select>
              </div>
              <button
                onClick={handleFamilyLink}
                disabled={!selectedParentUser}
                className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
              >
                Hinzufügen
              </button>
            </div>
          ) : (
            <p className="text-xs text-gray-400 italic">Maximal zwei Erziehungsberechtigte möglich.</p>
          )}

          {saved && <p className="mt-2 text-sm text-brand-success">Gespeichert</p>}
        </div>
      )}
    </div>
  )
}
