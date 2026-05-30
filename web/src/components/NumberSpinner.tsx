import { ChevronUp, ChevronDown } from 'lucide-react'

interface NumberSpinnerProps {
  value: number
  min?: number
  max?: number
  step?: number
  onChange: (v: number) => void
  className?: string
}

export default function NumberSpinner({ value, min, max, step = 1, onChange, className = '' }: NumberSpinnerProps) {
  const atMin = min !== undefined && value <= min
  const atMax = max !== undefined && value >= max

  const btnBase = 'flex items-center justify-center w-full transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
  const btnColor = 'bg-brand-yellow text-brand-black hover:bg-brand-black hover:text-brand-yellow'

  return (
    <div className={`inline-flex border border-brand-border rounded-md overflow-hidden focus-within:ring-2 focus-within:ring-brand-yellow focus-within:border-brand-yellow ${className}`}>
      <input
        type="number"
        value={value}
        min={min}
        max={max}
        onChange={e => onChange(parseInt(e.target.value) || 0)}
        style={{ MozAppearance: 'textfield', WebkitAppearance: 'none' } as React.CSSProperties}
        className="w-20 pl-3 py-2 text-sm text-brand-text focus:outline-none [&::-webkit-inner-spin-button]:hidden [&::-webkit-outer-spin-button]:hidden"
      />
      <div className="flex flex-col w-6">
        <button
          type="button"
          disabled={atMax}
          onClick={() => onChange(Math.min(max ?? Infinity, value + step))}
          className={`${btnBase} ${btnColor} flex-1`}
          aria-label="Erhöhen"
        >
          <ChevronUp className="w-3 h-3" />
        </button>
        <button
          type="button"
          disabled={atMin}
          onClick={() => onChange(Math.max(min ?? 0, value - step))}
          className={`${btnBase} ${btnColor} flex-1`}
          aria-label="Verringern"
        >
          <ChevronDown className="w-3 h-3" />
        </button>
      </div>
    </div>
  )
}
