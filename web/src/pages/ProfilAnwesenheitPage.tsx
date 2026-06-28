import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../lib/api'
import AttendanceStatsView from '../components/AttendanceStatsView'

interface MemberRef { id: number; first_name: string; last_name: string }

// Auswahl- und Lade-Logik für die Spieler-/Eltern-Anwesenheit. Wird sowohl von
// der eigenständigen Seite (/profil/anwesenheit) als auch als Profil-Tab genutzt.
// forcedMemberId überschreibt die Auswahl (Trainer-Drilldown aus der Team-Sicht).
export function ProfilAnwesenheitContent({ forcedMemberId }: { forcedMemberId?: number }) {
  const [own, setOwn] = useState<MemberRef | null>(null)
  const [children, setChildren] = useState<MemberRef[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(forcedMemberId ?? null)
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    api.get('/profile/me').then(r => {
      const ownMember: MemberRef | null = r.data?.own_member ?? null
      const kids: MemberRef[] = r.data?.children ?? []
      setOwn(ownMember)
      setChildren(kids)
      setLoaded(true)
      if (forcedMemberId == null) {
        setSelectedId(ownMember?.id ?? kids[0]?.id ?? null)
      }
    }).catch(() => setLoaded(true))
  }, [forcedMemberId])

  // Auswählbare Mitglieder: eigenes Mitglied (falls Spieler) + alle verlinkten Kinder.
  const options = useMemo(() => {
    const list: { id: number; label: string }[] = []
    if (own) list.push({ id: own.id, label: `${own.first_name} ${own.last_name}` })
    for (const k of children) list.push({ id: k.id, label: `${k.first_name} ${k.last_name}` })
    return list
  }, [own, children])

  const effectiveId = forcedMemberId ?? selectedId

  if (!loaded) return <p className="text-brand-text-muted text-sm p-4">Laden…</p>
  if (effectiveId == null) {
    return <p className="text-brand-text-muted text-sm p-4">Keine Anwesenheitsdaten verfügbar.</p>
  }

  return (
    <div className="space-y-4">
      {forcedMemberId == null && options.length > 1 && (
        <div className="flex flex-wrap gap-2">
          {options.map(o => (
            <button
              key={o.id}
              onClick={() => setSelectedId(o.id)}
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                effectiveId === o.id
                  ? 'bg-brand-yellow text-brand-black'
                  : 'bg-brand-surface-card text-brand-text-muted hover:text-brand-text'
              }`}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}
      <AttendanceStatsView memberId={effectiveId} />
    </div>
  )
}

export default function ProfilAnwesenheitPage() {
  const [params] = useSearchParams()
  const memberParam = params.get('member')
  const forced = memberParam ? Number(memberParam) : undefined

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-bold text-brand-text mb-6">Anwesenheit</h1>
      <ProfilAnwesenheitContent forcedMemberId={forced} />
    </div>
  )
}
