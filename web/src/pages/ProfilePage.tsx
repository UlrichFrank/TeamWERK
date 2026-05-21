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

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-bold mb-6">Mein Profil</h1>

      {/* Tab Navigation */}
      <div className="flex gap-1 mb-6 border-b border-gray-200 sm:flex-wrap">
        <button
          onClick={() => setActiveTab('account')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'account'
              ? 'border-brand-yellow text-brand-black'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          Konto
        </button>
        <button
          onClick={() => setActiveTab('profile')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'profile'
              ? 'border-brand-yellow text-brand-black'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          Profil
        </button>
        {showMemberTab && (
          <button
            onClick={() => setActiveTab('member')}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'member'
                ? 'border-brand-yellow text-brand-black'
                : 'border-transparent text-gray-600 hover:text-gray-900'
            }`}
          >
            Mitgliedsdaten
          </button>
        )}
        <button
          onClick={() => setActiveTab('misc')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'misc'
              ? 'border-brand-yellow text-brand-black'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          Sonstiges
        </button>
      </div>

      {/* Tab Content */}
      {activeTab === 'account' && <ProfileAccountTab user={user} logout={logout} />}
      {activeTab === 'profile' && <ProfileProfilTab children={children} parents={parents} />}
      {showMemberTab && activeTab === 'member' && <ProfileMemberTab ownMember={ownMember} />}
      {activeTab === 'misc' && <ProfileMiscTab />}
    </div>
  )
}
