export function DaySeparator({ label }: { label: string }) {
  return (
    <div
      role="separator"
      aria-label={label}
      className="flex items-center gap-3 my-3 text-xs text-brand-text-muted"
    >
      <div className="flex-1 h-px bg-brand-border-subtle" />
      <span>{label}</span>
      <div className="flex-1 h-px bg-brand-border-subtle" />
    </div>
  )
}
