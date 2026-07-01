import { useState, useEffect, useCallback, useRef } from 'react'
import { createPortal } from 'react-dom'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Folder, FileText, Upload, FolderPlus, Download, Trash2, Pencil,
  MoreVertical, ChevronRight, Lock, X, Check, AlertTriangle, Link2,
} from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { CLUB_FUNCTION_OPTIONS } from '../lib/constants'
import { errorMessage, errorStatus } from '../lib/errors'
import { useMediaQuery } from '../lib/useMediaQuery'
import { openBlobNatively } from '../lib/openFileNatively'

// ── Types ──────────────────────────────────────────────────────────────────

interface FolderItem {
  id: number
  name: string
  parent_id: number | null
  has_children: boolean
  can_read: boolean
  can_write: boolean
  created_at?: string
  created_by_name?: string
}

interface FileItem {
  id: number
  name: string
  size: number
  mime_type: string
  uploaded_by_name: string
  created_at: string
}

interface FolderContents {
  folders: FolderItem[]
  files: FileItem[]
  can_read: boolean
  can_write: boolean
}

interface Permission {
  id: number
  principal_type: string
  principal_ref: string
  display_name?: string
  can_read: boolean
  can_write: boolean
}

interface BreadcrumbEntry {
  id: number | null
  name: string
}

// ── Helpers ────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(iso: string): string {
  return iso.slice(0, 10)
}

// ── Sub-components ─────────────────────────────────────────────────────────

function NewFolderModal({ parentId, onCreated, onClose }: {
  parentId: number | null
  onCreated: () => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setSaving(true)
    try {
      await api.post('/folders', { name: name.trim(), parent_id: parentId })
      onCreated()
    } catch {
      setError('Ordner konnte nicht erstellt werden.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
        <div className="flex justify-between items-center mb-4">
          <h2 className="font-semibold text-brand-text">Neuer Ordner</h2>
          <button onClick={onClose} aria-label="Schließen"><X className="w-5 h-5 text-brand-text-muted" /></button>
        </div>
        <form onSubmit={submit} className="space-y-4">
          <input
            autoFocus
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Ordnername"
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
          {error && <p className="text-sm text-brand-danger">{error}</p>}
          <div className="flex justify-end gap-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text">Abbrechen</button>
            <button
              type="submit"
              disabled={saving || !name.trim()}
              className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Erstellen
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function UploadModal({ folderId, onUploaded, onClose }: {
  folderId: number
  onUploaded: () => void
  onClose: () => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  async function upload() {
    if (!file) return
    setError('')
    const formData = new FormData()
    formData.append('file', file)
    try {
      await api.post(`/folders/${folderId}/files`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: e => {
          if (e.total) setProgress(Math.round((e.loaded / e.total) * 100))
        },
      })
      setDone(true)
      onUploaded()
    } catch (e) {
      const status = errorStatus(e)
      if (status === 413) setError('Datei zu groß (max. 50 MB).')
      else if (status === 403) setError('Keine Berechtigung zum Hochladen.')
      else setError('Upload fehlgeschlagen.')
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
        <div className="flex justify-between items-center mb-4">
          <h2 className="font-semibold text-brand-text">Datei hochladen</h2>
          <button onClick={onClose} aria-label="Schließen"><X className="w-5 h-5 text-brand-text-muted" /></button>
        </div>
        <div className="space-y-4">
          {!done ? (
            <>
              <input
                ref={inputRef}
                type="file"
                className="hidden"
                onChange={e => setFile(e.target.files?.[0] ?? null)}
              />
              <button
                onClick={() => inputRef.current?.click()}
                className="w-full border-2 border-dashed border-brand-border rounded-lg py-6 text-sm text-brand-text-muted hover:border-brand-yellow transition-colors text-center"
              >
                {file ? file.name : 'Datei auswählen…'}
              </button>
              {progress > 0 && progress < 100 && (
                <div className="w-full bg-brand-border-subtle rounded-full h-2">
                  <div className="bg-brand-yellow h-2 rounded-full transition-all" style={{ width: `${progress}%` }} />
                </div>
              )}
              {error && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>
              )}
              <div className="flex justify-end gap-2">
                <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text">Abbrechen</button>
                <button
                  onClick={upload}
                  disabled={!file}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  <Upload className="w-4 h-4 inline mr-1" />Hochladen
                </button>
              </div>
            </>
          ) : (
            <div className="text-center space-y-3 py-4">
              <Check className="w-8 h-8 text-green-600 mx-auto" />
              <p className="text-sm text-brand-text">Datei erfolgreich hochgeladen.</p>
              <button onClick={onClose} className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">Schließen</button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

const ROLE_OPTIONS = ['admin', 'standard']

const PRINCIPAL_TYPE_LABELS: Record<string, string> = {
  everyone: 'Alle Nutzer',
  role: 'Rolle',
  club_function: 'Vereinsfunktion',
  user: 'Person',
}

function PermissionsModal({ folderId, canWrite, onClose }: {
  folderId: number
  canWrite: boolean
  onClose: () => void
}) {
  const [perms, setPerms] = useState<Permission[]>([])
  const [loading, setLoading] = useState(true)
  const [newType, setNewType] = useState('everyone')
  const [newRef, setNewRef] = useState('')
  const [newRead, setNewRead] = useState(true)
  const [newWrite, setNewWrite] = useState(false)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [pickerUsers, setPickerUsers] = useState<{ id: number; name: string }[]>([])
  const [pickerLoaded, setPickerLoaded] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const { data } = await api.get<Permission[]>(`/folders/${folderId}/permissions`)
      setPerms(data)
    } finally {
      setLoading(false)
    }
  }, [folderId])

  // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
  useEffect(() => { load() }, [load])

  async function loadPickerUsers() {
    if (pickerLoaded) return
    try {
      const { data } = await api.get<{ id: number; name: string }[]>('/users/picker')
      setPickerUsers(data)
    } catch {
      // ignore — picker stays empty
    } finally {
      setPickerLoaded(true)
    }
  }

  async function addPerm(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.post(`/folders/${folderId}/permissions`, {
        principal_type: newType,
        principal_ref: newType === 'everyone' ? '' : newRef,
        can_read: newRead,
        can_write: newWrite,
      })
      setNewType('everyone')
      setNewRef('')
      setNewRead(true)
      setNewWrite(false)
      load()
    } catch (e) {
      if (errorStatus(e) === 403) setError('Du kannst keine höheren Rechte vergeben als du selbst hast.')
      else setError('Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  async function removePerm(permId: number) {
    await api.delete(`/folders/${folderId}/permissions/${permId}`)
    load()
  }

  function permLabel(p: Permission): string {
    if (p.principal_type === 'everyone') return 'Alle Nutzer'
    const ref = p.principal_type === 'user' ? (p.display_name ?? p.principal_ref) : p.principal_ref
    return `${PRINCIPAL_TYPE_LABELS[p.principal_type] ?? p.principal_type}: ${ref}`
  }

  return (
    <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
        <div className="flex justify-between items-center mb-4">
          <h2 className="font-semibold text-brand-text flex items-center gap-2">
            <Lock className="w-4 h-4" />Berechtigungen
          </h2>
          <button onClick={onClose} aria-label="Schließen"><X className="w-5 h-5 text-brand-text-muted" /></button>
        </div>

        {loading ? (
          <p className="text-sm text-brand-text-muted">Laden…</p>
        ) : (
          <div className="space-y-2 mb-5">
            {perms.length === 0 && <p className="text-sm text-brand-text-muted italic">Keine direkten Berechtigungen.</p>}
            {perms.map(p => (
              <div key={p.id} className="flex items-center justify-between bg-brand-surface-card rounded-lg px-3 py-2 text-sm">
                <div>
                  <span className="text-brand-text">{permLabel(p)}</span>
                  <span className="text-brand-text-muted ml-2">
                    {p.can_read && 'Lesen'}{p.can_read && p.can_write && ' + '}{p.can_write && 'Schreiben'}
                  </span>
                </div>
                {canWrite && (
                  <button onClick={() => removePerm(p.id)} aria-label="Entfernen" className="text-brand-danger hover:opacity-70 ml-2">
                    <X className="w-4 h-4" />
                  </button>
                )}
              </div>
            ))}
          </div>
        )}

        {canWrite && (
          <form onSubmit={addPerm} className="border-t border-brand-border-subtle pt-4 space-y-3">
            <p className="text-xs font-medium text-brand-text-muted uppercase tracking-wide">Berechtigung hinzufügen</p>
            <select
              value={newType}
              onChange={e => {
                const t = e.target.value
                setNewType(t)
                setNewRef('')
                if (t === 'user') loadPickerUsers()
              }}
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow"
            >
              {Object.entries(PRINCIPAL_TYPE_LABELS).map(([v, l]) => (
                <option key={v} value={v}>{l}</option>
              ))}
            </select>

            {newType === 'role' && (
              <select
                value={newRef}
                onChange={e => setNewRef(e.target.value)}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              >
                <option value="">Rolle wählen…</option>
                {ROLE_OPTIONS.map(r => <option key={r} value={r}>{r}</option>)}
              </select>
            )}
            {newType === 'club_function' && (
              <select
                value={newRef}
                onChange={e => setNewRef(e.target.value)}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              >
                <option value="">Funktion wählen…</option>
                {CLUB_FUNCTION_OPTIONS.map(f => <option key={f.value} value={f.value}>{f.label}</option>)}
              </select>
            )}
            {newType === 'user' && (
              <select
                value={newRef}
                onChange={e => setNewRef(e.target.value)}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              >
                <option value="">Person wählen…</option>
                {pickerUsers.map(u => (
                  <option key={u.id} value={String(u.id)}>{u.name}</option>
                ))}
              </select>
            )}

            <div className="flex gap-4 text-sm">
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={newRead} onChange={e => setNewRead(e.target.checked)} className="accent-brand-yellow" />
                <span className="text-brand-text">Lesen</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={newWrite} onChange={e => setNewWrite(e.target.checked)} className="accent-brand-yellow" />
                <span className="text-brand-text">Schreiben</span>
              </label>
            </div>

            {error && <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>}

            <button
              type="submit"
              disabled={saving || (!newRead && !newWrite) || (newType !== 'everyone' && !newRef)}
              className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Hinzufügen
            </button>
          </form>
        )}
      </div>
    </div>
  )
}

function ActionMenu({ items }: { items: { label: string; icon: React.ReactNode; danger?: boolean; onClick: () => void }[] }) {
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState({ top: 0, right: 0 })
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (!buttonRef.current?.contains(e.target as Node) && !menuRef.current?.contains(e.target as Node))
        setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  function toggle(e: React.MouseEvent) {
    e.stopPropagation()
    if (buttonRef.current) {
      const rect = buttonRef.current.getBoundingClientRect()
      setPos({ top: rect.bottom + 4, right: window.innerWidth - rect.right })
    }
    setOpen(o => !o)
  }

  return (
    <>
      <button
        ref={buttonRef}
        onClick={toggle}
        aria-label="Aktionen"
        className="p-1 rounded hover:bg-brand-table-select transition-colors"
      >
        <MoreVertical className="w-4 h-4 text-brand-text-muted" />
      </button>
      {open && createPortal(
        <div
          ref={menuRef}
          style={{ position: 'fixed', top: pos.top, right: pos.right, zIndex: 9999 }}
          className="bg-white border border-brand-border-subtle rounded-lg shadow-lg min-w-[140px] py-1"
        >
          {items.map(item => (
            <button
              key={item.label}
              onClick={() => { item.onClick(); setOpen(false) }}
              className={`w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-brand-surface-card transition-colors ${item.danger ? 'text-brand-danger' : 'text-brand-text'}`}
            >
              {item.icon}{item.label}
            </button>
          ))}
        </div>,
        document.body
      )}
    </>
  )
}

function RenameModal({ type, id, currentName, onRenamed, onClose }: {
  type: 'folder' | 'file'
  id: number
  currentName: string
  onRenamed: () => void
  onClose: () => void
}) {
  const [name, setName] = useState(currentName)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || name.trim() === currentName) { onClose(); return }
    setSaving(true)
    try {
      const url = type === 'folder' ? `/folders/${id}` : `/files/${id}`
      await api.put(url, { name: name.trim() })
      onRenamed()
    } catch {
      setError('Umbenennen fehlgeschlagen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
        <div className="flex justify-between items-center mb-4">
          <h2 className="font-semibold text-brand-text">Umbenennen</h2>
          <button onClick={onClose} aria-label="Schließen"><X className="w-5 h-5 text-brand-text-muted" /></button>
        </div>
        <form onSubmit={submit} className="space-y-4">
          <input
            autoFocus
            value={name}
            onChange={e => setName(e.target.value)}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
          {error && <p className="text-sm text-brand-danger">{error}</p>}
          <div className="flex justify-end gap-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text">Abbrechen</button>
            <button
              type="submit"
              disabled={saving || !name.trim()}
              className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ── Main Page ──────────────────────────────────────────────────────────────

export default function DocumentsPage() {
  const { folderId: folderIdParam } = useParams()
  const navigate = useNavigate()
  const { hasCapability } = useAuth()
  const isMobile = useMediaQuery('(max-width: 639px)')

  const currentFolderId = folderIdParam ? parseInt(folderIdParam) : null

  const [rootFolders, setRootFolders] = useState<FolderItem[]>([])
  const [contents, setContents] = useState<FolderContents | null>(null)
  const [breadcrumb, setBreadcrumb] = useState<BreadcrumbEntry[]>([{ id: null, name: 'Dokumente' }])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [showNewFolder, setShowNewFolder] = useState(false)
  const [showUpload, setShowUpload] = useState(false)
  const [showPermissions, setShowPermissions] = useState<number | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<{ type: 'folder' | 'file'; id: number; name: string } | null>(null)
  const [renaming, setRenaming] = useState<{ type: 'folder' | 'file'; id: number; name: string } | null>(null)

  const loadRoot = useCallback(async () => {
    try {
      const { data } = await api.get<FolderItem[]>('/folders')
      setRootFolders(data)
    } catch { /* ignore */ }
  }, [])

  const loadContents = useCallback(async (id: number) => {
    setLoading(true)
    setError('')
    try {
      const { data } = await api.get<FolderContents>(`/folders/${id}/contents`)
      setContents(data)
    } catch (e) {
      if (errorStatus(e) === 403) setError('Kein Zugriff auf diesen Ordner.')
      else setError('Fehler beim Laden.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    loadRoot()
  }, [loadRoot])

  useEffect(() => {
    if (currentFolderId) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
      loadContents(currentFolderId)
    } else {
      setContents(null)
      setLoading(false)
    }
  }, [currentFolderId, loadContents])

  function navigateTo(folder: FolderItem) {
    setBreadcrumb(prev => {
      const exists = prev.findIndex(e => e.id === folder.id)
      if (exists >= 0) return prev.slice(0, exists + 1)
      return [...prev, { id: folder.id, name: folder.name }]
    })
    navigate(`/dokumente/${folder.id}`)
  }

  function navigateToBreadcrumb(entry: BreadcrumbEntry) {
    if (entry.id === null) {
      setBreadcrumb([{ id: null, name: 'Dokumente' }])
      navigate('/dokumente')
    } else {
      const idx = breadcrumb.findIndex(e => e.id === entry.id)
      setBreadcrumb(breadcrumb.slice(0, idx + 1))
      navigate(`/dokumente/${entry.id}`)
    }
  }

  const [fileError, setFileError] = useState('')
  const [linkToast, setLinkToast] = useState('')

  async function copyFileLink(file: FileItem) {
    const url = `${window.location.origin}/dokumente/datei/${file.id}`
    try {
      await navigator.clipboard.writeText(url)
      setLinkToast('Link kopiert')
    } catch {
      setLinkToast(url)
    }
    setTimeout(() => setLinkToast(''), 2500)
  }

  async function openFile(file: FileItem) {
    setFileError('')
    if (!isMobile) {
      // Desktop: nativer Browser-PDF-Viewer in neuem Tab. window.open MUSS
      // synchron im Click-Handler stehen (Popup-Blocker); URL wird nach
      // Token-Fetch nachgereicht.
      const tab = window.open('about:blank', '_blank')
      try {
        const { data } = await api.get<{ token: string }>(`/files/${file.id}/download-token`)
        if (tab) tab.location.href = `/api/files/${file.id}/download?token=${data.token}`
      } catch (e) {
        if (tab) tab.close()
        const status = errorStatus(e)
        if (status === 403) setFileError('Du hast keinen Zugriff auf diese Datei.')
        else if (status === 404) setFileError('Datei nicht gefunden.')
        else setFileError('Datei konnte nicht geöffnet werden.')
      }
      return
    }
    // Mobile: den schwachen In-App-Render überspringen und die Datei direkt im
    // nativen Viewer öffnen (zoomen/scrollen/schließen).
    try {
      const { data } = await api.get<{ token: string }>(`/files/${file.id}/download-token`)
      const res = await api.get<Blob>(`/files/${file.id}/download?token=${data.token}`, {
        responseType: 'blob',
      })
      openBlobNatively(res.data, file.name)
    } catch (e) {
      const status = errorStatus(e)
      if (status === 403) setFileError('Du hast keinen Zugriff auf diese Datei.')
      else if (status === 404) setFileError('Datei nicht gefunden.')
      else setFileError('Datei konnte nicht geöffnet werden.')
    }
  }

  async function deleteItem(type: 'folder' | 'file', id: number) {
    try {
      if (type === 'folder') {
        await api.delete(`/folders/${id}`)
        loadRoot()
      } else {
        await api.delete(`/files/${id}`)
      }
      if (currentFolderId) loadContents(currentFolderId)
    } catch (e) {
      alert(errorMessage(e, 'Löschen fehlgeschlagen.'))
    } finally {
      setConfirmDelete(null)
    }
  }

  const isAdmin = hasCapability('manage_documents')
  const canWrite = isAdmin || (contents?.can_write ?? false)
  const displayFolders = currentFolderId ? (contents?.folders ?? []) : rootFolders
  const displayFiles = currentFolderId ? (contents?.files ?? []) : []

  return (
    <div>
      {/* Header — same sticky pattern as MembersPage */}
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold text-brand-text">Dokumente</h1>
            {/* Breadcrumb only when inside a subfolder */}
            {breadcrumb.length > 1 && (
              <nav className="flex items-center gap-1 flex-wrap text-sm mt-1">
                {breadcrumb.map((entry, i) => (
                  <span key={i} className="flex items-center gap-1">
                    {i > 0 && <ChevronRight className="w-3 h-3 text-brand-text-muted" />}
                    <button
                      onClick={() => navigateToBreadcrumb(entry)}
                      className={i === breadcrumb.length - 1 ? 'font-medium text-brand-text' : 'text-brand-text-muted hover:text-brand-text'}
                    >
                      {entry.name}
                    </button>
                  </span>
                ))}
              </nav>
            )}
          </div>
          {canWrite && (
            <div className="flex gap-2">
              <button
                onClick={() => setShowNewFolder(true)}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-1.5 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center gap-1"
              >
                <FolderPlus className="w-4 h-4" />
                <span className="hidden sm:inline">Neuer Ordner</span>
              </button>
              {currentFolderId && (
                <button
                  onClick={() => setShowUpload(true)}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-1.5 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center gap-1"
                >
                  <Upload className="w-4 h-4" />
                  <span className="hidden sm:inline">Hochladen</span>
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4 flex items-center gap-2">
          <AlertTriangle className="w-4 h-4" />{error}
        </div>
      )}
      {fileError && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4 flex items-center gap-2">
          <AlertTriangle className="w-4 h-4" />{fileError}
        </div>
      )}

      <div className="mt-4">
          {loading ? (
            <p className="text-sm text-brand-text-muted">Laden…</p>
          ) : (
            <>
              {/* Mobile: single card with list */}
              <div className="sm:hidden bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
                {displayFolders.length === 0 && displayFiles.length === 0 ? (
                  <p className="text-sm text-brand-text-muted italic px-4 py-6 text-center">Dieser Ordner ist leer.</p>
                ) : (
                  <div className="divide-y divide-brand-border-subtle">
                    {displayFolders.map(folder => (
                      <div key={folder.id} className="flex items-center hover:bg-brand-table-select transition-colors">
                        <button
                          onClick={() => navigateTo(folder)}
                          className="flex-1 flex items-center gap-3 px-4 py-3 text-left min-w-0"
                        >
                          <Folder className="w-5 h-5 text-brand-text-muted flex-shrink-0" />
                          <p className="text-sm font-medium text-brand-text truncate">{folder.name}</p>
                        </button>
                        <div className="pr-2">
                          <ActionMenu items={[
                            { label: 'Öffnen', icon: <Folder className="w-4 h-4" />, onClick: () => navigateTo(folder) },
                            ...(folder.can_write ? [{ label: 'Umbenennen', icon: <Pencil className="w-4 h-4" />, onClick: () => setRenaming({ type: 'folder', id: folder.id, name: folder.name }) }] : []),
                            ...(folder.can_write ? [{ label: 'Berechtigungen', icon: <Lock className="w-4 h-4" />, onClick: () => setShowPermissions(folder.id) }] : []),
                            ...(folder.can_write ? [{ label: 'Löschen', icon: <Trash2 className="w-4 h-4" />, danger: true, onClick: () => setConfirmDelete({ type: 'folder', id: folder.id, name: folder.name }) }] : []),
                          ]} />
                        </div>
                      </div>
                    ))}
                    {displayFiles.map(file => (
                      <div key={file.id} className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-brand-table-select transition-colors" onClick={() => openFile(file)}>
                        <FileText className="w-5 h-5 text-brand-text-muted flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-brand-text truncate">{file.name}</p>
                          <p className="text-xs text-brand-text-muted">{formatBytes(file.size)} · {formatDate(file.created_at)}</p>
                        </div>
                        <span onClick={e => e.stopPropagation()}>
                          <ActionMenu items={[
                            { label: 'Herunterladen', icon: <Download className="w-4 h-4" />, onClick: () => openFile(file) },
                            { label: 'Link kopieren', icon: <Link2 className="w-4 h-4" />, onClick: () => copyFileLink(file) },
                            ...(canWrite ? [{ label: 'Umbenennen', icon: <Pencil className="w-4 h-4" />, onClick: () => setRenaming({ type: 'file', id: file.id, name: file.name }) }] : []),
                            ...(canWrite ? [{ label: 'Löschen', icon: <Trash2 className="w-4 h-4" />, danger: true, onClick: () => setConfirmDelete({ type: 'file', id: file.id, name: file.name }) }] : []),
                          ]} />
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Desktop: Table */}
              <div className="hidden sm:block bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
                {displayFolders.length === 0 && displayFiles.length === 0 ? (
                  <p className="text-sm text-brand-text-muted italic px-4 py-6 text-center">Dieser Ordner ist leer.</p>
                ) : (
                  <table className="w-full">
                    <thead>
                      <tr>
                        <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
                        <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Größe</th>
                        <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Datum</th>
                        <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Hochgeladen von</th>
                        <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-right w-20">Aktionen</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-brand-border-subtle">
                      {displayFolders.map(folder => (
                        <tr key={`f-${folder.id}`} className="hover:bg-brand-table-select transition-colors cursor-pointer" onClick={() => navigateTo(folder)}>
                          <td className="px-4 py-3 text-sm text-brand-text">
                            <span className="flex items-center gap-2">
                              <Folder className="w-4 h-4 text-brand-text-muted" />{folder.name}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm text-brand-text-muted">—</td>
                          <td className="px-4 py-3 text-sm text-brand-text-muted">{folder.created_at ? formatDate(folder.created_at) : '—'}</td>
                          <td className="px-4 py-3 text-sm text-brand-text-muted">{folder.created_by_name ?? '—'}</td>
                          <td className="px-4 py-3 text-right">
                            {folder.can_write && (
                              <ActionMenu items={[
                                { label: 'Umbenennen', icon: <Pencil className="w-4 h-4" />, onClick: () => setRenaming({ type: 'folder', id: folder.id, name: folder.name }) },
                                { label: 'Berechtigungen', icon: <Lock className="w-4 h-4" />, onClick: () => setShowPermissions(folder.id) },
                                { label: 'Löschen', icon: <Trash2 className="w-4 h-4" />, danger: true, onClick: () => setConfirmDelete({ type: 'folder', id: folder.id, name: folder.name }) },
                              ]} />
                            )}
                          </td>
                        </tr>
                      ))}
                      {displayFiles.map(file => (
                        <tr key={`d-${file.id}`} className="hover:bg-brand-table-select transition-colors cursor-pointer" onClick={() => openFile(file)}>
                          <td className="px-4 py-3 text-sm text-brand-text">
                            <span className="flex items-center gap-2">
                              <FileText className="w-4 h-4 text-brand-text-muted" />{file.name}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm text-brand-text-muted">{formatBytes(file.size)}</td>
                          <td className="px-4 py-3 text-sm text-brand-text-muted">{formatDate(file.created_at)}</td>
                          <td className="px-4 py-3 text-sm text-brand-text-muted">{file.uploaded_by_name}</td>
                          <td className="px-4 py-3 text-right" onClick={e => e.stopPropagation()}>
                            <ActionMenu items={[
                              { label: 'Herunterladen', icon: <Download className="w-4 h-4" />, onClick: () => openFile(file) },
                              { label: 'Link kopieren', icon: <Link2 className="w-4 h-4" />, onClick: () => copyFileLink(file) },
                              ...(canWrite ? [{ label: 'Umbenennen', icon: <Pencil className="w-4 h-4" />, onClick: () => setRenaming({ type: 'file', id: file.id, name: file.name }) }] : []),
                              ...(canWrite ? [{ label: 'Löschen', icon: <Trash2 className="w-4 h-4" />, danger: true, onClick: () => setConfirmDelete({ type: 'file', id: file.id, name: file.name }) }] : []),
                            ]} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </>
          )}
      </div>

      {/* Modals */}
      {showNewFolder && (
        <NewFolderModal
          parentId={currentFolderId}
          onCreated={() => { setShowNewFolder(false); loadRoot(); if (currentFolderId) loadContents(currentFolderId) }}
          onClose={() => setShowNewFolder(false)}
        />
      )}
      {showUpload && currentFolderId && (
        <UploadModal
          folderId={currentFolderId}
          onUploaded={() => loadContents(currentFolderId!)}
          onClose={() => setShowUpload(false)}
        />
      )}
      {showPermissions !== null && (
        <PermissionsModal
          folderId={showPermissions}
          canWrite={canWrite}
          onClose={() => setShowPermissions(null)}
        />
      )}

      {renaming && (
        <RenameModal
          type={renaming.type}
          id={renaming.id}
          currentName={renaming.name}
          onRenamed={() => {
            setRenaming(null)
            if (renaming.type === 'folder') loadRoot()
            if (currentFolderId) loadContents(currentFolderId)
          }}
          onClose={() => setRenaming(null)}
        />
      )}

      {linkToast && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-brand-black text-brand-white text-sm px-4 py-2 rounded-lg shadow-lg z-50">
          {linkToast}
        </div>
      )}

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
            <h2 className="font-semibold text-brand-text mb-2">Löschen bestätigen</h2>
            <p className="text-sm text-brand-text-muted mb-4">„{confirmDelete.name}" wirklich löschen?</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setConfirmDelete(null)} className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text">Abbrechen</button>
              <button
                onClick={() => deleteItem(confirmDelete.type, confirmDelete.id)}
                className="bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors"
              >
                Löschen
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
