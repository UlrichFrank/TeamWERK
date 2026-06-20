import { useState, FormEvent } from 'react'
import { Link } from 'react-router-dom'
import axios from 'axios'

const Sidebar = () => (
  <div className="hidden sm:flex flex-col justify-center items-center sm:w-56 shrink-0 px-8 py-12 text-brand-black">
    <img src="/logo.svg" alt="Team Stuttgart" className="h-20 w-20 mb-6" />
    <h1 className="text-2xl font-bold mb-1">TeamWERK</h1>
    <p className="text-brand-black/50 text-sm">Team Stuttgart</p>
  </div>
)

const MobileLogo = () => (
  <div className="sm:hidden flex flex-col items-center mb-8">
    <img src="/logo.svg" alt="Team Stuttgart" className="h-16 w-16 mb-4" />
    <h1 className="text-2xl font-bold mb-1">TeamWERK</h1>
    <p className="text-brand-black/50 text-sm">Team Stuttgart</p>
  </div>
)

export default function RequestMembershipPage() {
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [comment, setComment] = useState('')
  const [isChild, setIsChild] = useState(false)
  const [parentEmail, setParentEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const payload = isChild
        ? { first_name: firstName, last_name: lastName, is_child: true, parent_email: parentEmail, comment: comment || undefined }
        : { first_name: firstName, last_name: lastName, email, comment: comment || undefined }
      await axios.post('/api/auth/request-membership', payload)
      setSent(true)
    } catch {
      setError('Fehler beim Senden. Bitte versuche es erneut.')
    }
  }

  if (sent) {
    return (
      <div className="min-h-screen flex flex-col sm:flex-row bg-brand-gray">
        <Sidebar />
        <div className="flex-1 flex items-center justify-center bg-brand-white sm:rounded-l-3xl sm:border-l-4 sm:border-brand-yellow">
          <div className="w-full max-w-sm px-4 sm:px-8 py-8 sm:py-0">
            <MobileLogo />
            <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
              <h2 className="text-xl font-bold mb-2">Antrag gesendet!</h2>
              <p className="text-sm text-brand-text-muted">
                Dein Antrag wurde weitergeleitet. Du erhältst eine E-Mail sobald er bearbeitet wurde.
              </p>
              <div className="mt-6">
                <Link to="/login" className="text-sm text-brand-black hover:text-brand-yellow transition-colors">
                  Zurück zur Anmeldung
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex flex-col sm:flex-row bg-brand-gray">
      <Sidebar />
      <div className="flex-1 flex items-center justify-center bg-brand-white sm:rounded-l-3xl sm:border-l-4 sm:border-brand-yellow">
        <div className="w-full max-w-sm px-4 sm:px-8 py-8 sm:py-0">
          <MobileLogo />
          <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8">
            <h2 className="text-2xl font-bold mb-1">Beitrittsantrag</h2>
            <p className="text-sm text-brand-text-muted mb-6">Team Stuttgart – TeamWERK</p>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && <p className="text-brand-danger text-sm">{error}</p>}
              <label className="flex items-start gap-2 cursor-pointer">
                <input
                  type="checkbox" checked={isChild} onChange={e => setIsChild(e.target.checked)}
                  className="mt-0.5 h-4 w-4 accent-brand-yellow"
                />
                <span className="text-sm text-brand-text">
                  Kinderaccount anlegen
                  <span className="block text-brand-text-muted">Für ein Kind ohne eigene E-Mail-Adresse</span>
                </span>
              </label>
              <div>
                <label className="block text-sm font-medium text-brand-black mb-1">
                  {isChild ? 'Vorname des Kindes' : 'Vorname'}
                </label>
                <input
                  type="text" value={firstName} onChange={e => setFirstName(e.target.value)} required
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-black mb-1">
                  {isChild ? 'Nachname des Kindes' : 'Nachname'}
                </label>
                <input
                  type="text" value={lastName} onChange={e => setLastName(e.target.value)} required={isChild}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              {isChild ? (
                <div>
                  <label className="block text-sm font-medium text-brand-black mb-1">E-Mail der Eltern</label>
                  <input
                    type="email" value={parentEmail} onChange={e => setParentEmail(e.target.value)} required
                    placeholder="verwaltendes Elternteil"
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                  <p className="mt-1 text-xs text-brand-text-muted">
                    Der Login-Name (Vorname.Nachname) und der Link zum Passwort-Setzen gehen an diese Adresse.
                  </p>
                </div>
              ) : (
                <div>
                  <label className="block text-sm font-medium text-brand-black mb-1">E-Mail</label>
                  <input
                    type="email" value={email} onChange={e => setEmail(e.target.value)} required
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-brand-black mb-1">
                  Kommentar <span className="text-brand-text-subtle font-normal">(optional)</span>
                </label>
                <input
                  type="text" value={comment} onChange={e => setComment(e.target.value)}
                  placeholder="z.B. Mannschaft, Ansprechpartner …"
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <button
                type="submit"
                className="w-full bg-brand-yellow text-brand-black rounded-md py-2.5 sm:py-2 text-sm font-semibold hover:bg-brand-black hover:text-brand-yellow transition-colors"
              >
                Antrag absenden
              </button>
            </form>
            <div className="mt-4 text-center text-sm">
              <Link to="/login" className="text-brand-black hover:text-brand-yellow transition-colors">
                Zurück zur Anmeldung
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
