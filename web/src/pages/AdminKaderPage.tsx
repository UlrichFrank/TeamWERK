import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import KaderMemberSearch from '../components/KaderMemberSearch'
import CopyKaderModal from '../components/CopyKaderModal'

interface Season {
  id: number
  name: string
  is_active: boolean
}

interface Member {
  id: number
  name: string
  birth_year: number
  gender: string
}

interface Kader {
  id: number
  season_id: number
  age_class: string
  gender: string
  members: Member[]
  member_count: number
}

const GENDER_LABEL: Record<string, string> = { m: 'männlich', f: 'weiblich', mixed: 'gemischt' }
const GENDER_SHORT: Record<string, string> = { m: 'm', f: 'w', mixed: 'mix' }

export default function AdminKaderPage() {
  const [activeSeason, setActiveSeason] = useState<Season | null>(null)
  const [kaderList, setKaderList] = useState<Kader[]>([])
  const [loading, setLoading] = useState(true)
  const [showCopyModal, setShowCopyModal] = useState(false)
  const [removing, setRemoving] = useState<Record<string, boolean>>({})
  const [initializing, setInitializing] = useState(false)
  const [toast, setToast] = useState<string | null>(null)

  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(null), 3000)
  }

  const loadAll = async () => {
    const [seasonsRes, kaderRes] = await Promise.all([
      api.get('/admin/seasons'),
      api.get('/admin/kader'),
    ])
    const seasons: Season[] = seasonsRes.data ?? []
    setActiveSeason(seasons.find(s => s.is_active) ?? null)
    setKaderList(Array.isArray(kaderRes.data) ? kaderRes.data : [])
  }

  useEffect(() => {
    loadAll().finally(() => setLoading(false))
  }, [])

  const handleRemoveMember = async (kaderId: number, memberId: number) => {
    const key = `${kaderId}-${memberId}`
    setRemoving(prev => ({ ...prev, [key]: true }))
    try {
      await api.put(`/admin/kader/${kaderId}`, { members_add: [], members_remove: [memberId] })
      await loadAll()
    } catch {
      showToast('Fehler beim Entfernen')
    } finally {
      setRemoving(prev => ({ ...prev, [key]: false }))
    }
  }

  const handleInitialize = async () => {
    if (!activeSeason) return
    setInitializing(true)
    try {
      await api.post('/admin/kader', { season_id: activeSeason.id })
      await loadAll()
      showToast('Kader angelegt')
    } catch {
      showToast('Fehler beim Anlegen')
    } finally {
      setInitializing(false)
    }
  }

  if (loading) return <div className="text-gray-400 text-sm">Laden…</div>

  return (
    <div className="max-w-3xl">
      {/* Toast */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 bg-brand-black text-brand-white text-sm px-4 py-2 rounded-lg shadow-lg">
          {toast}
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between mb-6 gap-4">
        <h1 className="text-2xl font-bold">
          Kader
          {activeSeason && (
            <span className="ml-2 text-base font-normal text-gray-500">{activeSeason.name}</span>
          )}
        </h1>
        {activeSeason && kaderList.length > 0 && (
          <button
            onClick={() => setShowCopyModal(true)}
            className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors whitespace-nowrap"
          >
            Aus vorheriger Saison kopieren
          </button>
        )}
      </div>

      {/* No active season */}
      {!activeSeason && (
        <div className="bg-gray-50 rounded-xl border-t-4 border-brand-yellow p-8 text-center">
          <p className="text-gray-500 text-sm">Bitte aktivieren Sie eine Saison unter <strong>Saisons</strong>.</p>
        </div>
      )}

      {/* No kader yet */}
      {activeSeason && kaderList.length === 0 && (
        <div className="bg-gray-50 rounded-xl border-t-4 border-brand-yellow p-8 text-center space-y-4">
          <p className="text-gray-500 text-sm">Noch keine Kader für <strong>{activeSeason.name}</strong> vorhanden.</p>
          <div className="flex gap-3 justify-center flex-wrap">
            <button
              onClick={handleInitialize}
              disabled={initializing}
              className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
            >
              {initializing ? 'Anlegen…' : 'Kader für Saison anlegen'}
            </button>
            <button
              onClick={() => setShowCopyModal(true)}
              className="border border-gray-300 text-gray-700 px-4 py-2 rounded-md text-sm font-medium hover:border-gray-500 transition-colors"
            >
              Aus vorheriger Saison kopieren
            </button>
          </div>
        </div>
      )}

      {/* Kader list */}
      {kaderList.map(k => (
        <div key={k.id} className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow mb-4 overflow-hidden">
          {/* Header */}
          <div className="px-5 py-3 border-b border-gray-200 flex items-center justify-between">
            <h2 className="font-semibold text-sm">
              {k.age_class} <span className="font-normal text-gray-500">{GENDER_LABEL[k.gender]}</span>
            </h2>
            <span className="text-xs text-gray-400">{k.member_count} Mitglieder</span>
          </div>

          {/* Member search */}
          <div className="px-5 pt-4 pb-2">
            <KaderMemberSearch
              kaderId={k.id}
              onMemberAdded={loadAll}
            />
          </div>

          {/* Member list */}
          {(k.members ?? []).length === 0 ? (
            <p className="text-xs text-gray-400 italic px-5 py-3">Keine Mitglieder</p>
          ) : (
            <ul className="divide-y divide-gray-100 px-5 pb-4">
              {(k.members ?? []).map(m => (
                <li key={m.id} className="flex items-center justify-between py-2 gap-2">
                  <span className="text-sm">
                    {m.name}{' '}
                    <span className="text-gray-400 text-xs">
                      ({m.birth_year}/{GENDER_SHORT[m.gender] ?? m.gender})
                    </span>
                  </span>
                  <button
                    onClick={() => handleRemoveMember(k.id, m.id)}
                    disabled={removing[`${k.id}-${m.id}`]}
                    className="text-xs text-gray-400 hover:text-brand-error transition-colors disabled:opacity-40 px-1.5 py-0.5 rounded"
                    title="Mitglied entfernen"
                  >
                    ×
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      ))}

      {/* Copy modal */}
      {showCopyModal && activeSeason && (
        <CopyKaderModal
          toSeasonId={activeSeason.id}
          toSeasonName={activeSeason.name}
          onDone={async () => {
            setShowCopyModal(false)
            await loadAll()
            showToast('Kader erfolgreich kopiert')
          }}
          onClose={() => setShowCopyModal(false)}
        />
      )}
    </div>
  )
}
