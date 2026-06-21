import { useEffect, useState, FormEvent } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Mail } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import ProfileProfilTab from '../components/profile/ProfileProfilTab'
import ProfileMemberTab from '../components/profile/ProfileMemberTab'
import ProfileBankTab from '../components/profile/ProfileBankTab'
import ProfileMiscTab from '../components/profile/ProfileMiscTab'
import ProfileDatenschutzTab from '../components/profile/ProfileDatenschutzTab'
import { Member, Parent, Phone } from './ProfilePage'

export interface UserContact {
  first_name: string
  last_name: string
  street: string
  zip: string
  city: string
  recovery_email: string
  phones: Phone[]
  visibility: {
    phones_visible: boolean
    address_visible: boolean
    photo_visible: boolean
    email_visible: boolean
    whatsapp_visible: boolean
  }
}

type TabName = 'profile' | 'member' | 'banking' | 'datenschutz' | 'misc'

const labels: Record<TabName, string> = {
  profile: 'Kontakt',
  member: 'Mitgliedsdaten',
  banking: 'Bankdaten',
  datenschutz: 'Datenschutz',
  misc: 'Sonstiges',
}

export default function ChildProfilePage() {
  const { memberId } = useParams<{ memberId: string }>()
  const navigate = useNavigate()
  const [member, setMember] = useState<Member | null>(null)
  const [parents, setParents] = useState<Parent[]>([])
  const [userContact, setUserContact] = useState<UserContact | null>(null)
  const [activeTab, setActiveTab] = useState<TabName>('profile')
  const [newRecoveryEmail, setNewRecoveryEmail] = useState('')
  const [recoverySaving, setRecoverySaving] = useState(false)
  const [recoverySent, setRecoverySent] = useState(false)
  const [recoveryError, setRecoveryError] = useState('')

  const submitRecoveryEmail = async (e: FormEvent) => {
    e.preventDefault()
    setRecoverySaving(true)
    setRecoveryError('')
    try {
      await api.post(`/profile/kind/${memberId}/recovery-email`, { new_email: newRecoveryEmail })
      setRecoverySent(true)
      setNewRecoveryEmail('')
    } catch (err) {
      setRecoveryError(
        (err as { response?: { status?: number } })?.response?.status === 409
          ? 'Keine bisherige Adresse hinterlegt — bitte den Vorstand kontaktieren.'
          : 'Änderung fehlgeschlagen. Bitte E-Mail-Adresse prüfen.'
      )
    } finally {
      setRecoverySaving(false)
    }
  }

  const load = () => {
    api.get(`/profile/kind/${memberId}`)
      .then(r => {
        setMember(r.data.member)
        setParents(r.data.parents ?? [])
        setUserContact(r.data.user_contact ?? null)
      })
      .catch(err => { if (err.response?.status === 403) navigate('/') })
  }

  // load kapselt memberId, soll nur bei dessen Änderung neu laufen
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [memberId])
  useLiveUpdates(event => { if (event === 'members') load() })

  if (!member) return null

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-bold mb-6">{member.first_name}</h1>

      <div className="flex gap-1 mb-6 border-b border-brand-border-subtle flex-wrap">
        {(['profile', 'member', 'banking', 'datenschutz', 'misc'] as TabName[]).map(tab => (
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
        <>
          <ProfileProfilTab
            mode="child"
            childMemberId={memberId}
            ownMember={member}
            userContact={userContact}
            children={[]}
            parents={parents}
          />
          <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 mt-6">
            <div className="flex items-center gap-2 mb-2">
              <Mail className="w-5 h-5 text-brand-text-muted" />
              <h2 className="text-lg font-semibold text-brand-text">Eltern-E-Mail (Passwort-Reset)</h2>
            </div>
            <p className="text-sm text-brand-text-muted mb-4">
              An diese Adresse gehen Passwort-Mails für den Account von {member.first_name}. Das Kind selbst kann sie nicht ändern.
            </p>
            <div className="mb-4 text-sm text-brand-text">
              Aktuell: <span className="font-medium">{userContact?.recovery_email || '— nicht hinterlegt —'}</span>
            </div>
            {recoverySent ? (
              <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                Bestätigung nötig: Wir haben einen Link an die <strong>bisherige</strong> Adresse gesendet. Nach deren
                Bestätigung geht ein zweiter Link an die <strong>neue</strong> Adresse. Erst danach wird die Änderung wirksam.
              </div>
            ) : (
              <form onSubmit={submitRecoveryEmail} className="space-y-3">
                <input
                  type="email" required value={newRecoveryEmail} onChange={e => setNewRecoveryEmail(e.target.value)}
                  placeholder="Neue Eltern-E-Mail"
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
                {recoveryError && (
                  <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{recoveryError}</div>
                )}
                <button
                  type="submit" disabled={recoverySaving}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Ändern
                </button>
              </form>
            )}
          </div>
        </>
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
      {activeTab === 'datenschutz' && (
        <ProfileDatenschutzTab ownMember={member} onUpdated={load} />
      )}
      {activeTab === 'misc' && (
        <ProfileMiscTab />
      )}
    </div>
  )
}
