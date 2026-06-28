import { Clock, RefreshCw, CheckCircle, AlertTriangle } from 'lucide-react'

export type VideoStatus = 'uploading' | 'queued' | 'processing' | 'ready' | 'failed' | string

interface PillConfig {
  label: string
  className: string
  Icon: typeof Clock
  spin?: boolean
}

// Status-Darstellung für Spielvideos. Nur brand-*-Tokens, lucide-Icons.
const CONFIG: Record<string, PillConfig> = {
  uploading: { label: 'Wird hochgeladen', className: 'bg-brand-border-subtle text-brand-text-muted', Icon: Clock },
  queued: { label: 'In Warteschlange', className: 'bg-brand-border-subtle text-brand-text-muted', Icon: Clock },
  processing: { label: 'Wird verarbeitet', className: 'bg-brand-blue/10 text-brand-blue', Icon: RefreshCw, spin: true },
  ready: { label: 'Bereit', className: 'bg-brand-green/10 text-brand-green', Icon: CheckCircle },
  failed: { label: 'Fehlgeschlagen', className: 'bg-brand-danger-light text-brand-danger', Icon: AlertTriangle },
}

export default function VideoStatusPill({ status }: { status: VideoStatus }) {
  const cfg = CONFIG[status] ?? {
    label: status,
    className: 'bg-brand-border-subtle text-brand-text-muted',
    Icon: Clock,
  }
  const { label, className, Icon, spin } = cfg
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${className}`}>
      <Icon className={`w-3 h-3${spin ? ' animate-spin' : ''}`} aria-hidden="true" />
      {label}
    </span>
  )
}
