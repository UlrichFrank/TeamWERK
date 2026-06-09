interface ToggleProps {
  enabled: boolean
  onToggle: () => void
  label: string
}

export default function Toggle({ enabled, onToggle, label }: ToggleProps) {
  return (
    <button
      onClick={onToggle}
      aria-label={label}
      className={`relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors ${
        enabled ? 'bg-brand-yellow' : 'bg-brand-border'
      }`}
    >
      <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
        enabled ? 'translate-x-6' : 'translate-x-1'
      }`} />
    </button>
  )
}
