import { useEffect, useState } from 'react'
import { Trophy } from 'lucide-react'
import { PlayerStat, fetchMemberStats, sevenMeterRate } from '../lib/staffeln'

const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-4'

/**
 * Saisonbilanz eines Mitglieds aus den ausgewerteten BWHV-Spielberichten.
 *
 * Ein Mitglied kann in mehreren Staffeln auflaufen (Doppelspielrecht), deshalb
 * je Staffel ein Block. Ohne ausgewertete Spiele rendert das Panel nichts —
 * eine leere Kachel im Profil wäre Rauschen.
 */
export default function SaisonbilanzPanel({ memberId }: { memberId: number }) {
  const [stats, setStats] = useState<PlayerStat[]>([])
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    let active = true
    fetchMemberStats(memberId)
      .then((s) => { if (active) setStats(s) })
      .catch(() => { /* Statistik ist Beiwerk — ein Fehler verdrängt das Profil nicht */ })
      .finally(() => { if (active) setLoaded(true) })
    return () => { active = false }
  }, [memberId])

  if (!loaded || stats.length === 0) return null

  return (
    <div className={CARD}>
      <h2 className="text-sm font-medium text-brand-text mb-3 inline-flex items-center gap-1">
        <Trophy className="w-4 h-4" /> Saisonbilanz
      </h2>
      <div className="space-y-3">
        {stats.map((s) => (
          <div key={s.playerId}>
            <p className="text-xs text-brand-text-muted mb-1">{s.teamName}</p>
            <dl className="grid grid-cols-2 sm:grid-cols-4 gap-2">
              <Kennzahl label="Spiele" value={String(s.games)} />
              <Kennzahl label="Tore" value={String(s.goals)} />
              <Kennzahl
                label="7m"
                value={sevenMeterRate(s) === null ? '—' : `${s.sevenMGoals}/${s.sevenMAttempts}`}
                hint={sevenMeterRate(s) === null ? undefined : `${sevenMeterRate(s)} %`}
              />
              <Kennzahl
                label="Strafen"
                value={
                  [s.twoMin && `${s.twoMin}×2′`, s.warnings && `${s.warnings}×V`, s.disq && `${s.disq}×D`]
                    .filter(Boolean)
                    .join(' ') || '—'
                }
              />
            </dl>
          </div>
        ))}
      </div>
      <p className="text-xs text-brand-text-subtle mt-3">
        Aus den offiziellen Spielberichten des Verbands.
      </p>
    </div>
  )
}

function Kennzahl({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div>
      <dt className="text-xs text-brand-text-muted">{label}</dt>
      <dd className="text-sm text-brand-text font-medium">
        {value}
        {hint && <span className="text-xs text-brand-text-muted font-normal ml-1">{hint}</span>}
      </dd>
    </div>
  )
}
