import { useRef, useState, useMemo, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

import { X, User, CreditCard, ChevronDown, AlertTriangle, ChevronRight } from 'lucide-react'
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
  member_number_conflict?: string
  user_id?: number
  user_photo_url?: string
  can?: { edit: boolean; delete: boolean }
}

interface ImportRow {
  line: number
  status: 'created' | 'updated' | 'unchanged' | 'skipped' | 'error' | 'not_found'
  name: string
  dob?: string
  changes?: string[]
  message?: string
  iban_warning?: string
}

interface ImportReport {
  total: number
  created: number
  updated: number
  unchanged: number
  skipped?: number
  errors: number
  not_found?: number
  rows: ImportRow[]
}

// Aktualisierbare DB-Spalten ↔ Anzeige-Label für die Feld-Auswahl beim Import.
// Die `col`-Werte müssen exakt den Backend-Spaltennamen entsprechen (siehe Import-Handler).
const IMPORT_FIELDS: { col: string; label: string }[] = [
  { col: 'member_number', label: 'Mitgliedsnummer' },
  { col: 'date_of_birth', label: 'Geburtsdatum' },
  { col: 'gender', label: 'Geschlecht' },
  { col: 'pass_number', label: 'Passnummer' },
  { col: 'position', label: 'Position' },
  { col: 'status', label: 'Status / Beitragsfrei' },
  { col: 'home_club', label: 'Stammverein' },
  { col: 'jersey_number', label: 'Trikotnummer' },
  { col: 'street', label: 'Adresse' },
  { col: 'zip', label: 'PLZ' },
  { col: 'city', label: 'Ort' },
  { col: 'join_date', label: 'Mitglied seit' },
  { col: 'account_holder', label: 'Kontoinhaber' },
  { col: 'sepa_mandat', label: 'SEPA-Mandat' },
  { col: 'iban', label: 'IBAN' },
]

const genderLabel = (g?: string) => g === 'm' ? 'm' : g === 'f' ? 'w' : 'd'

const statusBadgeStyles = (status: string) => {
  if (status === 'aktiv') return 'bg-brand-black text-white'
  if (status === 'verletzt') return 'bg-brand-yellow text-brand-black'
  if (status === 'honorar') return 'bg-brand-blue/10 text-brand-blue'
  if (status === 'anwaerter') return 'bg-brand-green/10 text-brand-green'
  return 'bg-brand-border-subtle text-brand-text-muted'
}

const STATUS_LABEL: Record<string, string> = {
  aktiv: 'Aktiv',
  verletzt: 'Verletzt',
  pausiert: 'Pausiert',
  passiv: 'Passiv',
  honorar: 'Honorar',
  anwaerter: 'Anwärter',
  ausgetreten: 'Ausgetreten',
}

const MEMBER_NUMBER_CONFLICT_LABEL: Record<string, string> = {
  duplicate: 'Mitgliedsnummer doppelt vergeben',
  non_numeric: 'Mitgliedsnummer nicht numerisch',
  missing: 'Mitgliedsnummer fehlt',
}

const rowStatusIcon = (s: ImportRow['status']) => {
  if (s === 'created') return '+'
  if (s === 'updated') return '~'
  if (s === 'unchanged') return '='
  if (s === 'skipped') return '⊘'
  if (s === 'not_found') return '—'
  return '✗'
}

const rowStatusColor = (s: ImportRow['status']) => {
  if (s === 'created') return 'text-brand-success'
  if (s === 'updated') return 'text-brand-blue'
  if (s === 'unchanged') return 'text-brand-text-subtle'
  if (s === 'skipped') return 'text-brand-text-muted'
  if (s === 'not_found') return 'text-brand-text-muted'
  return 'text-brand-danger'
}

// A change overwrites an existing value when the old (quoted) part before → is non-empty.
// Format produced by backend: `Label: "old" → "new"` — link notes use a different pattern.
const isOverwrite = (change: string) => /: ".+" →/.test(change)

const rowHasOverwrites = (row: ImportRow) =>
  row.status === 'updated' && (row.changes?.some(isOverwrite) ?? false)

const CLUB_FUNCTION_LABELS: Record<string, string> = {
  spieler: 'Spieler',
  trainer: 'Trainer',
  sportliche_leitung: 'Sportliche Leitung',
  vorstand: 'Vorstand',
  vorstand_beisitzer: 'Vorstands-Beisitzer',
  kassierer: 'Kassierer',
}

export default function MembersPage() {
  const navigate = useNavigate()
  const { hasCapability } = useAuth()
  const [clubFunctionFilter, setClubFunctionFilter] = useState('')
  const [unlinkedUserFilter, setUnlinkedUserFilter] = useState(false)
  const [hasDraftFilter, setHasDraftFilter] = useState(false)
  const extraParams = useMemo<Record<string, string>>(() => {
    const p: Record<string, string> = {}
    if (clubFunctionFilter) p.club_function = clubFunctionFilter
    if (unlinkedUserFilter) p.unlinked_user = '1'
    if (hasDraftFilter) p.has_draft = '1'
    return p
  }, [clubFunctionFilter, unlinkedUserFilter, hasDraftFilter])
  const { items, setSearch, currentPage, totalPages, goToPage, refresh } = usePagination<Member>('/members', 20, extraParams)
  useLiveUpdates((event) => { if (event === 'members') refresh() })
  const isAdmin = hasCapability('manage_members')

  const [deletingIds, setDeletingIds] = useState<Set<number>>(new Set())

  const handleDelete = async (m: Member) => {
    if (!confirm(`Mitglied „${m.first_name} ${m.last_name}" wirklich löschen?`)) return
    setDeletingIds(prev => new Set(prev).add(m.id))
    try {
      await api.delete(`/members/${m.id}`)
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
  const [importMode, setImportMode] = useState<'append' | 'update' | 'enrich'>('append')
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<ImportReport | null>(null)
  const [previewResult, setPreviewResult] = useState<ImportReport | null>(null)
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())
  // Feld-Auswahl (nur update/enrich): standardmäßig alle Spalten ausgewählt.
  const [selectedFields, setSelectedFields] = useState<Set<string>>(() => new Set(IMPORT_FIELDS.map(f => f.col)))
  // Mitglieder-Auswahl: angehakte CSV-Zeilennummern aus der Vorschau (default alle updated-Zeilen).
  const [selectedLines, setSelectedLines] = useState<Set<number>>(new Set())
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [showActionsMenu, setShowActionsMenu] = useState(false)
  const actionsMenuRef = useRef<HTMLDivElement>(null)

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

  // Feld-Whitelist nur bei update/enrich anhängen. Leere Auswahl → Sentinel '__none__'
  // (matcht keine Spalte → es wird nichts aktualisiert), sonst würde das Backend leer = alle interpretieren.
  const appendFields = (fd: FormData) => {
    if (importMode === 'update' || importMode === 'enrich') {
      fd.append('fields', selectedFields.size > 0 ? [...selectedFields].join(',') : '__none__')
    }
  }

  const handlePreview = async () => {
    if (!importFile) return
    setImporting(true)
    try {
      const fd = new FormData()
      fd.append('file', importFile)
      fd.append('mode', importMode)
      fd.append('preview', '1')
      appendFields(fd)
      const res = await api.post<ImportReport>('/members/import', fd)
      setPreviewResult(res.data)
      setExpandedRows(new Set())
      // Standardmäßig alle Zeilen mit Änderungen anhaken.
      setSelectedLines(new Set(res.data.rows.filter(r => r.status === 'updated').map(r => r.line)))
    } catch {
      alert('Vorschau fehlgeschlagen. Bitte CSV-Format prüfen.')
    } finally {
      setImporting(false)
    }
  }

  const handleImport = async () => {
    if (!importFile) return
    setImporting(true)
    try {
      const fd = new FormData()
      fd.append('file', importFile)
      fd.append('mode', importMode)
      appendFields(fd)
      // Nur angehakte Zeilen anwenden. Leere Auswahl → Sentinel '0' (keine reale Datenzeile).
      if (importMode === 'update' || importMode === 'enrich') {
        fd.append('apply_lines', selectedLines.size > 0 ? [...selectedLines].join(',') : '0')
      }
      const res = await api.post<ImportReport>('/members/import', fd)
      setImportResult(res.data)
      if (res.data.created > 0 || res.data.updated > 0) refresh()
    } catch {
      alert('Import fehlgeschlagen. Bitte CSV-Format prüfen.')
    } finally {
      setImporting(false)
    }
  }

  const toggleField = (col: string) => {
    setSelectedFields(prev => {
      const next = new Set(prev)
      if (next.has(col)) next.delete(col); else next.add(col)
      return next
    })
  }

  const toggleLine = (line: number) => {
    setSelectedLines(prev => {
      const next = new Set(prev)
      if (next.has(line)) next.delete(line); else next.add(line)
      return next
    })
  }

  const resetImport = () => {
    setShowImport(false)
    setImportFile(null)
    setImportResult(null)
    setPreviewResult(null)
    setExpandedRows(new Set())
    setSelectedFields(new Set(IMPORT_FIELDS.map(f => f.col)))
    setSelectedLines(new Set())
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const toggleRow = (line: number) => {
    setExpandedRows(prev => {
      const next = new Set(prev)
      if (next.has(line)) next.delete(line); else next.add(line)
      return next
    })
  }

  useEscapeKey(showNew ? resetNew : showImport ? resetImport : showActionsMenu ? () => setShowActionsMenu(false) : null)

  useEffect(() => {
    if (!showActionsMenu) return
    const handler = (e: MouseEvent) => {
      if (actionsMenuRef.current && !actionsMenuRef.current.contains(e.target as Node)) {
        setShowActionsMenu(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showActionsMenu])

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
              className="border border-brand-border rounded-md px-3 py-2.5 sm:py-1.5 text-xs text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-full sm:w-auto"
            />
            <select
              value={clubFunctionFilter}
              onChange={e => setClubFunctionFilter(e.target.value)}
              className="border border-brand-border rounded-md px-3 py-2.5 sm:py-1.5 text-xs text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-full sm:w-auto"
            >
              <option value="">Alle Funktionen</option>
              {Object.entries(CLUB_FUNCTION_LABELS).map(([val, label]) => (
                <option key={val} value={val}>{label}</option>
              ))}
            </select>
            {isAdmin && (
              <div className="flex flex-wrap gap-3 items-center">
                <label className="flex items-center gap-1.5 text-xs text-brand-text cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={unlinkedUserFilter}
                    onChange={e => setUnlinkedUserFilter(e.target.checked)}
                    className="rounded border-brand-border accent-brand-yellow"
                  />
                  Ohne App-Account
                </label>
                <label className="flex items-center gap-1.5 text-xs text-brand-text cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={hasDraftFilter}
                    onChange={e => setHasDraftFilter(e.target.checked)}
                    className="rounded border-brand-border accent-brand-yellow"
                  />
                  Mit Änderungsantrag
                </label>
              </div>
            )}
            {isAdmin && (
              <div ref={actionsMenuRef} className="relative">
                <div className="flex">
                  <button
                    onClick={() => setShowNew(true)}
                    className="text-xs bg-brand-yellow text-brand-black border border-brand-yellow rounded-l-md px-3 py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
                  >
                    + Neu
                  </button>
                  <button
                    onClick={() => setShowActionsMenu(v => !v)}
                    aria-label="Weitere Aktionen"
                    className="text-xs bg-brand-yellow text-brand-black border border-brand-yellow border-l-brand-black/20 border-l rounded-r-md px-2 py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
                  >
                    <ChevronDown className="w-4 h-4" />
                  </button>
                </div>
                {showActionsMenu && (
                  <div
                    className="absolute right-0 mt-1 w-40 bg-white border border-brand-border rounded-md shadow-lg z-20 overflow-hidden"
                    onBlur={() => setShowActionsMenu(false)}
                  >
                    <button
                      onClick={() => { setShowActionsMenu(false); setShowImport(true) }}
                      className="w-full text-left px-4 py-2.5 text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
                    >
                      Import CSV
                    </button>
                    <button
                      onClick={() => { setShowActionsMenu(false); handleExport() }}
                      className="w-full text-left px-4 py-2.5 text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
                    >
                      Export CSV
                    </button>
                  </div>
                )}
              </div>
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
                  {m.can?.edit && (
                    <>
                      {m.has_pending_profil_draft && <User size={14} className="inline ml-2 text-brand-text-muted" aria-label="Persönliche Daten ausstehend" />}
                      {m.has_pending_bank_draft && <CreditCard size={14} className="inline ml-1 text-brand-text-muted" aria-label="Bankdaten ausstehend" />}
                    </>
                  )}
                  {m.member_number_conflict && (
                    <AlertTriangle
                      size={14}
                      className="inline ml-2 text-brand-danger"
                      aria-label={MEMBER_NUMBER_CONFLICT_LABEL[m.member_number_conflict] ?? 'Mitgliedsnummer-Konflikt'}
                    />
                  )}
                  {/* Status badge inline on mobile */}
                  <div className="sm:hidden mt-0.5">
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeStyles(m.status)}`}>
                      {STATUS_LABEL[m.status] ?? m.status}
                    </span>
                  </div>
                </td>
                <td className="hidden sm:table-cell px-4 py-3">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeStyles(m.status)}`}>
                    {STATUS_LABEL[m.status] ?? m.status}
                  </span>
                </td>
                <td className="hidden md:table-cell px-4 py-3 text-brand-text-muted">{m.position || '–'}</td>
                <td className="hidden lg:table-cell px-4 py-3 text-brand-text-muted">{m.club_functions?.includes('spieler') ? genderLabel(m.gender) : '–'}</td>
                <td className="hidden xl:table-cell px-4 py-3 text-brand-text-muted">{m.pass_number || '–'}</td>
                {m.can?.delete && (
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

            {/* Step 1: file + mode selection */}
            {!previewResult && !importResult && (
              <div className="px-6 py-5 space-y-5">
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">CSV-Datei</label>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".csv"
                    onChange={e => { setImportFile(e.target.files?.[0] ?? null); setPreviewResult(null) }}
                    className="block w-full text-sm text-brand-text-muted file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-sm file:font-medium file:bg-brand-yellow file:text-brand-black hover:file:bg-black hover:file:text-white cursor-pointer"
                  />
                  <p className="text-xs text-brand-text-subtle mt-1">Semikolon-getrennt, UTF-8. Spalten: Name, Vorname, Mitgliedsnummer, Email, Email 2, Geschlecht, Adresse, PLZ, Ort, Mitglied seit, Stammverein, Status, geboren am, SEPA Mandat, Kontoinhaber, IBAN</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-2">Modus</label>
                  <div className="space-y-2">
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input type="radio" name="importMode" value="append" checked={importMode === 'append'} onChange={() => setImportMode('append')} className="mt-0.5 accent-brand-yellow" />
                      <div>
                        <span className="text-sm font-medium text-brand-text">Nur neue anlegen</span>
                        <p className="text-xs text-brand-text-subtle">Neue Mitglieder anlegen, bestehende unverändert lassen (füllt keine fehlenden Felder bei Bestandsmitgliedern)</p>
                      </div>
                    </label>
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input type="radio" name="importMode" value="update" checked={importMode === 'update'} onChange={() => setImportMode('update')} className="mt-0.5 accent-brand-yellow" />
                      <div>
                        <span className="text-sm font-medium text-brand-text">Fehlende + geänderte Felder aktualisieren</span>
                        <p className="text-xs text-brand-text-subtle">Bestehende Mitglieder werden mit nicht-leeren CSV-Werten aktualisiert. Felder werden nie geleert.</p>
                      </div>
                    </label>
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input type="radio" name="importMode" value="enrich" checked={importMode === 'enrich'} onChange={() => setImportMode('enrich')} className="mt-0.5 accent-brand-yellow" />
                      <div>
                        <span className="text-sm font-medium text-brand-text">Nur leere Felder ergänzen</span>
                        <p className="text-xs text-brand-text-subtle">Keine neuen Mitglieder, keine Überschreibung. Nur leere DB-Felder werden aus der CSV befüllt.</p>
                      </div>
                    </label>
                  </div>
                </div>

                {(importMode === 'update' || importMode === 'enrich') && (
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <label className="block text-sm font-medium text-brand-text-muted">Felder aktualisieren</label>
                      <div className="flex gap-2 text-xs">
                        <button
                          type="button"
                          onClick={() => setSelectedFields(new Set(IMPORT_FIELDS.map(f => f.col)))}
                          className="text-brand-blue hover:underline"
                        >
                          Alle
                        </button>
                        <span className="text-brand-text-subtle">·</span>
                        <button
                          type="button"
                          onClick={() => setSelectedFields(new Set())}
                          className="text-brand-blue hover:underline"
                        >
                          Keine
                        </button>
                      </div>
                    </div>
                    <div className="grid grid-cols-2 gap-x-4 gap-y-1.5">
                      {IMPORT_FIELDS.map(f => (
                        <label key={f.col} className="flex items-center gap-2 cursor-pointer text-sm text-brand-text">
                          <input
                            type="checkbox"
                            checked={selectedFields.has(f.col)}
                            onChange={() => toggleField(f.col)}
                            className="accent-brand-yellow"
                          />
                          {f.label}
                        </label>
                      ))}
                    </div>
                    <p className="text-xs text-brand-text-subtle mt-2">Nur angehakte Spalten werden bei Bestandsmitgliedern aktualisiert. Neu angelegte Mitglieder bekommen alle Felder.</p>
                  </div>
                )}

                <div className="flex justify-end gap-2 pt-1">
                  <button onClick={resetImport} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                    Abbrechen
                  </button>
                  <button
                    onClick={handlePreview}
                    disabled={!importFile || importing}
                    className="px-4 py-2 text-sm bg-brand-yellow text-brand-black font-medium rounded-md hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >
                    {importing ? 'Analysiere…' : 'Vorschau'}
                  </button>
                </div>
              </div>
            )}

            {/* Step 2: preview report */}
            {previewResult && !importResult && (
              <div className="flex flex-col flex-1 min-h-0">
                <div className="px-6 py-4 border-b border-brand-border-subtle bg-brand-surface-card">
                  <p className="text-sm font-medium text-brand-text mb-1">{previewResult.total} Zeilen analysiert — Vorschau (nichts gespeichert)</p>
                  <div className="flex flex-wrap gap-3 text-xs">
                    {previewResult.created > 0 && <span className="text-brand-success font-medium">+ {previewResult.created} neu</span>}
                    {previewResult.updated > 0 && <span className="text-brand-blue font-medium">~ {previewResult.updated} Änderungen</span>}
                    {previewResult.unchanged > 0 && <span className="text-brand-text-subtle">= {previewResult.unchanged} unverändert</span>}
                    {(previewResult.not_found ?? 0) > 0 && <span className="text-brand-text-muted font-medium">— {previewResult.not_found} nicht gefunden</span>}
                    {previewResult.errors > 0 && <span className="text-brand-danger font-medium">✗ {previewResult.errors} Fehler</span>}
                    {previewResult.rows.some(r => r.iban_warning) && (
                      <span className="text-amber-600 font-medium flex items-center gap-1">
                        <AlertTriangle className="w-3 h-3" />{previewResult.rows.filter(r => r.iban_warning).length} IBAN-Warnungen
                      </span>
                    )}
                  </div>
                </div>
                <div className="overflow-y-auto flex-1 px-6 py-3 space-y-0.5 text-xs font-mono">
                  {previewResult.rows.filter(r => r.status !== 'unchanged').map((row, i) => {
                    const hasDetails = (row.changes && row.changes.length > 0) || row.message || row.iban_warning
                    const expanded = expandedRows.has(row.line)
                    const hasOw = rowHasOverwrites(row)
                    const selectable = row.status === 'updated'
                    return (
                      <div key={i} className={hasOw ? 'bg-amber-50 -mx-2 px-2 rounded' : ''}>
                        <div className="flex items-center gap-1">
                          {selectable ? (
                            <input
                              type="checkbox"
                              checked={selectedLines.has(row.line)}
                              onChange={() => toggleLine(row.line)}
                              aria-label={`Zeile ${row.line} anwenden`}
                              className="accent-brand-yellow shrink-0"
                            />
                          ) : (
                            <span className="w-3.5 shrink-0" />
                          )}
                          <button
                            onClick={() => hasDetails && toggleRow(row.line)}
                            className={`flex items-center gap-1 flex-1 text-left ${rowStatusColor(row.status)} ${hasDetails ? 'cursor-pointer hover:underline' : 'cursor-default'} ${selectable && !selectedLines.has(row.line) ? 'opacity-50' : ''}`}
                          >
                            {hasDetails && <ChevronRight className={`w-3 h-3 shrink-0 transition-transform ${expanded ? 'rotate-90' : ''}`} />}
                            {!hasDetails && <span className="w-3" />}
                            <span className="font-bold">{rowStatusIcon(row.status)}</span>
                            <span>Z.{row.line} {row.name}{row.dob ? ` (${row.dob.slice(0, 10)})` : ''}</span>
                            {hasOw && <AlertTriangle className="w-3 h-3 text-amber-500 ml-1" />}
                            {row.iban_warning && <AlertTriangle className="w-3 h-3 text-amber-500 ml-1" />}
                          </button>
                        </div>
                        {expanded && (
                          <div className="pl-7 space-y-0.5">
                            {row.changes?.map((c, j) => {
                              const ow = isOverwrite(c)
                              return (
                                <div key={j} className={`flex items-center gap-1 ${ow ? 'text-amber-600' : 'text-brand-text-muted'}`}>
                                  {ow && <AlertTriangle className="w-3 h-3 shrink-0" />}
                                  <span>{c}</span>
                                </div>
                              )
                            })}
                            {row.iban_warning && <div className="text-amber-600 flex items-center gap-1"><AlertTriangle className="w-3 h-3" />{row.iban_warning}</div>}
                            {row.message && <div className="italic text-brand-text-muted">{row.message}</div>}
                          </div>
                        )}
                      </div>
                    )
                  })}
                  {previewResult.unchanged > 0 && (
                    <div className="text-brand-text-subtle pt-1">= {previewResult.unchanged}× unverändert</div>
                  )}
                </div>
                <div className="px-6 py-4 border-t border-brand-border-subtle flex items-center justify-between gap-3">
                  <button onClick={() => setPreviewResult(null)} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                    Zurück
                  </button>
                  <div className="flex items-center gap-3">
                    {previewResult.updated > 0 && (
                      <div className="flex items-center gap-2 text-xs text-brand-text-muted">
                        <span>{selectedLines.size}/{previewResult.updated} ausgewählt</span>
                        <button
                          type="button"
                          onClick={() => setSelectedLines(new Set(previewResult.rows.filter(r => r.status === 'updated').map(r => r.line)))}
                          className="text-brand-blue hover:underline"
                        >
                          Alle
                        </button>
                        <span className="text-brand-text-subtle">·</span>
                        <button
                          type="button"
                          onClick={() => setSelectedLines(new Set())}
                          className="text-brand-blue hover:underline"
                        >
                          Keine
                        </button>
                      </div>
                    )}
                    <button
                      onClick={handleImport}
                      disabled={importing}
                      className="px-4 py-2 text-sm bg-brand-yellow text-brand-black font-medium rounded-md hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                    >
                      {importing ? 'Importieren…' : 'Jetzt anwenden'}
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Step 3: result */}
            {importResult && (
              <div className="flex flex-col flex-1 min-h-0">
                <div className="px-6 py-4 border-b border-brand-border-subtle bg-brand-surface-card">
                  <p className="text-sm font-medium text-brand-text mb-2">{importResult.total} Zeilen verarbeitet</p>
                  <div className="flex flex-wrap gap-3 text-xs">
                    {importResult.created > 0 && <span className="text-brand-success font-medium">+ {importResult.created} neu</span>}
                    {importResult.updated > 0 && <span className="text-brand-blue font-medium">~ {importResult.updated} aktualisiert</span>}
                    {(importResult.skipped ?? 0) > 0 && <span className="text-brand-text-muted font-medium">⊘ {importResult.skipped} übersprungen</span>}
                    {importResult.unchanged > 0 && <span className="text-brand-text-subtle">= {importResult.unchanged} unverändert</span>}
                    {(importResult.not_found ?? 0) > 0 && <span className="text-brand-text-muted font-medium">— {importResult.not_found} nicht gefunden</span>}
                    {importResult.errors > 0 && <span className="text-brand-danger font-medium">✗ {importResult.errors} Fehler</span>}
                  </div>
                </div>
                <div className="overflow-y-auto flex-1 px-6 py-3 space-y-0.5 text-xs font-mono">
                  {importResult.rows.filter(r => r.status !== 'unchanged').map((row, i) => {
                    const hasDetails = (row.changes && row.changes.length > 0) || row.message || row.iban_warning
                    const expanded = expandedRows.has(row.line)
                    const hasOw = rowHasOverwrites(row)
                    return (
                      <div key={i} className={hasOw ? 'bg-amber-50 -mx-2 px-2 rounded' : ''}>
                        <button
                          onClick={() => hasDetails && toggleRow(row.line)}
                          className={`flex items-center gap-1 w-full text-left ${rowStatusColor(row.status)} ${hasDetails ? 'cursor-pointer hover:underline' : 'cursor-default'}`}
                        >
                          {hasDetails && <ChevronRight className={`w-3 h-3 shrink-0 transition-transform ${expanded ? 'rotate-90' : ''}`} />}
                          {!hasDetails && <span className="w-3" />}
                          <span className="font-bold">{rowStatusIcon(row.status)}</span>
                          <span>Z.{row.line} {row.name}{row.dob ? ` (${row.dob.slice(0, 10)})` : ''}</span>
                          {hasOw && <AlertTriangle className="w-3 h-3 text-amber-500 ml-1" />}
                          {row.iban_warning && <AlertTriangle className="w-3 h-3 text-amber-500 ml-1" />}
                        </button>
                        {expanded && (
                          <div className="pl-7 space-y-0.5">
                            {row.changes?.map((c, j) => {
                              const ow = isOverwrite(c)
                              return (
                                <div key={j} className={`flex items-center gap-1 ${ow ? 'text-amber-600' : 'text-brand-text-muted'}`}>
                                  {ow && <AlertTriangle className="w-3 h-3 shrink-0" />}
                                  <span>{c}</span>
                                </div>
                              )
                            })}
                            {row.iban_warning && <div className="text-amber-600 flex items-center gap-1"><AlertTriangle className="w-3 h-3" />{row.iban_warning}</div>}
                            {row.message && <div className="italic text-brand-text-muted">{row.message}</div>}
                          </div>
                        )}
                      </div>
                    )
                  })}
                  {importResult.unchanged > 0 && (
                    <div className="text-brand-text-subtle pt-1">= {importResult.unchanged}× unverändert</div>
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
