import { useEffect, useState, FormEvent } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Mail } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import ProfileProfilTab from '../components/profile/ProfileProfilTab'
import ProfileMemberTab from '../components/profile/ProfileMemberTab'
import ProfileBankTab from '../components/profile/ProfileBankTab'
import ProfileKalenderTab from '../components/profile/ProfileKalenderTab'
import ProfileMiscTab from '../components/profile/ProfileMiscTab'
import ProfileDatenschutzTab from '../components/profile/ProfileDatenschutzTab'
import { ProfilAnwesenheitContent } from './ProfilAnwesenheitPage'
import { ProfilTrainingstagebuchContent } from './ProfilTrainingstagebuchPage'
import { Member, Parent, Phone } from './ProfilePage'
import { BTN_PRIMARY, INPUT, TAB, TAB_ACTIVE, TAB_BAR, TAB_INACTIVE } from '../lib/buttonStyles'
import { PAGE_TITLE, SECTION_TITLE } from '../lib/typography'

export interface UserContact {
  first_name: string
  last_name: string
  street: string
  zip: string
  city: string
  date_of_birth: string
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

type TabName = 'profile' | 'member' | 'banking' | 'anwesenheit' | 'tagebuch' | 'kalender' | 'datenschutz' | 'misc'

const labels: Record<TabName, string> = {
  profile: 'Kontakt',
  member: 'Mitgliedsdaten',
  banking: 'Bankdaten',
  anwesenheit: 'Anwesenheit',
  tagebuch: 'Trainingstagebuch',
  kalender: 'Kalender-Abo',
  datenschutz: 'Datenschutz',
  misc: 'Sonstiges',
}

export default function ChildProfilePage() {
  const { memberId } = useParams<{ memberId: string }>()
  const navigate = useNavigate()
  const [member, setMember] = useState<Member | null>(null)
  const [parents, setParents] = useState<Parent[]>([])
  const [userContact, setUserContact] = useState<UserContact | null>(null)
  const [, setHasCalendarToken] = useState(false)
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
        setHasCalendarToken(!!r.data.calendar_token)
      })
      .catch(err => { if (err.response?.status === 403) navigate('/') })
  }

  // load kapselt memberId, soll nur bei dessen Änderung neu laufen
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [memberId])
  useLiveUpdates(event => { if (event === 'members') load() })

  if (!member) return null

  // Anwesenheit-Tab nur für Spieler-Kinder (Vereinsfunktion `spieler`).
  const isPlayer = !!member.club_functions?.includes('spieler')
  const tabs: TabName[] = [
    'profile',
    'member',
    'banking',
    ...(isPlayer ? (['anwesenheit', 'tagebuch'] as TabName[]) : []),
    'kalender',
    'datenschutz',
    'misc',
  ]

  return (
    <div className="max-w-4xl">
      <h1 className={`${PAGE_TITLE} mb-6`}>{member.first_name}</h1>

      <div className={`${TAB_BAR} mb-6`}>
        {tabs.map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`${TAB} ${activeTab === tab ? TAB_ACTIVE : TAB_INACTIVE}`}
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
          <div className="bg-brand-surface-card rounded-xs shadow-sm border-t-2 eckfahne border-brand-yellow transform-gpu p-6 mt-6">
            <div className="flex items-center gap-2 mb-2">
              <Mail className="w-5 h-5 text-brand-text-muted" />
              <h2 className={SECTION_TITLE}>Eltern-E-Mail (Passwort-Reset)</h2>
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
                  className={INPUT}
                />
                {recoveryError && (
                  <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{recoveryError}</div>
                )}
                <button
                  type="submit" disabled={recoverySaving}
                  className={BTN_PRIMARY}
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
      {isPlayer && activeTab === 'anwesenheit' && (
        <ProfilAnwesenheitContent forcedMemberId={member.id} />
      )}
      {/* Eltern lesen das Tagebuch ihres Kindes, dürfen es aber nicht
          befüllen — die Erfassung ist die Selbstauskunft des Spielers. */}
      {isPlayer && activeTab === 'tagebuch' && (
        <ProfilTrainingstagebuchContent forcedMemberId={member.id} />
      )}
      {activeTab === 'kalender' && (
        <ProfileKalenderTab apiPath={`/profile/kind/${memberId}/calendar-token`} />
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
