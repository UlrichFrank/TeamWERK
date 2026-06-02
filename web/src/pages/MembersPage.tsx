import { useRef, useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'

import { X, User, CreditCard } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { usePagination } from '../lib/usePagination'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import ActionMenu from '../components/ActionMenu'
import Pagination from '../components/Pagination'
import { useEscapeKey } from '../lib/useEscapeKey'
import PersonChip from '../components/PersonChip'

interface Member {
  id: number; first_name: string; last_name: string
  status: string; pass_number?: string; position?: string; gender?: string; club_functions?: string[]
  has_pending_profil_draft?: boolean
  has_pending_bank_draft?: boolean
  user_id?: number
  user_photo_url?: string
}

interface ImportRow {
  line: number
  status: 'created' | 'updated' | 'unchanged' | 'error'
  name: string
  dob?: string
  changes?: string[]
  message?: string
}

interface ImportReport {
  total: number
  created: number
  updated: number
  unchanged: number
  errors: number
  rows: ImportRow[]
}

const genderLabel = (g?: string) => g === 'm' ? 'm' : g === 'f' ? 'w' : 'd'

const statusBadgeStyles = (status: string) => {
  if (status === 'aktiv') return 'bg-brand-black text-white'
  if (status === 'verletzt') return 'bg-brand-yellow text-brand-black'
  return 'bg-brand-border-subtle text-brand-text-muted'
}

const rowStatusIcon = (s: ImportRow['status']) => {
  if (s === 'created') return '+'
  if (s === 'updated') return '~'
  if (s === 'unchanged') return '='
  return '✗'
}

const rowStatusColor = (s: ImportRow['status']) => {
  if (s === 'created') return 'text-brand-success'
  if (s === 'updated') return 'text-brand-blue'
  if (s === 'unchanged') return 'text-brand-text-subtle'
  return 'text-brand-danger'
}

const CLUB_FUNCTION_LABELS: Record<string, string> = {
  spieler: 'Spieler',
  trainer: 'Trainer',
  vorstand: 'Vorstand',
  vorstand_beisitzer: 'Vorstands-Beisitzer',
}

export default function MembersPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const [clubFunctionFilter, setClubFunctionFilter] = useState('')
  const extraParams = useMemo<Record<string, string>>(
    () => clubFunctionFilter ? { club_function: clubFunctionFilter } : ({} as Record<string, string>), // backend param name unchanged
    [clubFunctionFilter]
  )
  const { items, setSearch, currentPage, totalPages, goToPage, refresh } = usePagination<Member>('/members', 20, extraParams)
  useLiveUpdates((event) => { if (event === 'members') refresh() })
  const isAdmin = user?.role === 'admin'

  const [deletingIds, setDeletingIds] = useState<Set<number>>(new Set())

  const handleDelete = async (m: Member) => {
    if (!confirm(`Mitglied „${m.first_name} ${m.last_name}" wirklich löschen?`)) return
    setDeletingIds(prev => new Set(prev).add(m.id))
    try {
      await api.delete(`/admin/members/${m.id}`)
      refresh()
    } catch {
      alert('Löschen fehlgeschlagen.')
    } finally {
      setDeletingIds(prev => { const s = new Set(prev); s.delete(m.id); return s })
    }
  }

  const [showNew, setShowNew] = useState(false)
  const [newFirstName, setNewFirstName] = useState('')
  const [newLastName, setNewLastName] = useState('')
  const [creating, setCreating] = useState(false)

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newFirstName.trim() || !newLastName.trim()) return
    setCreating(true)
    try {
      const res = await api.post<{ id: number }>('/members', { first_name: newFirstName.trim(), last_name: newLastName.trim() })
      navigate(`/mitglieder/${res.data.id}`)
    } catch {
      alert('Anlegen fehlgeschlagen.')
      setCreating(false)
    }
  }

  const resetNew = () => {
    setShowNew(false)
    setNewFirstName('')
    setNewLastName('')
    setCreating(false)
  }

  const [showImport, setShowImport] = useState(false)
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importMode, setImportMode] = useState<'append' | 'update'>('append')
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<ImportReport | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleExport = () => {
    api.get('/members/export', { responseType: 'blob' }).then(r => {
      const url = URL.createObjectURL(r.data)
      const a = document.createElement('a')
      a.href = url
      a.download = 'mitglieder.csv'
      a.click()
      URL.revokeObjectURL(url)
    })
  }

  const handleImport = async () => {
    if (!importFile) return
    setImporting(true)
    try {
      const fd = new FormData()
      fd.append('file', importFile)
      fd.append('mode', importMode)
      const res = await api.post<ImportReport>('/members/import', fd)
      setImportResult(res.data)
      if (res.data.created > 0 || res.data.updated > 0) refresh()
    } catch {
      setImportResult(null)
      alert('Import fehlgeschlagen. Bitte CSV-Format prüfen.')
    } finally {
      setImporting(false)
    }
  }

  const resetImport = () => {
    setShowImport(false)
    setImportFile(null)
    setImportResult(null)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  useEscapeKey(showNew ? resetNew : showImport ? resetImport : null)

  return (
    <div>
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Mitglieder</h1>
          <div className="flex flex-col sm:flex-row gap-2">
            <input
              type="search"
              placeholder="Suchen…"
              onChange={e => setSearch(e.target.value)}
              className="border border-brand-border rounded-md px-3 py-2.5 sm:py-1.5 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-full sm:w-auto"
            />
            <select
              value={clubFunctionFilter}
              onChange={e => setClubFunctionFilter(e.target.value)}
              className="border border-brand-border rounded-md px-3 py-2.5 sm:py-1.5 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-full sm:w-auto"
            >
              <option value="">Alle Funktionen</option>
              {Object.entries(CLUB_FUNCTION_LABELS).map(([val, label]) => (
                <option key={val} value={val}>{label}</option>
              ))}
            </select>
            {isAdmin && (
              <>
                <button
                  onClick={() => setShowNew(true)}
                  className="text-sm bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-2.5 sm:py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
                >
                  + Neu
                </button>
                <button
                  onClick={() => setShowImport(true)}
                  className="text-sm bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-2.5 sm:py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
                >
                  Import CSV
                </button>
                <button
                  onClick={handleExport}
                  className="text-sm bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-2.5 sm:py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
                >
                  Export CSV
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Table — always visible, columns drop off as screen shrinks */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
              <th className="hidden sm:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Status</th>
              <th className="hidden md:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Position</th>
              <th className="hidden lg:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Gesch.</th>
              <th className="hidden xl:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Passnummer</th>
              {isAdmin && <th className="bg-brand-surface-card px-4 py-3" />}
            </tr>
          </thead>
          <tbody className="divide-y divide-brand-border-subtle">
            {items.map(m => (
              <tr key={m.id} className="hover:bg-brand-table-select transition-colors cursor-pointer" onClick={() => navigate(`/mitglieder/${m.id}`)}>
                <td className="px-4 py-3 font-medium text-brand-text">
                  <PersonChip userId={m.user_id} name={`${m.last_name}, ${m.first_name}`} photoUrl={m.user_photo_url} />
                  {isAdmin && (
                    <>
                      {m.has_pending_profil_draft && <User size={14} className="inline ml-2 text-brand-text-muted" aria-label="Persönliche Daten ausstehend" />}
                      {m.has_pending_bank_draft && <CreditCard size={14} className="inline ml-1 text-brand-text-muted" aria-label="Bankdaten ausstehend" />}
                    </>
                  )}
                  {/* Status badge inline on mobile */}
                  <div className="sm:hidden mt-0.5">
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeStyles(m.status)}`}>
                      {m.status}
                    </span>
                  </div>
                </td>
                <td className="hidden sm:table-cell px-4 py-3">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeStyles(m.status)}`}>
                    {m.status}
                  </span>
                </td>
                <td className="hidden md:table-cell px-4 py-3 text-brand-text-muted">{m.position || '–'}</td>
                <td className="hidden lg:table-cell px-4 py-3 text-brand-text-muted">{m.club_functions?.includes('spieler') ? genderLabel(m.gender) : '–'}</td>
                <td className="hidden xl:table-cell px-4 py-3 text-brand-text-muted">{m.pass_number || '–'}</td>
                {isAdmin && (
                  <td className="px-4 py-3 text-right" onClick={e => e.stopPropagation()}>
                    <ActionMenu actions={[
                      { label: deletingIds.has(m.id) ? 'Löschen…' : 'Löschen', onClick: () => handleDelete(m), variant: 'danger' },
                    ]} />
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={goToPage} />

      {/* Neu-Mitglied Modal */}
      {showNew && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm">
            <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
              <h2 className="font-semibold text-base text-brand-text">Neues Mitglied anlegen</h2>
              <button onClick={resetNew} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="px-6 py-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Vorname</label>
                <input
                  autoFocus
                  type="text"
                  value={newFirstName}
                  onChange={e => setNewFirstName(e.target.value)}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Nachname</label>
                <input
                  type="text"
                  value={newLastName}
                  onChange={e => setNewLastName(e.target.value)}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  required
                />
              </div>
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={resetNew} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                  Abbrechen
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="px-4 py-2 text-sm bg-brand-yellow text-brand-black font-medium rounded-md hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                >
                  {creating ? 'Anlegen…' : 'Anlegen'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Import Modal */}
      {showImport && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-lg max-h-[90vh] flex flex-col">
            <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
              <h2 className="font-semibold text-base text-brand-text">CSV-Import</h2>
              <button onClick={resetImport} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>

            {!importResult ? (
              <div className="px-6 py-5 space-y-5">
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">CSV-Datei</label>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".csv"
                    onChange={e => setImportFile(e.target.files?.[0] ?? null)}
                    className="block w-full text-sm text-brand-text-muted file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-sm file:font-medium file:bg-brand-yellow file:text-brand-black hover:file:bg-black hover:file:text-white cursor-pointer"
                  />
                  <p className="text-xs text-brand-text-subtle mt-1">Erwartet: Semikolon-getrennt, UTF-8. Spalten: Mitgliedsnummer, Vorname, Nachname, Geburtsdatum, Geschlecht, Passnummer, Trikotnummer, Position, Status, Benutzer_Email, Erziehungsberechtigter1_Email, Erziehungsberechtigter2_Email</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-2">Modus</label>
                  <div className="space-y-2">
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="importMode"
                        value="append"
                        checked={importMode === 'append'}
                        onChange={() => setImportMode('append')}
                        className="mt-0.5 accent-brand-yellow"
                      />
                      <div>
                        <span className="text-sm font-medium text-brand-text">Nur ergänzen</span>
                        <p className="text-xs text-brand-text-subtle">Neue Mitglieder anlegen, bestehende unverändert lassen</p>
                      </div>
                    </label>
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="importMode"
                        value="update"
                        checked={importMode === 'update'}
                        onChange={() => setImportMode('update')}
                        className="mt-0.5 accent-brand-yellow"
                      />
                      <div>
                        <span className="text-sm font-medium text-brand-text">Fehlende + geänderte Felder aktualisieren</span>
                        <p className="text-xs text-brand-text-subtle">Bestehende Mitglieder werden mit nicht-leeren CSV-Werten aktualisiert. Felder werden nie geleert.</p>
                      </div>
                    </label>
                  </div>
                </div>

                <div className="flex justify-end gap-2 pt-1">
                  <button onClick={resetImport} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                    Abbrechen
                  </button>
                  <button
                    onClick={handleImport}
                    disabled={!importFile || importing}
                    className="px-4 py-2 text-sm bg-brand-yellow text-brand-black font-medium rounded-md hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >
                    {importing ? 'Importieren…' : 'Import starten'}
                  </button>
                </div>
              </div>
            ) : (
              <div className="flex flex-col flex-1 min-h-0">
                {/* Summary */}
                <div className="px-6 py-4 border-b border-brand-border-subtle bg-brand-surface-card">
                  <p className="text-sm font-medium text-brand-text mb-2">{importResult.total} Zeilen verarbeitet</p>
                  <div className="flex flex-wrap gap-3 text-xs">
                    {importResult.created > 0 && <span className="text-brand-success font-medium">+ {importResult.created} neu</span>}
                    {importResult.updated > 0 && <span className="text-brand-blue font-medium">~ {importResult.updated} aktualisiert</span>}
                    {importResult.unchanged > 0 && <span className="text-brand-text-subtle">= {importResult.unchanged} unverändert</span>}
                    {importResult.errors > 0 && <span className="text-brand-danger font-medium">✗ {importResult.errors} Fehler</span>}
                  </div>
                </div>

                {/* Detail rows */}
                <div className="overflow-y-auto flex-1 px-6 py-3 space-y-1 text-xs font-mono">
                  {importResult.rows.filter(r => r.status !== 'unchanged').map((row, i) => (
                    <div key={i} className={rowStatusColor(row.status)}>
                      <span className="font-bold">{rowStatusIcon(row.status)}</span>{' '}
                      <span>Z.{row.line} {row.name}{row.dob ? ` (${row.dob.slice(0, 10)})` : ''}</span>
                      {row.changes?.map((c, j) => (
                        <div key={j} className="pl-4 text-brand-text-muted">{c}</div>
                      ))}
                      {row.message && <span className="pl-4 italic">{row.message}</span>}
                    </div>
                  ))}
                  {importResult.unchanged > 0 && (
                    <div className="text-brand-text-subtle">= {importResult.unchanged}× unverändert</div>
                  )}
                </div>

                <div className="px-6 py-4 border-t border-brand-border-subtle flex justify-end">
                  <button onClick={resetImport} className="px-4 py-2 text-sm bg-brand-yellow text-brand-black font-medium rounded-md hover:bg-brand-black hover:text-brand-yellow transition-colors">
                    Schließen
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
