import { SlidersHorizontal } from 'lucide-react'

interface Props {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
  title?: string
  disabled?: boolean
}

export default function BrandCheckbox({ checked, onChange, label, title, disabled = false }: Props) {
  return (
    <label
      title={title}
      className={`inline-flex items-center justify-center gap-1 px-2 py-1 text-xs rounded transition-colors cursor-pointer select-none ${
        checked
          ? 'bg-brand-yellow text-brand-black font-medium'
          : 'bg-white border border-brand-border text-brand-text-muted hover:border-brand-yellow'
      } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
        disabled={disabled}
        className="sr-only"
      />
      <SlidersHorizontal className="w-3 h-3 flex-shrink-0" style={{ opacity: checked ? 1 : 0.5 }} />
      {label && <span>{label}</span>}
    </label>
  )
}
