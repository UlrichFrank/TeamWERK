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
  team_number: number
  dedicated_birth_year: number | null
  birth_years: number[]
  bracket_years: number[]
  members: Member[]
  member_count: number
}

const GENDER_LABEL: Record<string, string> = { m: 'männlich', f: 'weiblich', mixed: 'gemischt' }
const GENDER_SHORT: Record<string, string> = { m: 'm', f: 'w', mixed: 'mix' }

function groupKey(k: Kader) { return `${k.age_class}|${k.gender}` }

function birthYearLabel(years: number[]) {
  if (years.length === 1) return `Jg. ${years[0]}`
  if (years.length === 2) return `Jg. ${years[0]}/${years[1]}`
  return ''
}

export default function AdminKaderPage() {
  const [activeSeason, setActiveSeason] = useState<Season | null>(null)
  const [kaderList, setKaderList] = useState<Kader[]>([])
  const [loading, setLoading] = useState(true)
  const [showCopyModal, setShowCopyModal] = useState(false)
  const [removing, setRemoving] = useState<Record<string, boolean>>({})
  const [initializing, setInitializing] = useState(false)
  const [toast, setToast] = useState<string | null>(null)

  // Per-kader mode toggle state: kader IDs where user clicked "Dediziert" but not yet picked a year
  const [pendingDedicated, setPendingDedicated] = useState<Set<number>>(new Set())

  // Create new team modal
  const [createModal, setCreateModal] = useState<{
    ageClass: string
    gender: string
    nextTeamNumber: number
    bracketYears: number[]
  } | null>(null)
  const [createDedicatedYear, setCreateDedicatedYear] = useState<number | null>(null)
  const [creating, setCreating] = useState(false)

  // Delete confirmation
  const [deleteConfirm, setDeleteConfirm] = useState<Kader | null>(null)
  const [deleting, setDeleting] = useState(false)

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
    setPendingDedicated(new Set())
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

  const handleSetDedicatedYear = async (k: Kader, year: number) => {
    try {
      await api.put(`/admin/kader/${k.id}`, { dedicated_birth_year: year })
      await loadAll()
    } catch {
      showToast('Fehler beim Speichern')
    }
  }

  const handleSetMixed = async (k: Kader) => {
    if (pendingDedicated.has(k.id)) {
      // Just cancel pending mode, no API call needed
      setPendingDedicated(prev => { const s = new Set(prev); s.delete(k.id); return s })
      return
    }
    try {
      await api.put(`/admin/kader/${k.id}`, { set_dedicated_birth_year: true })
      await loadAll()
    } catch {
      showToast('Fehler beim Speichern')
    }
  }

  const handleCreateKader = async () => {
    if (!createModal || !activeSeason) return
    setCreating(true)
    try {
      await api.post('/admin/kader', {
        season_id: activeSeason.id,
        age_class: createModal.ageClass,
        gender: createModal.gender,
        team_number: createModal.nextTeamNumber,
        dedicated_birth_year: createDedicatedYear,
      })
      setCreateModal(null)
      setCreateDedicatedYear(null)
      await loadAll()
      showToast('Kader angelegt')
    } catch (err: any) {
      if (err?.response?.status === 409) {
        showToast('Kader existiert bereits')
      } else {
        showToast('Fehler beim Anlegen')
      }
    } finally {
      setCreating(false)
    }
  }

  const handleDeleteKader = async () => {
    if (!deleteConfirm) return
    setDeleting(true)
    try {
      await api.delete(`/admin/kader/${deleteConfirm.id}`)
      setDeleteConfirm(null)
      await loadAll()
      showToast('Kader gelöscht')
    } catch (err: any) {
      if (err?.response?.status === 409) {
        showToast(`Kader hat noch ${err.response.data?.member_count ?? ''} Mitglieder`)
      } else {
        showToast('Fehler beim Löschen')
      }
    } finally {
      setDeleting(false)
    }
  }

  if (loading) return <div className="text-gray-400 text-sm">Laden…</div>

  // Group kader by age_class|gender
  const groups = new Map<string, Kader[]>()
  for (const k of kaderList) {
    const key = groupKey(k)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(k)
  }

  // Ordered unique group keys (preserving sort order from API)
  const groupOrder: string[] = []
  for (const k of kaderList) {
    const key = groupKey(k)
    if (!groupOrder.includes(key)) groupOrder.push(key)
  }

  return (
    <div className="max-w-3xl">
      {/* Toast */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 bg-brand-black text-white text-sm px-4 py-2 rounded-lg shadow-lg">
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

      {/* Kader groups */}
      {groupOrder.map(key => {
        const group = groups.get(key)!
        const hasMultiple = group.length > 1
        const firstK = group[0]
        const canAddMore = group.length < 2

        return (
          <div key={key} className="mb-6">
            {/* Kader cards for this group */}
            {group.map(k => {
              const isDedicated = k.dedicated_birth_year !== null
              const isPending = pendingDedicated.has(k.id)
              const showDedicatedDropdown = isPending || isDedicated
              const title = hasMultiple
                ? `${k.age_class} ${k.team_number} ${GENDER_LABEL[k.gender]}`
                : `${k.age_class} ${GENDER_LABEL[k.gender]}`

              return (
                <div key={k.id} className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow mb-3 overflow-hidden">
                  {/* Card header */}
                  <div className="px-5 py-3 border-b border-gray-200 flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <h2 className="font-semibold text-sm truncate">{title}</h2>
                      {k.birth_years.length > 0 && (
                        <span className="text-xs bg-brand-yellow text-brand-black px-2 py-0.5 rounded-full whitespace-nowrap font-medium">
                          {birthYearLabel(k.birth_years)}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="text-xs text-gray-400">{k.member_count} Mitgl.</span>
                      <button
                        onClick={() => k.member_count === 0 ? setDeleteConfirm(k) : showToast('Erst alle Mitglieder entfernen')}
                        disabled={k.member_count > 0}
                        title={k.member_count > 0 ? 'Erst alle Mitglieder entfernen' : 'Kader löschen'}
                        className="text-gray-300 hover:text-red-500 transition-colors disabled:cursor-not-allowed disabled:opacity-40 px-1 py-0.5 rounded text-sm leading-none"
                      >
                        ×
                      </button>
                    </div>
                  </div>

                  {/* Mode toggle */}
                  <div className="px-5 pt-3 pb-2 flex items-center gap-3 flex-wrap">
                    <span className="text-xs text-gray-500 font-medium">Jahrgänge:</span>
                    <div className="flex rounded-md border border-gray-200 overflow-hidden text-xs">
                      <button
                        onClick={() => handleSetMixed(k)}
                        className={`px-3 py-1 transition-colors ${!showDedicatedDropdown
                          ? 'bg-brand-yellow text-brand-black font-medium'
                          : 'bg-white text-gray-600 hover:bg-gray-50'}`}
                      >
                        Gemischt
                      </button>
                      <button
                        onClick={() => {
                          if (!showDedicatedDropdown) {
                            setPendingDedicated(prev => new Set(prev).add(k.id))
                          }
                        }}
                        className={`px-3 py-1 transition-colors border-l border-gray-200 ${showDedicatedDropdown
                          ? 'bg-brand-yellow text-brand-black font-medium'
                          : 'bg-white text-gray-600 hover:bg-gray-50'}`}
                      >
                        Dediziert
                      </button>
                    </div>
                    {showDedicatedDropdown && (
                      <select
                        value={isDedicated && !isPending ? k.dedicated_birth_year! : ''}
                        onChange={e => {
                          const yr = parseInt(e.target.value)
                          if (!isNaN(yr)) handleSetDedicatedYear(k, yr)
                        }}
                        className="border border-gray-200 rounded px-2 py-1 text-xs bg-white focus:outline-none focus:ring-1 focus:ring-brand-blue"
                      >
                        <option value="">Jahrgang wählen…</option>
                        {k.bracket_years.map(yr => (
                          <option key={yr} value={yr}>{yr}</option>
                        ))}
                      </select>
                    )}
                  </div>

                  {/* Member search */}
                  <div className="px-5 pt-2 pb-2">
                    <KaderMemberSearch
                      kaderId={k.id}
                      onMemberAdded={loadAll}
                      birthYears={k.birth_years}
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
                            className="text-xs text-gray-400 hover:text-red-500 transition-colors disabled:opacity-40 px-1.5 py-0.5 rounded"
                            title="Mitglied entfernen"
                          >
                            ×
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )
            })}

            {/* Add second team button */}
            {canAddMore && activeSeason && (
              <button
                onClick={() => {
                  const nextNum = group.length + 1
                  const bracketYears = firstK.bracket_years
                  setCreateModal({
                    ageClass: firstK.age_class,
                    gender: firstK.gender,
                    nextTeamNumber: nextNum,
                    bracketYears,
                  })
                  setCreateDedicatedYear(null)
                }}
                className="text-sm text-brand-blue hover:text-brand-black transition-colors flex items-center gap-1 px-1 py-1"
              >
                <span className="text-base leading-none">+</span>
                Mannschaft anlegen ({firstK.age_class} {GENDER_LABEL[firstK.gender]})
              </button>
            )}
          </div>
        )
      })}

      {/* Create team modal */}
      {createModal && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 w-full max-w-sm mx-4">
            <h3 className="font-semibold text-base mb-4">
              Neue Mannschaft — {createModal.ageClass} {GENDER_LABEL[createModal.gender]} #{createModal.nextTeamNumber}
            </h3>
            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium text-gray-600 block mb-1">Jahrgang</label>
                <select
                  value={createDedicatedYear ?? ''}
                  onChange={e => {
                    const v = e.target.value
                    setCreateDedicatedYear(v === '' ? null : parseInt(v))
                  }}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-blue"
                >
                  <option value="">Gemischt (alle Jahrgänge)</option>
                  {createModal.bracketYears.map(yr => (
                    <option key={yr} value={yr}>{yr}</option>
                  ))}
                </select>
              </div>
            </div>
            <div className="flex gap-2 mt-5 justify-end">
              <button
                onClick={() => { setCreateModal(null); setCreateDedicatedYear(null) }}
                className="px-4 py-2 text-sm border border-gray-300 rounded-md hover:border-gray-500 transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleCreateKader}
                disabled={creating}
                className="px-4 py-2 text-sm bg-brand-yellow text-brand-black font-medium rounded-md hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
              >
                {creating ? 'Anlegen…' : 'Anlegen'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation dialog */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 w-full max-w-sm mx-4">
            <h3 className="font-semibold text-base mb-2">Kader löschen?</h3>
            <p className="text-sm text-gray-600 mb-5">
              {deleteConfirm.age_class}{deleteConfirm.team_number > 1 ? ` ${deleteConfirm.team_number}` : ''} {GENDER_LABEL[deleteConfirm.gender]} wird unwiderruflich gelöscht.
            </p>
            <div className="flex gap-2 justify-end">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2 text-sm border border-gray-300 rounded-md hover:border-gray-500 transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleDeleteKader}
                disabled={deleting}
                className="px-4 py-2 text-sm bg-red-500 text-white font-medium rounded-md hover:bg-red-600 transition-colors disabled:opacity-50"
              >
                {deleting ? 'Löschen…' : 'Löschen'}
              </button>
            </div>
          </div>
        </div>
      )}

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
