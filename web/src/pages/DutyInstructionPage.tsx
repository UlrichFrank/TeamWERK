import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import MarkdownRenderer from '../components/MarkdownRenderer'

interface DutyTypeItem {
  id: number
  name: string
  instruction_md?: string
  instruction_updated_at?: string
}

export default function DutyInstructionPage() {
  const { typeId } = useParams<{ typeId: string }>()
  const [item, setItem] = useState<DutyTypeItem | null>(null)
  const [notFound, setNotFound] = useState(false)

  const load = useCallback(async () => {
    if (!typeId) return
    const idNum = parseInt(typeId, 10)
    const { data } = await api.get<DutyTypeItem[]>('/duty-types')
    const found = (data ?? []).find(t => t.id === idNum) ?? null
    setItem(found)
    setNotFound(!found)
  }, [typeId])

  useEffect(() => { load() }, [load])
  useLiveUpdates(event => { if (event === 'duties') load() })

  if (notFound) {
    return <p className="text-sm text-brand-text-muted">Diensttyp nicht gefunden.</p>
  }

  if (!item) {
    return <p className="text-sm text-brand-text-muted">Lade Anleitung…</p>
  }

  const updated = item.instruction_updated_at ? new Date(item.instruction_updated_at) : null
  const updatedLabel = updated
    ? updated.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' })
    : ''

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold mb-1">Anleitung: {item.name}</h1>
      {updatedLabel && (
        <p className="text-xs text-brand-text-muted mb-6">Zuletzt geändert am {updatedLabel}</p>
      )}

      {item.instruction_md ? (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <MarkdownRenderer markdown={item.instruction_md} />
        </div>
      ) : (
        <p className="text-sm text-brand-text-muted italic">Für diesen Dienst gibt es noch keine Anleitung.</p>
      )}
    </div>
  )
}
