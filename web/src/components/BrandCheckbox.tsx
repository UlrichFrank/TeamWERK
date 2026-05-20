interface Props {
  checked: boolean
  onChange: (checked: boolean) => void
  label: string
  disabled?: boolean
}

export default function BrandCheckbox({ checked, onChange, label, disabled = false }: Props) {
  return (
    <label className={`inline-flex items-center gap-1 px-2 py-1 text-xs rounded transition-colors cursor-pointer select-none ${
      checked
        ? 'bg-brand-yellow text-brand-black font-medium'
        : 'bg-white border border-gray-300 text-gray-600 hover:border-brand-yellow'
    } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}>
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
        disabled={disabled}
        className="sr-only"
      />
      {checked && (
        <svg
          className="w-3 h-3 flex-shrink-0"
          viewBox="0 0 16 16"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <polyline points="13 3 6 13 3 10"></polyline>
        </svg>
      )}
      <span>{label}</span>
    </label>
  )
}
