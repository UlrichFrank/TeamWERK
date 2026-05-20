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
          : 'bg-white border border-gray-300 text-gray-600 hover:border-brand-yellow'
      } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
        disabled={disabled}
        className="sr-only"
      />
      <svg
        className="w-3 h-3 flex-shrink-0"
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        style={{ opacity: checked ? 1 : 0.3 }}
      >
        <polyline points="13 3 6 13 3 10"></polyline>
      </svg>
      {label && <span>{label}</span>}
    </label>
  )
}
