import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { useEscapeKey } from '../lib/useEscapeKey'

interface Entry {
  type: 'feat' | 'fix'
  scope: string
  message: string
}

interface Group {
  date: string
  entries: Entry[]
}

function parseChangelog(raw: string): Group[] {
  const groups: Group[] = []
  let current: Group | null = null
  for (const line of raw.split('\n')) {
    const dateMatch = line.match(/^##\s+(.+)$/)
    if (dateMatch) {
      current = { date: dateMatch[1].trim(), entries: [] }
      groups.push(current)
      continue
    }
    const entryMatch = line.match(/^-\s+\[(feat|fix)\]\s+([^:]+):\s+(.+)$/)
    if (entryMatch && current) {
      current.entries.push({
        type: entryMatch[1] as 'feat' | 'fix',
        scope: entryMatch[2].trim(),
        message: entryMatch[3].trim(),
      })
    }
  }
  return groups
}

interface Props {
  onClose: () => void
}

export default function ChangelogModal({ onClose }: Props) {
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEscapeKey(onClose)

  useEffect(() => {
    fetch('/CHANGELOG.md')
      .then(r => {
        if (!r.ok) throw new Error()
        return r.text()
      })
      .then(text => setGroups(parseChangelog(text)))
      .catch(() => setError(true))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-end sm:items-center justify-center z-50 p-0 sm:p-4">
      <div className="bg-white rounded-t-xl sm:rounded-xl shadow-xl border-t-4 border-brand-yellow w-full sm:max-w-lg max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle shrink-0">
          <h2 className="text-base font-bold text-brand-text">Versionshistorie</h2>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-brand-border-subtle transition-colors"
            aria-label="Schließen"
          >
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <div className="overflow-y-auto flex-1 px-6 py-4">
          {loading && (
            <p className="text-sm text-brand-text-muted text-center py-8">Lädt…</p>
          )}
          {error && (
            <p className="text-sm text-brand-danger text-center py-8">Versionshistorie konnte nicht geladen werden.</p>
          )}
          {!loading && !error && groups.map(group => (
            <div key={group.date} className="mb-5 last:mb-0">
              <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">{group.date}</p>
              <ul className="space-y-1.5">
                {group.entries.map((entry, i) => (
                  <li
                    key={i}
                    className="grid grid-cols-[2.5rem_6rem_1fr] sm:grid-cols-[2.5rem_8rem_1fr] items-start gap-2 text-sm"
                  >
                    <span className={`mt-0.5 rounded px-1.5 py-0.5 text-xs font-semibold text-center ${
                      entry.type === 'feat'
                        ? 'bg-brand-yellow/30 text-brand-black'
                        : 'bg-brand-danger-light text-brand-danger'
                    }`}>
                      {entry.type}
                    </span>
                    <span
                      className="text-brand-text-muted truncate"
                      title={entry.scope}
                    >
                      {entry.scope}
                    </span>
                    <span className="text-brand-text break-words">{entry.message}</span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
