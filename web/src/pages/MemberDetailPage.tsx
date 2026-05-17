import { useEffect, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; member_number: string; pass_number: string
  jersey_number?: number; position: string; status: string; user_id?: number
}

interface Team { id: number; name: string }
interface Season { id: number; name: string; is_active: number }
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
  const [teams, setTeams] = useState<Team[]>([])
  const [seasons, setSeasons] = useState<Season[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [selectedTeam, setSelectedTeam] = useState('')
  const [selectedSeason, setSelectedSeason] = useState('')
  const [isPrimary, setIsPrimary] = useState(false)
  const [selectedParentUser, setSelectedParentUser] = useState('')
  const [selectedLinkedUser, setSelectedLinkedUser] = useState('')
  const [currentUserID, setCurrentUserID] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.get('/admin/teams').then(r => setTeams(r.data ?? []))
    api.get('/admin/seasons').then(r => {
      const s: Season[] = r.data ?? []
      setSeasons(s)
      const active = s.find(x => x.is_active)
      if (active) setSelectedSeason(String(active.id))
    })
    if (isAdmin) api.get('/admin/users').then(r => setUsers(r.data ?? []))
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

  const handleAssignTeam = async () => {
    if (!selectedTeam || !selectedSeason || !id) return
    await api.post(`/members/${id}/team-assignment`, {
      team_id: Number(selectedTeam), season_id: Number(selectedSeason), is_primary: isPrimary,
    })
    setSaved(true); setTimeout(() => setSaved(false), 2000)
  }

  const handleFamilyLink = async () => {
    if (!selectedParentUser || !id) return
    await api.post('/admin/family-links', { parent_user_id: Number(selectedParentUser), member_id: Number(id) })
    setSaved(true); setTimeout(() => setSaved(false), 2000)
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
      <div className="bg-white rounded-xl shadow p-6 mb-4">
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
                      ? 'bg-brand-blue text-white border-brand-blue'
                      : 'text-gray-600 border-gray-300 hover:border-brand-blue'
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
                    ? 'bg-brand-blue text-white border-brand-blue'
                    : 'text-gray-600 border-gray-300 hover:border-brand-blue'
                }`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
        <div className="mt-4 flex items-center gap-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="bg-brand-blue text-white px-4 py-2 rounded-md text-sm hover:bg-brand-blue-dark disabled:opacity-50"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-green-600">Gespeichert</span>}
        </div>
      </div>

      {/* Team-Zuordnung (nur bei existierendem, nicht-passivem Mitglied) */}
      {!isNew && !isPassive && (
        <div className="bg-white rounded-xl shadow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-4">Mannschaft zuweisen</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Mannschaft</label>
              <select
                value={selectedTeam}
                onChange={e => setSelectedTeam(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                <option value="">– auswählen –</option>
                {teams.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Saison</label>
              <select
                value={selectedSeason}
                onChange={e => setSelectedSeason(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                {seasons.map(s => <option key={s.id} value={s.id}>{s.name}{s.is_active ? ' (aktiv)' : ''}</option>)}
              </select>
            </div>
          </div>
          <label className="flex items-center gap-2 mt-3 text-sm text-gray-700">
            <input type="checkbox" checked={isPrimary} onChange={e => setIsPrimary(e.target.checked)} />
            Primärmannschaft
          </label>
          <button
            onClick={handleAssignTeam}
            className="mt-3 bg-brand-blue text-white px-4 py-2 rounded-md text-sm hover:bg-brand-blue-dark"
          >
            Zuweisen
          </button>
        </div>
      )}

      {/* Nutzer verknüpfen (Admin only, existierendes Mitglied) */}
      {isAdmin && !isNew && (
        <div className="bg-white rounded-xl shadow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-1">Nutzer verknüpfen</h2>
          <p className="text-xs text-gray-500 mb-4">Das Mitglied erhält damit Zugang zu seinem eigenen Profil.</p>
          {currentUserID && (
            <p className="text-xs text-brand-green mb-3">
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
              className="bg-brand-blue text-white px-4 py-2 rounded-md text-sm hover:bg-brand-blue-dark"
            >
              Speichern
            </button>
          </div>
          {saved && <p className="mt-2 text-sm text-green-600">Gespeichert</p>}
        </div>
      )}

      {/* Familien-Verlinkung (Admin only, existierendes Mitglied) */}
      {isAdmin && !isNew && (
        <div className="bg-white rounded-xl shadow p-6">
          <h2 className="font-semibold text-gray-700 mb-4">Elternteil verknüpfen</h2>
          <div className="flex gap-3 items-end">
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">Elternteil (Nutzer)</label>
              <select
                value={selectedParentUser}
                onChange={e => setSelectedParentUser(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                <option value="">– auswählen –</option>
                {users.filter(u => u.role === 'elternteil').map(u => (
                  <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                ))}
              </select>
            </div>
            <button
              onClick={handleFamilyLink}
              className="bg-brand-blue text-white px-4 py-2 rounded-md text-sm hover:bg-brand-blue-dark"
            >
              Verknüpfen
            </button>
          </div>
          {saved && <p className="mt-2 text-sm text-green-600">Gespeichert</p>}
        </div>
      )}
    </div>
  )
}
