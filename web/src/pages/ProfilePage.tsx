import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import ProfileAccountTab from '../components/profile/ProfileAccountTab'
import ProfileProfilTab from '../components/profile/ProfileProfilTab'
import ProfileMemberTab from '../components/profile/ProfileMemberTab'
import ProfileBankTab from '../components/profile/ProfileBankTab'
import ProfileMiscTab from '../components/profile/ProfileMiscTab'
import ProfileKalenderTab from '../components/profile/ProfileKalenderTab'
import ProfileDatenschutzTab from '../components/profile/ProfileDatenschutzTab'
import { ProfilAnwesenheitContent } from './ProfilAnwesenheitPage'

export interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; pass_number: string
  jersey_number?: number; position: string; status: string
  street?: string; zip?: string; city?: string
  iban?: string
  account_holder?: string
  club_functions?: string[]
  photo_url?: string
  photo_visible?: boolean
  phones_visible?: boolean
  address_visible?: boolean
  email_visible?: boolean
  cross_team_visible?: boolean
  dsgvo_verarbeitung?: boolean
  dsgvo_verarbeitung_date?: string
  dsgvo_weitergabe?: boolean
  dsgvo_weitergabe_date?: string
  foto_veroeffentlichung?: boolean
  foto_veroeffentlichung_date?: string
  has_bank_data?: boolean
  sepa_mandat?: boolean
  sepa_mandat_date?: string
}

export interface Parent {
  id: number; name: string; email: string
}

export interface Phone {
  id: number; label: string; number: string; sort_order: number
}

export interface Visibility {
  phones_visible: boolean; address_visible: boolean; photo_visible: boolean; email_visible: boolean; whatsapp_visible: boolean
}

export interface ChangeDraft {
  id: number
  field_name: string
  old_value: { iban?: string; account_holder?: string; [k: string]: string | number | undefined } | null
  new_value: { iban?: string; account_holder?: string; [k: string]: string | number | undefined } | null
  created_at: string
}

type TabName = 'account' | 'profile' | 'member' | 'banking' | 'anwesenheit' | 'kalender' | 'datenschutz' | 'misc'

export default function ProfilePage() {
  const { user, logout } = useAuth()
  const [activeTab, setActiveTab] = useState<TabName>(() => {
    const saved = localStorage.getItem('profileTab')
    return (saved as TabName) || 'account'
  })

  const [ownMember, setOwnMember] = useState<Member | null>(null)
  const [children, setChildren] = useState<Member[]>([])
  const [parents, setParents] = useState<Parent[]>([])
  const [recoveryEmail, setRecoveryEmail] = useState('')
  const [draftRefreshKey, setDraftRefreshKey] = useState(0)

  useEffect(() => {
    localStorage.setItem('profileTab', activeTab)
  }, [activeTab])

  const loadProfile = () => {
    api.get('/profile/me').then(r => {
      setOwnMember(r.data?.own_member ?? null)
      setChildren(r.data?.children ?? [])
      setParents(r.data?.parents ?? [])
      setRecoveryEmail(r.data?.recovery_email ?? '')
    })
  }

  useEffect(() => { loadProfile() }, [])

  useLiveUpdates((event) => { if (event === 'members') loadProfile() })

  const hasPlayerFunction = (m: { club_functions?: string[] } | null | undefined) =>
    !!m?.club_functions?.includes('spieler')

  const showMemberTabs = ownMember !== null
  const showAttendanceTab = hasPlayerFunction(ownMember) || children.some(hasPlayerFunction)

  const handleDraftWithdrawn = () => {
    setDraftRefreshKey(k => k + 1)
  }

  const tabs: TabName[] = [
    'account',
    'profile',
    ...(showMemberTabs ? (['member', 'banking'] as TabName[]) : []),
    ...(showAttendanceTab ? (['anwesenheit'] as TabName[]) : []),
    'kalender',
    ...(showMemberTabs ? (['datenschutz'] as TabName[]) : []),
    'misc',
  ]

  const labels: Record<TabName, string> = {
    account: 'Konto',
    profile: 'Kontakt',
    member: 'Mitgliedsdaten',
    banking: 'Bankdaten',
    anwesenheit: 'Anwesenheit',
    kalender: 'Kalender-Abo',
    datenschutz: 'Datenschutz',
    misc: 'Sonstiges',
  }

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-bold mb-6">Mein Profil</h1>

      {/* Tab Navigation */}
      <div className="flex gap-1 mb-6 border-b border-brand-border-subtle flex-wrap">
        {tabs.map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab
                ? 'border-brand-yellow text-brand-text'
                : 'border-transparent text-brand-text-muted hover:text-brand-text'
            }`}
          >
            {labels[tab]}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === 'account' && <ProfileAccountTab user={user} logout={logout} recoveryEmail={recoveryEmail} />}
      {activeTab === 'profile' && (
        <ProfileProfilTab
          children={children}
          parents={parents}
          ownMember={ownMember}
          draftRefreshKey={draftRefreshKey}
        />
      )}
      {showMemberTabs && activeTab === 'member' && (
        <ProfileMemberTab ownMember={ownMember} children={children} parents={parents} onDraftWithdrawn={handleDraftWithdrawn} />
      )}
      {showMemberTabs && activeTab === 'banking' && (
        <ProfileBankTab ownMember={ownMember} />
      )}
      {showAttendanceTab && activeTab === 'anwesenheit' && <ProfilAnwesenheitContent />}
      {activeTab === 'kalender' && <ProfileKalenderTab />}
      {showMemberTabs && activeTab === 'datenschutz' && ownMember && (
        <ProfileDatenschutzTab ownMember={ownMember} onUpdated={loadProfile} />
      )}
      {activeTab === 'misc' && <ProfileMiscTab />}
    </div>
  )
}
