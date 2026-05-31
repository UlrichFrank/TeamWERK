import { useEffect, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import MemberStammdatenTab from '../components/admin/MemberStammdatenTab'
import MemberKontaktTab from '../components/admin/MemberKontaktTab'
import MemberDatenschutzTab from '../components/admin/MemberDatenschutzTab'
import MemberFamilieTab from '../components/admin/MemberFamilieTab'
import MemberAdminTab from '../components/admin/MemberAdminTab'

interface Member {
  id: number
  first_name: string
  last_name: string
  date_of_birth: string
  member_number: string
  pass_number: string
  jersey_number?: number
  position: string
  gender: string
  status: string
  user_id?: number
  club_functions?: string[]
  street?: string
  zip?: string
  city?: string
  join_date?: string
  iban?: string
  account_holder?: string
  photo_url?: string
  photo_visible?: boolean
  dsgvo_verarbeitung?: boolean
  dsgvo_verarbeitung_date?: string
  dsgvo_weitergabe?: boolean
  dsgvo_weitergabe_date?: string
  sepa_mandat?: boolean
  sepa_mandat_date?: string
  sepa_mandat_url?: string
  welcome_email_sent_at?: string
}

interface User { id: number; name: string; email: string; role: string }

type TabName = 'stammdaten' | 'kontakt' | 'datenschutz' | 'familie' | 'admin'

export default function MemberDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user, loading: authLoading } = useAuth()
  const isNew = id === 'neu'
  const isAdmin = user?.role === 'admin'

  const [activeTab, setActiveTab] = useState<TabName>(() => {
    const saved = localStorage.getItem('memberDetailTab')
    return (saved as TabName) || 'stammdaten'
  })

  useEffect(() => {
    localStorage.setItem('memberDetailTab', activeTab)
  }, [activeTab])

  const [form, setForm] = useState<Omit<Member, 'id'>>({
    first_name: '', last_name: '', date_of_birth: '', member_number: '', pass_number: '',
    jersey_number: undefined, position: '', gender: 'u', status: 'aktiv', club_functions: [],
    street: '', zip: '', city: '', join_date: '', iban: '', account_holder: '',
    photo_url: '', photo_visible: false,
    dsgvo_verarbeitung: false, dsgvo_verarbeitung_date: '',
    dsgvo_weitergabe: false, dsgvo_weitergabe_date: '',
    sepa_mandat: false, sepa_mandat_date: '', sepa_mandat_url: '',
  })
  const [users, setUsers] = useState<User[]>([])
  const [linkedParents, setLinkedParents] = useState<User[]>([])
  const [currentUserID, setCurrentUserID] = useState<number | null>(null)
  const [welcomeEmailSentAt, setWelcomeEmailSentAt] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [drafts, setDrafts] = useState<Array<{ id: number; field_name: string; old_value: any; new_value: any }>>([])

  const loadLinkedParents = () => {
    if (isAdmin && !isNew && id) {
      api.get(`/admin/members/${id}/parents`).then(r => setLinkedParents(r.data ?? []))
    }
  }

  const loadDrafts = () => {
    if (!isNew && id) {
      api.get(`/members/${id}/change-drafts`).then(r => setDrafts(r.data?.drafts ?? []))
    }
  }

  const applyMemberToForm = (m: Member) => {
    setForm({
      first_name: m.first_name, last_name: m.last_name,
      date_of_birth: m.date_of_birth?.slice(0, 10) ?? '',
      member_number: m.member_number ?? '',
      pass_number: m.pass_number ?? '',
      jersey_number: m.jersey_number, position: m.position ?? '',
      gender: m.gender ?? 'u', status: m.status,
      club_functions: m.club_functions ?? [],
      street: m.street ?? '', zip: m.zip ?? '', city: m.city ?? '',
      join_date: m.join_date?.slice(0, 10) ?? '',
      iban: m.iban ?? '',
      account_holder: m.account_holder ?? '',
      photo_url: m.photo_url ?? '',
      photo_visible: m.photo_visible ?? false,
      dsgvo_verarbeitung: m.dsgvo_verarbeitung ?? false,
      dsgvo_verarbeitung_date: m.dsgvo_verarbeitung_date?.slice(0, 10) ?? '',
      dsgvo_weitergabe: m.dsgvo_weitergabe ?? false,
      dsgvo_weitergabe_date: m.dsgvo_weitergabe_date?.slice(0, 10) ?? '',
      sepa_mandat: m.sepa_mandat ?? false,
      sepa_mandat_date: m.sepa_mandat_date?.slice(0, 10) ?? '',
      sepa_mandat_url: m.sepa_mandat_url ?? '',
    })
    setCurrentUserID(m.user_id ?? null)
    setWelcomeEmailSentAt(m.welcome_email_sent_at ?? null)
  }

  const handleDraftAccept = async (draftId: number) => {
    if (!id) return
    try {
      await api.post(`/members/${id}/change-drafts/${draftId}/accept`)
      loadDrafts()
      api.get(`/members/${id}`).then(r => applyMemberToForm(r.data))
    } catch {
      setError('Fehler beim Annehmen der Änderung.')
    }
  }

  const handleDraftReject = async (draftId: number) => {
    if (!id) return
    try {
      await api.delete(`/members/${id}/change-drafts/${draftId}`)
      loadDrafts()
    } catch {
      setError('Fehler beim Ablehnen der Änderung.')
    }
  }

  useEffect(() => {
    if (authLoading) return
    let cancelled = false
    if (isAdmin) api.get('/admin/users').then(r => { if (!cancelled) setUsers(r.data.items ?? []) })
    if (!isNew && id) {
      loadDrafts()
      api.get(`/members/${id}`).then(r => {
        if (cancelled) return
        applyMemberToForm(r.data)
      })
      loadLinkedParents()
    }
    return () => { cancelled = true }
  }, [id, isNew, isAdmin, authLoading])

  const handleSave = async () => {
    setSaving(true); setError('')
    try {
      const body = {
        ...form,
        jersey_number: form.jersey_number ? Number(form.jersey_number) : null,
        club_functions: form.club_functions ?? [],
      }
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

  const handleFamilyLink = async (parentUserId: number) => {
    if (!id) return
    try {
      await api.post('/admin/family-links', { parent_user_id: parentUserId, member_id: Number(id) })
      loadLinkedParents()
      setSaved(true); setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Verknüpfen.')
    }
  }

  const handleRemoveParent = async (parentUserId: number) => {
    if (!id) return
    try {
      await api.delete('/admin/family-links', { data: { parent_user_id: parentUserId, member_id: Number(id) } })
      loadLinkedParents()
    } catch {
      setError('Fehler beim Entfernen.')
    }
  }

  const handleLinkUser = async (userId: number | null) => {
    if (!id) return
    await api.put(`/admin/members/${id}/user`, { user_id: userId })
    setCurrentUserID(userId)
    setSaved(true); setTimeout(() => setSaved(false), 2000)
  }

  const tabButtons: { name: TabName; label: string; show: boolean }[] = [
    { name: 'stammdaten', label: 'Stammdaten', show: true },
    { name: 'kontakt', label: 'Bankdaten', show: !isNew },
    { name: 'datenschutz', label: 'Datenschutz', show: !isNew && isAdmin },
    { name: 'familie', label: 'Familie', show: !isNew && isAdmin },
    { name: 'admin', label: 'Admin', show: !isNew && isAdmin },
  ]

  return (
    <div className="max-w-2xl">
      <div className="flex items-center gap-3 mb-6">
        <Link to="/mitglieder" className="text-sm text-brand-text-muted hover:text-brand-text">← Mitglieder</Link>
        <h1 className="text-2xl font-bold">{isNew ? 'Mitglied anlegen' : 'Mitglied bearbeiten'}</h1>
      </div>

      {/* Tab Navigation */}
      {!isNew && (
        <div className="flex gap-2 mb-6 border-b border-brand-border-subtle overflow-x-auto">
          {tabButtons.filter(t => t.show).map(tab => (
            <button
              key={tab.name}
              onClick={() => setActiveTab(tab.name)}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.name
                  ? 'border-brand-yellow text-brand-text'
                  : 'border-transparent text-brand-text-muted hover:text-brand-text'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      )}

      {/* Tab Content */}
      {activeTab === 'stammdaten' && (
        <MemberStammdatenTab
          form={form}
          memberId={isNew ? undefined : Number(id)}
          isNew={isNew}
          drafts={drafts}
          onFormChange={updates => setForm(f => ({ ...f, ...updates }))}
          onDraftAccept={handleDraftAccept}
          onDraftReject={handleDraftReject}
          onSave={handleSave}
          saving={saving}
          saved={saved}
          error={error}
        />
      )}

      {activeTab === 'kontakt' && (
        <MemberKontaktTab
          form={form}
          isNew={isNew}
          drafts={drafts}
          onFormChange={updates => setForm(f => ({ ...f, ...updates }))}
          onDraftAccept={handleDraftAccept}
          onDraftReject={handleDraftReject}
          onSave={handleSave}
          saving={saving}
          saved={saved}
          error={error}
        />
      )}

      {activeTab === 'datenschutz' && (
        <MemberDatenschutzTab
          form={form}
          isNew={isNew}
          drafts={drafts}
          onFormChange={updates => setForm(f => ({ ...f, ...updates }))}
          onDraftAccept={handleDraftAccept}
          onDraftReject={handleDraftReject}
          onSave={handleSave}
          saving={saving}
          saved={saved}
          error={error}
        />
      )}

      {activeTab === 'familie' && !isNew && (
        <MemberFamilieTab
          isNew={isNew}
          users={users}
          linkedParents={linkedParents}
          onAddParent={handleFamilyLink}
          onRemoveParent={handleRemoveParent}
          saving={saving}
          saved={saved}
          error={error}
        />
      )}

      {activeTab === 'admin' && (
        <MemberAdminTab
          isNew={isNew}
          memberId={id && !isNew ? Number(id) : undefined}
          users={users}
          currentUserId={currentUserID}
          welcomeEmailSentAt={welcomeEmailSentAt}
          onWelcomeEmailSent={sentAt => setWelcomeEmailSentAt(sentAt)}
          onLinkUser={handleLinkUser}
          saving={saving}
          saved={saved}
          error={error}
        />
      )}
    </div>
  )
}
