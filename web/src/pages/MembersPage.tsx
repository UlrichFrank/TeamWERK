import { useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { usePaginatedFetch } from '../lib/usePaginatedFetch'
import MobileCard from '../components/MobileCard'

interface Member {
  id: number; first_name: string; last_name: string
  status: string; pass_number?: string; position?: string; gender?: string
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
  return 'bg-gray-200 text-gray-600'
}

const rowStatusIcon = (s: ImportRow['status']) => {
  if (s === 'created') return '+'
  if (s === 'updated') return '~'
  if (s === 'unchanged') return '='
  return '✗'
}

const rowStatusColor = (s: ImportRow['status']) => {
  if (s === 'created') return 'text-green-700'
  if (s === 'updated') return 'text-brand-blue'
  if (s === 'unchanged') return 'text-gray-400'
  return 'text-red-600'
}

export default function MembersPage() {
  const { user } = useAuth()
  const { items, total, loading, setSearch, loadMore, reset } = usePaginatedFetch<Member>('/members')
  const isAdmin = user?.role === 'admin'

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
      if (res.data.created > 0 || res.data.updated > 0) reset()
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
              className="border border-gray-300 rounded-md px-3 py-2.5 sm:py-1.5 text-sm w-full sm:w-auto"
            />
            {isAdmin && (
              <>
                <Link
                  to="/mitglieder/neu"
                  className="text-sm bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-2.5 sm:py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors text-center"
                >
                  + Neu
                </Link>
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

      {/* Mobile: Cards */}
      <div className="sm:hidden space-y-0">
        {items.map(m => (
          <Link key={m.id} to={`/mitglieder/${m.id}`} className="block">
            <MobileCard
              title={`${m.last_name}, ${m.first_name}`}
              subtitle={m.position || '–'}
              badge={{ label: m.status, variant: m.status === 'aktiv' ? 'blue' : m.status === 'verletzt' ? 'yellow' : 'red' }}
            />
          </Link>
        ))}
      </div>

      {/* Desktop: Table */}
      <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              <th className="px-4 py-3 text-left">Name</th>
              <th className="px-4 py-3 text-left">Passnummer</th>
              <th className="px-4 py-3 text-left">Gesch.</th>
              <th className="px-4 py-3 text-left">Position</th>
              <th className="px-4 py-3 text-left">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {items.map(m => (
              <tr key={m.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium">
                  <Link to={`/mitglieder/${m.id}`} className="hover:text-brand-yellow transition-colors">
                    {m.last_name}, {m.first_name}
                  </Link>
                </td>
                <td className="px-4 py-3 text-gray-500">{m.pass_number || '–'}</td>
                <td className="px-4 py-3 text-gray-500">{genderLabel(m.gender)}</td>
                <td className="px-4 py-3 text-gray-500">{m.position || '–'}</td>
                <td className="px-4 py-3">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeStyles(m.status)}`}>
                    {m.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Load More Button */}
      {items.length < total && (
        <div className="mt-6 text-center">
          <button
            onClick={loadMore}
            disabled={loading}
            className="px-6 py-2.5 sm:py-2 bg-brand-yellow text-brand-black rounded font-medium hover:bg-brand-black hover:text-brand-yellow disabled:opacity-50 transition-colors"
          >
            {loading ? 'Lädt…' : `Mehr laden (${items.length}/${total})`}
          </button>
        </div>
      )}

      {/* Import Modal */}
      {showImport && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
            <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
              <h2 className="font-semibold text-base">CSV-Import</h2>
              <button onClick={resetImport} className="text-gray-400 hover:text-gray-600 text-xl leading-none">×</button>
            </div>

            {!importResult ? (
              <div className="px-6 py-5 space-y-5">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">CSV-Datei</label>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".csv"
                    onChange={e => setImportFile(e.target.files?.[0] ?? null)}
                    className="block w-full text-sm text-gray-600 file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-sm file:font-medium file:bg-brand-yellow file:text-brand-black hover:file:bg-black hover:file:text-white cursor-pointer"
                  />
                  <p className="text-xs text-gray-400 mt-1">Erwartet: Semikolon-getrennt, UTF-8. Spalten: Mitgliedsnummer, Vorname, Nachname, Geburtsdatum, Geschlecht, Passnummer, Trikotnummer, Position, Status, Benutzer_Email, Erziehungsberechtigter1_Email, Erziehungsberechtigter2_Email</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Modus</label>
                  <div className="space-y-2">
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="importMode"
                        value="append"
                        checked={importMode === 'append'}
                        onChange={() => setImportMode('append')}
                        className="mt-0.5"
                      />
                      <div>
                        <span className="text-sm font-medium">Nur ergänzen</span>
                        <p className="text-xs text-gray-400">Neue Mitglieder anlegen, bestehende unverändert lassen</p>
                      </div>
                    </label>
                    <label className="flex items-start gap-2 cursor-pointer">
                      <input
                        type="radio"
                        name="importMode"
                        value="update"
                        checked={importMode === 'update'}
                        onChange={() => setImportMode('update')}
                        className="mt-0.5"
                      />
                      <div>
                        <span className="text-sm font-medium">Fehlende + geänderte Felder aktualisieren</span>
                        <p className="text-xs text-gray-400">Bestehende Mitglieder werden mit nicht-leeren CSV-Werten aktualisiert. Felder werden nie geleert.</p>
                      </div>
                    </label>
                  </div>
                </div>

                <div className="flex justify-end gap-2 pt-1">
                  <button onClick={resetImport} className="px-4 py-2 text-sm border border-gray-300 rounded-md hover:border-gray-500 transition-colors">
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
                <div className="px-6 py-4 border-b border-gray-100 bg-gray-50">
                  <p className="text-sm font-medium mb-2">{importResult.total} Zeilen verarbeitet</p>
                  <div className="flex flex-wrap gap-3 text-xs">
                    {importResult.created > 0 && <span className="text-green-700 font-medium">+ {importResult.created} neu</span>}
                    {importResult.updated > 0 && <span className="text-brand-blue font-medium">~ {importResult.updated} aktualisiert</span>}
                    {importResult.unchanged > 0 && <span className="text-gray-400">= {importResult.unchanged} unverändert</span>}
                    {importResult.errors > 0 && <span className="text-red-600 font-medium">✗ {importResult.errors} Fehler</span>}
                  </div>
                </div>

                {/* Detail rows */}
                <div className="overflow-y-auto flex-1 px-6 py-3 space-y-1 text-xs font-mono">
                  {importResult.rows.filter(r => r.status !== 'unchanged').map((row, i) => (
                    <div key={i} className={rowStatusColor(row.status)}>
                      <span className="font-bold">{rowStatusIcon(row.status)}</span>{' '}
                      <span>Z.{row.line} {row.name}{row.dob ? ` (${row.dob.slice(0, 10)})` : ''}</span>
                      {row.changes?.map((c, j) => (
                        <div key={j} className="pl-4 text-gray-500">{c}</div>
                      ))}
                      {row.message && <span className="pl-4 italic">{row.message}</span>}
                    </div>
                  ))}
                  {importResult.unchanged > 0 && (
                    <div className="text-gray-400">= {importResult.unchanged}× unverändert</div>
                  )}
                </div>

                <div className="px-6 py-4 border-t border-gray-100 flex justify-end">
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
