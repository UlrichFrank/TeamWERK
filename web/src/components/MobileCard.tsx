import { ReactNode } from 'react'
import ActionMenu from './ActionMenu'

interface Action {
  label: string
  onClick: () => void
  variant?: 'default' | 'danger'
}

interface MobileCardProps {
  title: string
  subtitle?: string
  badge?: { label: string; variant?: 'yellow' | 'green' | 'red' | 'blue' }
  actions?: Action[]
  onClick?: () => void
  children?: ReactNode
}

export default function MobileCard({ title, subtitle, badge, actions, onClick, children }: MobileCardProps) {
  const badgeStyles = {
    yellow: 'bg-brand-yellow text-brand-black',
    green: 'bg-brand-green text-white',
    red: 'bg-brand-danger text-white',
    blue: 'bg-brand-blue text-white',
  }

  return (
    <div className={`bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4 mb-3${onClick ? ' cursor-pointer' : ''}`} onClick={onClick}>
      <div className="flex items-start justify-between gap-2 mb-1">
        <div className="flex-1 min-w-0">
          <div className="font-medium text-brand-text">{title}</div>
          {subtitle && <div className="text-sm text-brand-text-muted">{subtitle}</div>}
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          {badge && (
            <span className={`px-2 py-1 text-xs font-medium rounded whitespace-nowrap ${badgeStyles[badge.variant || 'yellow']}`}>
              {badge.label}
            </span>
          )}
          {actions && <ActionMenu actions={actions} />}
        </div>
      </div>
      {children && <div className="mt-3 text-sm text-brand-text">{children}</div>}
    </div>
  )
}
