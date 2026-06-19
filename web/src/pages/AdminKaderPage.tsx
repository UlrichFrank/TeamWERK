import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import KaderMemberSearch from '../components/KaderMemberSearch'
import KaderExtendedSearch from '../components/KaderExtendedSearch'
import KaderTrainerSearch from '../components/KaderTrainerSearch'
import PositionStatus from '../components/PositionStatus'
import CopyKaderModal from '../components/CopyKaderModal'
import AutoAssignModal from '../components/AutoAssignModal'
import { useEscapeKey } from '../lib/useEscapeKey'
import { errorStatus, errorData } from '../lib/errors'

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
  status?: string
}

interface Kader {
  id: number
  season_id: number
  age_class: string
  gender: string
  team_number: number
  team_id: number
  dedicated_birth_year: number | null
  games_per_season: number
  birth_years: number[]
  bracket_years: number[]
  members: Member[]
  member_count: number
  trainers: { id: number; name: string; user_id?: number; status?: string }[]
  extended_members: Member[]
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
  const [seasons, setSeasons] = useState<Season[]>([])
  const [selectedSeason, setSelectedSeason] = useState<Season | null>(null)
  const [kaderList, setKaderList] = useState<Kader[]>([])
  const [loading, setLoading] = useState(true)
  const [showCopyModal, setShowCopyModal] = useState(false)
  const [showAutoAssignModal, setShowAutoAssignModal] = useState(false)
  const [removing, setRemoving] = useState<Record<string, boolean>>({})
  const [initializing, setInitializing] = useState(false)
  const [toast, setToast] = useState<string | null>(null)
  const [activeAgeClass, setActiveAgeClass] = useState<string | null>(null)
  const [ageClassOptions, setAgeClassOptions] = useState<string[]>([])

  const [pendingDedicated, setPendingDedicated] = useState<Set<number>>(new Set())
  const [gpsValues, setGpsValues] = useState<Record<number, number>>({})

  const [createModal, setCreateModal] = useState<{
    ageClass: string
    gender: string
    nextTeamNumber: number
    bracketYears: number[]
  } | null>(null)
  const [createDedicatedYear, setCreateDedicatedYear] = useState<number | null>(null)
  const [creating, setCreating] = useState(false)

  const [deleteConfirm, setDeleteConfirm] = useState<Kader | null>(null)
  const [deleting, setDeleting] = useState(false)

  useEscapeKey(
    deleteConfirm ? () => setDeleteConfirm(null) :
    createModal ? () => setCreateModal(null) :
    null
  )

  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(null), 3000)
  }

  const loadKader = async (seasonId: number) => {
    const [kaderRes, ageClassRes] = await Promise.all([
      api.get(`/kader?season_id=${seasonId}`),
      api.get('/age-class-rules'),
    ])
    const list: Kader[] = Array.isArray(kaderRes.data) ? kaderRes.data : []
    setKaderList(list)
    setPendingDedicated(new Set())
    const initialGps: Record<number, number> = {}
    for (const k of list) initialGps[k.id] = k.games_per_season
    setGpsValues(initialGps)
    const options: string[] = (ageClassRes.data ?? []).map((r: { age_class: string }) => r.age_class)
    setAgeClassOptions(options)
    setActiveAgeClass(prev => {
      const classes = [...new Set(list.map(k => k.age_class))].sort()
      if (prev && classes.includes(prev)) return prev
      return classes[0] ?? null
    })
  }

  useEffect(() => {
    const init = async () => {
      const res = await api.get('/seasons')
      const all: Season[] = res.data ?? []
      setSeasons(all)
      const active = all.find(s => s.is_active) ?? null
      const defaultSeason = active ?? all[0] ?? null
      setSelectedSeason(defaultSeason)
      if (defaultSeason) await loadKader(defaultSeason.id)
    }
    init().finally(() => setLoading(false))
  }, [])

  useLiveUpdates(event => { if (event === 'kader' && selectedSeason) loadKader(selectedSeason.id) })

  const handleRemoveMember = async (kaderId: number, memberId: number) => {
    const key = `${kaderId}-${memberId}`
    setRemoving(prev => ({ ...prev, [key]: true }))
    try {
      await api.put(`/kader/${kaderId}`, { members_add: [], members_remove: [memberId] })
      if (selectedSeason) await loadKader(selectedSeason.id)
    } catch {
      showToast('Fehler beim Entfernen')
    } finally {
      setRemoving(prev => ({ ...prev, [key]: false }))
    }
  }

  const handleAddTrainer = async (kaderId: number, memberId: number) => {
    try {
      await api.put(`/kader/${kaderId}`, { trainers_add: [memberId] })
      if (selectedSeason) await loadKader(selectedSeason.id)
    } catch {
      showToast('Fehler beim Hinzufügen')
    }
  }

  const handleRemoveTrainer = async (kaderId: number, memberId: number) => {
    try {
      await api.put(`/kader/${kaderId}`, { trainers_remove: [memberId] })
      if (selectedSeason) await loadKader(selectedSeason.id)
    } catch {
      showToast('Fehler beim Entfernen')
    }
  }

  const handleRemoveExtendedMember = async (kaderId: number, memberId: number) => {
    const key = `ext-${kaderId}-${memberId}`
    setRemoving(prev => ({ ...prev, [key]: true }))
    try {
      await api.put(`/kader/${kaderId}`, { extended_members_remove: [memberId] })
      if (selectedSeason) await loadKader(selectedSeason.id)
    } catch {
      showToast('Fehler beim Entfernen')
    } finally {
      setRemoving(prev => ({ ...prev, [key]: false }))
    }
  }

  const handleInitialize = async () => {
    if (!selectedSeason) return
    setInitializing(true)
    try {
      await api.post('/kader', { season_id: selectedSeason.id })
      await loadKader(selectedSeason.id)
      showToast('Kader angelegt')
    } catch {
      showToast('Fehler beim Anlegen')
    } finally {
      setInitializing(false)
    }
  }

  const handleSetDedicatedYear = async (k: Kader, year: number) => {
    try {
      await api.put(`/kader/${k.id}`, { dedicated_birth_year: year })
      if (selectedSeason) await loadKader(selectedSeason.id)
    } catch {
      showToast('Fehler beim Speichern')
    }
  }

  const handleSetMixed = async (k: Kader) => {
    if (pendingDedicated.has(k.id)) {
      setPendingDedicated(prev => { const s = new Set(prev); s.delete(k.id); return s })
      return
    }
    try {
      await api.put(`/kader/${k.id}`, { set_dedicated_birth_year: true })
      if (selectedSeason) await loadKader(selectedSeason.id)
    } catch {
      showToast('Fehler beim Speichern')
    }
  }

  const handleCreateKader = async () => {
    if (!createModal || !selectedSeason) return
    if (!createModal.ageClass || !createModal.gender) return
    const teamNumber = kaderList.filter(k => k.age_class === createModal.ageClass && k.gender === createModal.gender).length + 1
    setCreating(true)
    try {
      await api.post('/kader', {
        season_id: selectedSeason.id,
        age_class: createModal.ageClass,
        gender: createModal.gender,
        team_number: teamNumber,
        dedicated_birth_year: createDedicatedYear,
      })
      setCreateModal(null)
      setCreateDedicatedYear(null)
      await loadKader(selectedSeason.id)
      showToast('Kader angelegt')
    } catch (e) {
      if (errorStatus(e) === 409) {
        showToast('Kader existiert bereits')
      } else {
        showToast('Fehler beim Anlegen')
      }
    } finally {
      setCreating(false)
    }
  }

  const handleSetAgeClass = async (k: Kader, newAgeClass: string) => {
    if (newAgeClass === k.age_class) return
    try {
      await api.put(`/kader/${k.id}`, { age_class: newAgeClass })
      setActiveAgeClass(newAgeClass)
      if (selectedSeason) await loadKader(selectedSeason.id)
    } catch {
      showToast('Fehler beim Speichern der Altersklasse')
    }
  }

  const handlePatchGamesPerSeason = async (kaderId: number, value: number) => {
    try {
      await api.patch(`/kader/${kaderId}/games-per-season`, { games_per_season: value })
    } catch {
      showToast('Fehler beim Speichern der Spielanzahl')
    }
  }

  const handleDeleteKader = async () => {
    if (!deleteConfirm) return
    setDeleting(true)
    try {
      await api.delete(`/kader/${deleteConfirm.id}`)
      setDeleteConfirm(null)
      if (selectedSeason) await loadKader(selectedSeason.id)
      showToast('Kader gelöscht')
    } catch (e) {
      if (errorStatus(e) === 409) {
        showToast(`Kader hat noch ${errorData<{ member_count?: number }>(e)?.member_count ?? ''} Mitglieder`)
      } else {
        showToast('Fehler beim Löschen')
      }
    } finally {
      setDeleting(false)
    }
  }

  if (loading) return <div className="text-brand-text-muted text-sm">Laden…</div>

  const groups = new Map<string, Kader[]>()
  for (const k of kaderList) {
    const key = groupKey(k)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(k)
  }

  const allGroupOrder: string[] = []
  for (const k of kaderList) {
    const key = groupKey(k)
    if (!allGroupOrder.includes(key)) allGroupOrder.push(key)
  }

  const ageClassTabs = [...new Set(kaderList.map(k => k.age_class))].sort()
  const groupOrder = activeAgeClass
    ? allGroupOrder.filter(key => key.startsWith(`${activeAgeClass}|`))
    : allGroupOrder

  return (
    <div className="max-w-3xl">
      {/* Toast */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 bg-brand-black text-white text-sm px-4 py-2 rounded-lg shadow-lg">
          {toast}
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between mb-6 gap-4 flex-wrap">
        <div className="flex items-center gap-3 flex-wrap">
          <h1 className="text-2xl font-bold">Kader</h1>
          {seasons.length > 0 && (
            <select
              value={selectedSeason?.id ?? ''}
              onChange={e => {
                const id = parseInt(e.target.value)
                const season = seasons.find(s => s.id === id) ?? null
                setSelectedSeason(season)
                if (season) loadKader(season.id)
              }}
              className="border border-brand-border rounded-md px-3 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow min-w-[9rem]"
            >
              {seasons.map(s => (
                <option key={s.id} value={s.id}>
                  {s.name}{s.is_active ? ' (aktiv)' : ''}
                </option>
              ))}
            </select>
          )}
        </div>
        {selectedSeason && (
          <div className="flex gap-2 flex-wrap">
            {kaderList.length > 0 && (
              <>
                <button
                  onClick={() => { setCreateModal({ ageClass: '', gender: '', nextTeamNumber: 1, bracketYears: [] }); setCreateDedicatedYear(null) }}
                  className="bg-brand-yellow text-brand-black px-4 py-1.5 rounded-md text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors whitespace-nowrap"
                >
                  + Mannschaft
                </button>
                <button
                  onClick={() => setShowCopyModal(true)}
                  className="border border-brand-border text-brand-text-muted px-4 py-1.5 rounded-md text-xs font-medium hover:border-brand-text-muted hover:text-brand-text transition-colors whitespace-nowrap"
                >
                  Aus vorheriger Saison kopieren
                </button>
                <button
                  onClick={() => setShowAutoAssignModal(true)}
                  className="border border-brand-border text-brand-text-muted px-4 py-1.5 rounded-md text-xs font-medium hover:border-brand-text-muted hover:text-brand-text transition-colors whitespace-nowrap"
                >
                  Auto-Assign
                </button>
              </>
            )}
          </div>
        )}
      </div>

      {/* Age class tabs */}
      {ageClassTabs.length > 0 && (
        <div className="flex gap-2 mb-6 border-b border-brand-border-subtle overflow-x-auto">
          {ageClassTabs.map(ac => (
            <button
              key={ac}
              onClick={() => setActiveAgeClass(ac)}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors whitespace-nowrap ${
                activeAgeClass === ac
                  ? 'border-brand-yellow text-brand-text'
                  : 'border-transparent text-brand-text-muted hover:text-brand-text'
              }`}
            >
              {ac}
            </button>
          ))}
        </div>
      )}

      {/* No seasons at all */}
      {seasons.length === 0 && (
        <div className="bg-brand-surface-card rounded-xl border-t-4 border-brand-yellow p-8 text-center">
          <p className="text-brand-text-muted text-sm">Bitte legen Sie eine Saison unter <strong>Einstellungen → Saisons</strong> an.</p>
        </div>
      )}

      {/* No kader yet */}
      {selectedSeason && kaderList.length === 0 && (
        <div className="bg-brand-surface-card rounded-xl border-t-4 border-brand-yellow p-8 text-center space-y-4">
          <p className="text-brand-text-muted text-sm">Noch keine Kader für <strong>{selectedSeason.name}</strong> vorhanden.</p>
          <div className="flex gap-3 justify-center flex-wrap">
            <button
              onClick={handleInitialize}
              disabled={initializing}
              className="bg-brand-yellow text-brand-black px-4 py-2.5 sm:py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
            >
              {initializing ? 'Anlegen…' : 'Kader für Saison anlegen'}
            </button>
            <button
              onClick={() => { setCreateModal({ ageClass: '', gender: '', nextTeamNumber: 1, bracketYears: [] }); setCreateDedicatedYear(null) }}
              className="border border-brand-border text-brand-text-muted px-4 py-2.5 sm:py-2 rounded-md text-sm font-medium hover:border-brand-text-muted hover:text-brand-text transition-colors"
            >
              + Einzelne Mannschaft
            </button>
            <button
              onClick={() => setShowCopyModal(true)}
              className="border border-brand-border text-brand-text-muted px-4 py-2.5 sm:py-2 rounded-md text-sm font-medium hover:border-brand-text-muted hover:text-brand-text transition-colors"
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

        return (
          <div key={key} className="mb-6">
            {group.map(k => {
              const isDedicated = k.dedicated_birth_year !== null
              const isPending = pendingDedicated.has(k.id)
              const showDedicatedDropdown = isPending || isDedicated
              const title = hasMultiple
                ? `${k.age_class} ${k.team_number} ${GENDER_LABEL[k.gender]}`
                : `${k.age_class} ${GENDER_LABEL[k.gender]}`

              return (
                <div key={k.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow mb-3">
                  {/* Card header */}
                  <div className="px-5 py-3 border-b border-brand-border-subtle flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <h2 className="font-semibold text-sm truncate text-brand-text">{title}</h2>
                      {k.birth_years.length > 0 && (
                        <span className="text-xs bg-brand-yellow text-brand-black px-2 py-0.5 rounded-full whitespace-nowrap font-medium">
                          {birthYearLabel(k.birth_years)}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="text-xs text-brand-text-muted">{k.member_count} Mitgl.</span>
                      <button
                        onClick={() => k.member_count === 0 ? setDeleteConfirm(k) : showToast('Erst alle Mitglieder entfernen')}
                        disabled={k.member_count > 0}
                        title={k.member_count > 0 ? 'Erst alle Mitglieder entfernen' : 'Kader löschen'}
                        aria-label="Kader löschen"
                        className="text-brand-text-subtle hover:text-brand-danger transition-colors disabled:cursor-not-allowed disabled:opacity-40 p-0.5 rounded"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  {/* Mode toggle */}
                  <div className="px-5 pt-3 pb-2 flex items-center gap-3 flex-wrap">
                    <span className="text-xs text-brand-text-muted font-medium">Jahrgänge:</span>
                    <div className="flex rounded-md border border-brand-border-subtle overflow-hidden text-xs">
                      <button
                        onClick={() => handleSetMixed(k)}
                        className={`px-3 py-1 transition-colors ${!showDedicatedDropdown
                          ? 'bg-brand-yellow text-brand-black font-medium'
                          : 'bg-white text-brand-text-muted hover:bg-brand-border-subtle'}`}
                      >
                        Gemischt
                      </button>
                      <button
                        onClick={() => {
                          if (!showDedicatedDropdown) {
                            setPendingDedicated(prev => new Set(prev).add(k.id))
                          }
                        }}
                        className={`px-3 py-1 transition-colors border-l border-brand-border-subtle ${showDedicatedDropdown
                          ? 'bg-brand-yellow text-brand-black font-medium'
                          : 'bg-white text-brand-text-muted hover:bg-brand-border-subtle'}`}
                      >
                        Dediziert
                      </button>
                    </div>
                    {showDedicatedDropdown && (
                      <select
                        key={k.dedicated_birth_year ?? 'empty'}
                        value={isDedicated && !isPending ? k.dedicated_birth_year! : ''}
                        onChange={e => {
                          const yr = parseInt(e.target.value)
                          if (!isNaN(yr)) handleSetDedicatedYear(k, yr)
                        }}
                        className="border border-brand-border-subtle rounded px-2 py-1 text-xs bg-white text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow"
                      >
                        <option value="">Jahrgang wählen…</option>
                        {k.bracket_years.map(yr => (
                          <option key={yr} value={yr}>{yr}</option>
                        ))}
                      </select>
                    )}
                    {ageClassOptions.length > 0 && (
                      <div className="ml-auto flex items-center gap-2 flex-wrap">
                        <span className="text-xs text-brand-text-muted font-medium">Altersklasse:</span>
                        <select
                          value={k.age_class}
                          onChange={e => handleSetAgeClass(k, e.target.value)}
                          className="border border-brand-border-subtle rounded px-2 py-1 text-xs bg-white text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow w-28"
                        >
                          {ageClassOptions.map(ac => (
                            <option key={ac} value={ac}>{ac}</option>
                          ))}
                        </select>
                        <span className="text-xs text-brand-text-muted font-medium">Spiele:</span>
                        <input
                          type="number"
                          min={0}
                          step={1}
                          value={gpsValues[k.id] ?? k.games_per_season}
                          onChange={e => setGpsValues(prev => ({ ...prev, [k.id]: Math.max(0, parseInt(e.target.value) || 0) }))}
                          onBlur={e => handlePatchGamesPerSeason(k.id, Math.max(0, parseInt(e.target.value) || 0))}
                          className="border border-brand-border-subtle rounded px-2 py-2.5 sm:py-1 text-xs bg-white text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow w-16 min-h-[44px] sm:min-h-0"
                          aria-label="Spiele pro Saison"
                        />
                      </div>
                    )}
                  </div>

                  {/* Position status */}
                  <div className="px-5 py-2 border-t border-brand-border-subtle">
                    <PositionStatus members={k.members ?? []} />
                  </div>

                  {/* Trainer search + list */}
                  <div className="px-5 py-2 border-t border-brand-border-subtle">
                    <KaderTrainerSearch
                      assignedTrainers={k.trainers ?? []}
                      onAdd={(memberId) => handleAddTrainer(k.id, memberId)}
                      onRemove={(memberId) => handleRemoveTrainer(k.id, memberId)}
                    />
                  </div>

                  {/* Member search */}
                  <div className="px-5 pt-2 pb-2 border-t border-brand-border-subtle">
                    <KaderMemberSearch
                      kaderId={k.id}
                      onMemberAdded={() => selectedSeason && loadKader(selectedSeason.id)}
                      birthYears={k.birth_years}
                    />
                  </div>

                  {/* Member list */}
                  {(k.members ?? []).length === 0 ? (
                    <p className="text-xs text-brand-text-subtle italic px-5 py-3">Keine Mitglieder</p>
                  ) : (
                    <ul className="divide-y divide-brand-border-subtle px-5 pb-4">
                      {(k.members ?? []).map(m => (
                        <li key={m.id} className="flex items-center justify-between py-2 gap-2">
                          <span className="text-sm text-brand-text flex items-center gap-1.5 flex-wrap">
                            {m.name}{' '}
                            <span className="text-brand-text-muted text-xs">
                              ({m.birth_year}/{GENDER_SHORT[m.gender] ?? m.gender})
                            </span>
                            {m.status === 'anwaerter' && (
                              <span className="inline-flex rounded-full px-2 py-0.5 text-xs font-medium bg-brand-green/10 text-brand-green">
                                Anwärter
                              </span>
                            )}
                          </span>
                          <button
                            onClick={() => handleRemoveMember(k.id, m.id)}
                            disabled={removing[`${k.id}-${m.id}`]}
                            className="text-brand-text-muted hover:text-brand-danger transition-colors disabled:opacity-40 p-1 rounded"
                            aria-label="Mitglied entfernen"
                            title="Mitglied entfernen"
                          >
                            <X className="w-3 h-3" />
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}

                  {/* Extended member search */}
                  <div className="px-5 pt-2 pb-2 border-t border-brand-border-subtle">
                    <p className="text-xs font-medium text-brand-text-muted mb-2">Erweiterter Kader</p>
                    <KaderExtendedSearch
                      kaderId={k.id}
                      onMemberAdded={() => selectedSeason && loadKader(selectedSeason.id)}
                    />
                  </div>

                  {/* Extended member list */}
                  {(k.extended_members ?? []).length === 0 ? (
                    <p className="text-xs text-brand-text-subtle italic px-5 py-3">Keine erweiterten Mitglieder</p>
                  ) : (
                    <ul className="divide-y divide-brand-border-subtle px-5 pb-4">
                      {(k.extended_members ?? []).map(m => (
                        <li key={m.id} className="flex items-center justify-between py-2 gap-2">
                          <span className="text-sm text-brand-text flex items-center gap-1.5 flex-wrap">
                            {m.name}{' '}
                            <span className="text-brand-text-muted text-xs">
                              ({m.birth_year}/{GENDER_SHORT[m.gender] ?? m.gender})
                            </span>
                            {m.status === 'anwaerter' && (
                              <span className="inline-flex rounded-full px-2 py-0.5 text-xs font-medium bg-brand-green/10 text-brand-green">
                                Anwärter
                              </span>
                            )}
                          </span>
                          <button
                            onClick={() => handleRemoveExtendedMember(k.id, m.id)}
                            disabled={removing[`ext-${k.id}-${m.id}`]}
                            className="text-brand-text-muted hover:text-brand-danger transition-colors disabled:opacity-40 p-1 rounded"
                            aria-label="Aus erweitertem Kader entfernen"
                            title="Aus erweitertem Kader entfernen"
                          >
                            <X className="w-3 h-3" />
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )
            })}

          </div>
        )
      })}


      {/* Create team modal */}
      {createModal && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4">
            <div className="px-6 py-4 border-b border-brand-border-subtle">
              <h3 className="font-semibold text-base text-brand-text">Neue Mannschaft anlegen</h3>
            </div>
            <div className="px-6 py-5 space-y-3">
              <div>
                <label className="text-xs font-medium text-brand-text-muted block mb-1">Altersklasse</label>
                <select
                  value={createModal.ageClass}
                  onChange={e => setCreateModal(prev => prev && ({ ...prev, ageClass: e.target.value }))}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                >
                  <option value="">Bitte wählen…</option>
                  {ageClassOptions.map(ac => (
                    <option key={ac} value={ac}>{ac}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-brand-text-muted block mb-1">Geschlecht</label>
                <select
                  value={createModal.gender}
                  onChange={e => setCreateModal(prev => prev && ({ ...prev, gender: e.target.value }))}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                >
                  <option value="">Bitte wählen…</option>
                  <option value="m">männlich</option>
                  <option value="f">weiblich</option>
                  <option value="mixed">gemischt</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-brand-text-muted block mb-1">Jahrgang</label>
                <select
                  value={createDedicatedYear ?? ''}
                  onChange={e => {
                    const v = e.target.value
                    setCreateDedicatedYear(v === '' ? null : parseInt(v))
                  }}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                >
                  <option value="">Gemischt (alle Jahrgänge)</option>
                  {createModal.bracketYears.map(yr => (
                    <option key={yr} value={yr}>{yr}</option>
                  ))}
                </select>
              </div>
            </div>
            <div className="flex gap-2 px-6 py-4 border-t border-brand-border-subtle justify-end">
              <button
                onClick={() => { setCreateModal(null); setCreateDedicatedYear(null) }}
                className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleCreateKader}
                disabled={creating || !createModal.ageClass || !createModal.gender}
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
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4">
            <div className="px-6 py-4 border-b border-brand-border-subtle">
              <h3 className="font-semibold text-base text-brand-text">Kader löschen?</h3>
            </div>
            <div className="px-6 py-5">
              <p className="text-sm text-brand-text-muted">
                {deleteConfirm.age_class}{deleteConfirm.team_number > 1 ? ` ${deleteConfirm.team_number}` : ''} {GENDER_LABEL[deleteConfirm.gender]} wird unwiderruflich gelöscht.
              </p>
            </div>
            <div className="flex gap-2 px-6 py-4 border-t border-brand-border-subtle justify-end">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleDeleteKader}
                disabled={deleting}
                className="px-4 py-2 text-sm bg-brand-danger text-white font-medium rounded-md hover:bg-brand-danger/90 transition-colors disabled:opacity-50"
              >
                {deleting ? 'Löschen…' : 'Löschen'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Copy modal */}
      {showCopyModal && selectedSeason && (
        <CopyKaderModal
          toSeasonId={selectedSeason.id}
          toSeasonName={selectedSeason.name}
          onDone={async () => {
            setShowCopyModal(false)
            await loadKader(selectedSeason.id)
            showToast('Kader erfolgreich kopiert')
          }}
          onClose={() => setShowCopyModal(false)}
        />
      )}

      {/* Auto-Assign modal */}
      {showAutoAssignModal && selectedSeason && (
        <AutoAssignModal
          seasonId={selectedSeason.id}
          onDone={async () => {
            setShowAutoAssignModal(false)
            await loadKader(selectedSeason.id)
            showToast('Auto-Assign abgeschlossen')
          }}
          onClose={() => setShowAutoAssignModal(false)}
        />
      )}
    </div>
  )
}
