import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useVault } from '../contexts/VaultContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { encryptBankData, decryptBankData } from '../lib/bankCrypto'
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
  home_club?: string
  home_club_id?: number | null
  home_club_name?: string
  street?: string
  zip?: string
  city?: string
  join_date?: string
  exit_date?: string
  iban?: string
  account_holder?: string
  bank_ciphertext?: string
  bank_dek_enc?: string
  photo_url?: string
  photo_visible?: boolean
  cross_team_visible?: boolean
  dsgvo_verarbeitung?: boolean
  dsgvo_verarbeitung_date?: string
  dsgvo_weitergabe?: boolean
  dsgvo_weitergabe_date?: string
  sepa_mandat?: boolean
  sepa_mandat_date?: string
  sepa_mandat_url?: string
  beitragsfrei?: boolean
  beitragsfrei_grund?: string
  zweitspielrecht?: boolean
  welcome_email_sent_at?: string
}

interface User { id: number; first_name: string; last_name: string; email: string; role: string }
interface PendingInvitation { id: number; email: string; member_id?: number | null }

type TabName = 'stammdaten' | 'kontakt' | 'datenschutz' | 'familie' | 'admin'

export default function MemberDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { loading: authLoading, hasCapability } = useAuth()
  const { privateKey } = useVault()
  const isNew = id === 'neu'
  const isAdmin = hasCapability('manage_members')

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
    home_club: '', home_club_id: null, street: '', zip: '', city: '', join_date: '', exit_date: '', iban: '', account_holder: '',
    photo_url: '', photo_visible: false,
    dsgvo_verarbeitung: false, dsgvo_verarbeitung_date: '',
    dsgvo_weitergabe: false, dsgvo_weitergabe_date: '',
    sepa_mandat: false, sepa_mandat_date: '', sepa_mandat_url: '',
    beitragsfrei: false, beitragsfrei_grund: '', zweitspielrecht: false,
  })
  const [users, setUsers] = useState<User[]>([])
  const [invitations, setInvitations] = useState<PendingInvitation[]>([])
  const [linkedParents, setLinkedParents] = useState<User[]>([])
  const [currentUserID, setCurrentUserID] = useState<number | null>(null)
  const [welcomeEmailSentAt, setWelcomeEmailSentAt] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  // Bankdaten-Envelope (Modell B): wird beim Laden gemerkt und nur bei entsperrtem
  // Tresor (privateKey) clientseitig entschlüsselt ins Formular übernommen.
  const [bankEnv, setBankEnv] = useState<{ bank_ciphertext: string; bank_dek_enc: string } | null>(null)
  type DraftValue = {
    verarbeitung?: boolean; weitergabe?: boolean
    account_holder?: string; iban?: string
    first_name?: string; last_name?: string
    street?: string; zip?: string; city?: string
    [k: string]: unknown
  } | null
  const [drafts, setDrafts] = useState<Array<{ id: number; field_name: string; old_value: DraftValue; new_value: DraftValue }>>([])

  const loadLinkedParents = () => {
    if (isAdmin && !isNew && id) {
      api.get(`/members/${id}/parents`).then(r => setLinkedParents(r.data ?? []))
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
      home_club: m.home_club ?? '',
      home_club_id: m.home_club_id ?? null,
      street: m.street ?? '', zip: m.zip ?? '', city: m.city ?? '',
      join_date: m.join_date?.slice(0, 10) ?? '',
      exit_date: m.exit_date?.slice(0, 10) ?? '',
      iban: '',
      account_holder: '',
      photo_url: m.photo_url ?? '',
      photo_visible: m.photo_visible ?? false,
      cross_team_visible: m.cross_team_visible ?? false,
      dsgvo_verarbeitung: m.dsgvo_verarbeitung ?? false,
      dsgvo_verarbeitung_date: m.dsgvo_verarbeitung_date?.slice(0, 10) ?? '',
      dsgvo_weitergabe: m.dsgvo_weitergabe ?? false,
      dsgvo_weitergabe_date: m.dsgvo_weitergabe_date?.slice(0, 10) ?? '',
      sepa_mandat: m.sepa_mandat ?? false,
      sepa_mandat_date: m.sepa_mandat_date?.slice(0, 10) ?? '',
      sepa_mandat_url: m.sepa_mandat_url ?? '',
      beitragsfrei: m.beitragsfrei ?? false,
      beitragsfrei_grund: m.beitragsfrei_grund ?? '',
      zweitspielrecht: m.zweitspielrecht ?? false,
    })
    setCurrentUserID(m.user_id ?? null)
    setWelcomeEmailSentAt(m.welcome_email_sent_at ?? null)
    setBankEnv(
      m.bank_ciphertext
        ? { bank_ciphertext: m.bank_ciphertext, bank_dek_enc: m.bank_dek_enc ?? '' }
        : null,
    )
  }

  // Bei entsperrtem Tresor den Bankdaten-Envelope clientseitig entschlüsseln und ins
  // Formular übernehmen (nur lesbar für vorstand/kassierer mit Tresor-Schlüssel).
  useEffect(() => {
    if (!privateKey || !bankEnv) return
    let cancelled = false
    decryptBankData(bankEnv, privateKey)
      .then(d => {
        if (!cancelled) setForm(prev => ({ ...prev, iban: d.iban, account_holder: d.account_holder }))
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [privateKey, bankEnv])

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
    if (isAdmin) {
      api.get('/users?limit=1000').then(r => { if (!cancelled) setUsers(r.data.items ?? []) })
      api.get('/invitations').then(r => { if (!cancelled) setInvitations(r.data ?? []) })
    }
    if (!isNew && id) {
      loadDrafts()
      api.get(`/members/${id}`).then(r => {
        if (cancelled) return
        applyMemberToForm(r.data)
      })
      loadLinkedParents()
    }
    return () => { cancelled = true }
    // loadDrafts/loadLinkedParents lesen nur id/isAdmin/isNew (bereits in den Deps); bewusst nicht als Deps, um Endlosschleifen durch neue Funktionsidentitäten pro Render zu vermeiden.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, isNew, isAdmin, authLoading])

  useLiveUpdates(event => {
    if (event === 'members' && !isNew && id) {
      api.get(`/members/${id}`).then(r => applyMemberToForm(r.data)).catch(() => {})
      loadLinkedParents()
      loadDrafts()
    }
  })

  const handleSave = async () => {
    setSaving(true); setError('')
    try {
      // Eintrittsdatum ist Pflicht; Austrittsdatum Pflicht bei Status 'ausgetreten'.
      if (!form.join_date) {
        setError('Eintrittsdatum ist erforderlich.')
        return
      }
      if (form.status === 'ausgetreten' && !form.exit_date) {
        setError('Austrittsdatum ist bei Status „ausgetreten" erforderlich.')
        return
      }
      // Bankdaten werden NICHT über den Member-Voll-Endpoint geschrieben (Zero-Knowledge):
      // sie laufen ausschließlich clientseitig verschlüsselt über den Bankdaten-Tab
      // (handleSaveBank → /bank-details). Hier herausnehmen, damit kein Klartext zum Server geht.
      const { iban: _iban, account_holder: _ah, ...rest } = form
      void _iban
      void _ah
      const body = {
        ...rest,
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
    } catch (err) {
      const ax = err as { response?: { status?: number; data?: unknown } }
      if (ax.response?.status === 409 && typeof ax.response.data === 'string') {
        setError(ax.response.data.trim())
      } else {
        setError('Fehler beim Speichern.')
      }
    } finally {
      setSaving(false)
    }
  }

  // Kassierer-Pfad: nur die Bankfelder über /bank-details speichern (Feld-Whitelist).
  // Es werden alle bankrelevanten Felder aus dem geladenen Formular gesendet, damit
  // nicht editierte Werte (z. B. Adresse) nicht überschrieben/genullt werden.
  const handleSaveBank = async () => {
    if (!id) return
    setSaving(true); setError('')
    try {
      // Bankdaten clientseitig an den öffentlichen Gruppen-Schlüssel verschlüsseln; ""
      // löscht den Envelope. Server sieht nie Klartext-IBAN.
      const hasBank = (form.iban ?? '') !== '' || (form.account_holder ?? '') !== ''
      const env = hasBank
        ? await encryptBankData({ iban: form.iban ?? '', account_holder: form.account_holder ?? '' })
        : { bank_ciphertext: '', bank_dek_enc: '' }
      await api.put(`/members/${id}/bank-details`, {
        bank_ciphertext: env.bank_ciphertext,
        bank_dek_enc: env.bank_dek_enc,
        sepa_mandat: !!form.sepa_mandat,
        sepa_mandat_date: form.sepa_mandat_date ?? '',
        street: form.street ?? '',
        zip: form.zip ?? '',
        city: form.city ?? '',
        beitragsfrei: !!form.beitragsfrei,
        beitragsfrei_grund: form.beitragsfrei ? (form.beitragsfrei_grund ?? '') : '',
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  const handleFamilyLink = async (parentUserId: number) => {
    if (!id) return
    try {
      await api.post('/family-links', { parent_user_id: parentUserId, member_id: Number(id) })
      loadLinkedParents()
      setSaved(true); setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Verknüpfen.')
    }
  }

  const handleRemoveParent = async (parentUserId: number) => {
    if (!id) return
    try {
      await api.delete('/family-links', { data: { parent_user_id: parentUserId, member_id: Number(id) } })
      loadLinkedParents()
    } catch {
      setError('Fehler beim Entfernen.')
    }
  }

  const handleLinkUser = async (userId: number | null) => {
    if (!id) return
    await api.put(`/members/${id}/user`, { user_id: userId })
    setCurrentUserID(userId)
    setSaved(true); setTimeout(() => setSaved(false), 2000)
  }

  const handleLinkInvitation = async (invitationId: number | null) => {
    if (!id) return
    const memberId = Number(id)
    const prev = invitations.find(i => i.member_id === memberId)
    if (prev && prev.id !== invitationId)
      await api.put(`/invitations/${prev.id}/member`, { member_id: null })
    if (invitationId !== null)
      await api.put(`/invitations/${invitationId}/member`, { member_id: memberId })
    const r = await api.get('/invitations')
    setInvitations(r.data ?? [])
    setSaved(true); setTimeout(() => setSaved(false), 2000)
  }

  const tabButtons: { name: TabName; label: string; show: boolean }[] = [
    // Stammdaten nur für Voll-Verwaltung (oder Neuanlage); Kassierer sieht nur Bankdaten.
    { name: 'stammdaten', label: 'Stammdaten', show: isNew || isAdmin },
    { name: 'kontakt', label: 'Bankdaten', show: !isNew },
    { name: 'datenschutz', label: 'Datenschutz', show: !isNew && isAdmin },
    { name: 'familie', label: 'Familie', show: !isNew && isAdmin },
    { name: 'admin', label: 'Admin', show: !isNew && isAdmin },
  ]

  // Falls der aktuell gewählte Tab für diese Rolle nicht sichtbar ist, auf den
  // ersten sichtbaren Tab zurückfallen (z. B. Kassierer → Bankdaten).
  useEffect(() => {
    const visible = tabButtons.filter(t => t.show).map(t => t.name)
    if (visible.length > 0 && !visible.includes(activeTab)) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
      setActiveTab(visible[0])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAdmin, isNew, activeTab])

  return (
    <div className="max-w-2xl">
      <div className="mb-6">
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
          memberId={isNew ? undefined : Number(id)}
          form={form}
          isNew={isNew}
          drafts={drafts}
          onFormChange={updates => setForm(f => ({ ...f, ...updates }))}
          onDraftAccept={handleDraftAccept}
          onDraftReject={handleDraftReject}
          onSave={handleSaveBank}
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
          memberId={id && !isNew ? Number(id) : undefined}
          memberUserId={currentUserID}
          users={users}
          linkedParents={linkedParents}
          onAddParent={handleFamilyLink}
          onRemoveParent={handleRemoveParent}
          onReload={() => { if (id) api.get(`/members/${id}`).then(r => applyMemberToForm(r.data)) }}
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
          invitations={invitations}
          currentUserId={currentUserID}
          welcomeEmailSentAt={welcomeEmailSentAt}
          onWelcomeEmailSent={sentAt => setWelcomeEmailSentAt(sentAt)}
          onLinkUser={handleLinkUser}
          onLinkInvitation={handleLinkInvitation}
          saving={saving}
          saved={saved}
          error={error}
        />
      )}
    </div>
  )
}
