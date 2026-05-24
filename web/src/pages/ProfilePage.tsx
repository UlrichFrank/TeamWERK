import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import ProfileAccountTab from '../components/profile/ProfileAccountTab'
import ProfileProfilTab from '../components/profile/ProfileProfilTab'
import ProfileMemberTab from '../components/profile/ProfileMemberTab'
import ProfileMiscTab from '../components/profile/ProfileMiscTab'

export interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; pass_number: string
  jersey_number?: number; position: string; status: string
  iban?: string
  account_holder?: string
}

export interface Parent {
  id: number; name: string; email: string
}

export interface Phone {
  id: number; label: string; number: string; sort_order: number
}

export interface Visibility {
  phones_visible: boolean; address_visible: boolean; photo_visible: boolean
}

export interface ChangeDraft {
  id: number
  field_name: string
  old_value: any
  new_value: any
  created_at: string
}

type TabName = 'account' | 'profile' | 'member' | 'misc'

export default function ProfilePage() {
  const { user, logout } = useAuth()
  const [activeTab, setActiveTab] = useState<TabName>(() => {
    const saved = localStorage.getItem('profileTab')
    return (saved as TabName) || 'account'
  })

  const [ownMember, setOwnMember] = useState<Member | null>(null)
  const [children, setChildren] = useState<Member[]>([])
  const [parents, setParents] = useState<Parent[]>([])
  const [draftRefreshKey, setDraftRefreshKey] = useState(0)

  useEffect(() => {
    localStorage.setItem('profileTab', activeTab)
  }, [activeTab])

  useEffect(() => {
    api.get('/profile/me').then(r => {
      setOwnMember(r.data?.own_member ?? null)
      setChildren(r.data?.children ?? [])
      setParents(r.data?.parents ?? [])
    })
  }, [])

  const showMemberTab = ownMember !== null

  const handleDraftWithdrawn = () => {
    setDraftRefreshKey(k => k + 1)
  }

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-bold mb-6">Mein Profil</h1>

      {/* Tab Navigation */}
      <div className="flex gap-1 mb-6 border-b border-brand-border-subtle sm:flex-wrap">
        {(['account', 'profile', ...(showMemberTab ? ['member'] : []), 'misc'] as TabName[]).map(tab => {
          const labels: Record<TabName, string> = { account: 'Konto', profile: 'Profil', member: 'Mitgliedsdaten', misc: 'Sonstiges' }
          return (
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
          )
        })}
      </div>

      {/* Tab Content */}
      {activeTab === 'account' && <ProfileAccountTab user={user} logout={logout} />}
      {activeTab === 'profile' && (
        <ProfileProfilTab
          children={children}
          parents={parents}
          ownMember={ownMember}
          draftRefreshKey={draftRefreshKey}
        />
      )}
      {showMemberTab && activeTab === 'member' && (
        <ProfileMemberTab ownMember={ownMember} onDraftWithdrawn={handleDraftWithdrawn} />
      )}
      {activeTab === 'misc' && <ProfileMiscTab />}
    </div>
  )
}
