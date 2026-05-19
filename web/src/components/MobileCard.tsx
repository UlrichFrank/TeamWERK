import { ReactNode } from 'react'

interface MobileCardProps {
  title: string
  subtitle?: string
  badge?: { label: string; variant?: 'yellow' | 'green' | 'red' | 'blue' }
  children?: ReactNode
}

export default function MobileCard({ title, subtitle, badge, children }: MobileCardProps) {
  const badgeStyles = {
    yellow: 'bg-brand-yellow text-brand-black',
    green: 'bg-brand-green text-white',
    red: 'bg-red-500 text-white',
    blue: 'bg-brand-blue text-white',
  }

  return (
    <div className="bg-white border border-brand-black/10 rounded p-4 mb-3">
      <div className="flex items-start justify-between mb-1">
        <div className="flex-1">
          <div className="font-medium text-brand-black">{title}</div>
          {subtitle && <div className="text-sm text-brand-black/60">{subtitle}</div>}
        </div>
        {badge && (
          <span className={`ml-2 px-2 py-1 text-xs font-medium rounded whitespace-nowrap ${badgeStyles[badge.variant || 'yellow']}`}>
            {badge.label}
          </span>
        )}
      </div>
      {children && <div className="mt-3 text-sm">{children}</div>}
    </div>
  )
}
