import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import ProfileProfilTab from '../components/profile/ProfileProfilTab'
import ProfileMemberTab from '../components/profile/ProfileMemberTab'
import ProfileBankTab from '../components/profile/ProfileBankTab'
import ProfileMiscTab from '../components/profile/ProfileMiscTab'
import { Member, Parent, Phone } from './ProfilePage'

export interface UserContact {
  first_name: string
  last_name: string
  street: string
  zip: string
  city: string
  phones: Phone[]
  visibility: {
    phones_visible: boolean
    address_visible: boolean
    photo_visible: boolean
    email_visible: boolean
    whatsapp_visible: boolean
  }
}

type TabName = 'profile' | 'member' | 'banking' | 'misc'

const labels: Record<TabName, string> = {
  profile: 'Kontakt',
  member: 'Mitgliedsdaten',
  banking: 'Bankdaten',
  misc: 'Sonstiges',
}

export default function ChildProfilePage() {
  const { memberId } = useParams<{ memberId: string }>()
  const navigate = useNavigate()
  const [member, setMember] = useState<Member | null>(null)
  const [parents, setParents] = useState<Parent[]>([])
  const [userContact, setUserContact] = useState<UserContact | null>(null)
  const [activeTab, setActiveTab] = useState<TabName>('profile')

  const load = () => {
    api.get(`/profile/kind/${memberId}`)
      .then(r => {
        setMember(r.data.member)
        setParents(r.data.parents ?? [])
        setUserContact(r.data.user_contact ?? null)
      })
      .catch(err => { if (err.response?.status === 403) navigate('/') })
  }

  useEffect(() => { load() }, [memberId])
  useLiveUpdates(event => { if (event === 'members') load() })

  if (!member) return null

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-bold mb-6">{member.first_name}</h1>

      <div className="flex gap-1 mb-6 border-b border-brand-border-subtle flex-wrap">
        {(['profile', 'member', 'banking', 'misc'] as TabName[]).map(tab => (
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

      {activeTab === 'profile' && (
        <ProfileProfilTab
          mode="child"
          childMemberId={memberId}
          ownMember={member}
          userContact={userContact}
          children={[]}
          parents={parents}
        />
      )}
      {activeTab === 'member' && (
        <ProfileMemberTab
          ownMember={member}
          parents={parents}
        />
      )}
      {activeTab === 'banking' && (
        <ProfileBankTab ownMember={member} />
      )}
      {activeTab === 'misc' && (
        <ProfileMiscTab />
      )}
    </div>
  )
}
