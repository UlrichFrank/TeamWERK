export type RsvpDefault = 'confirmed' | 'declined' | 'none'

const OPTIONS: [RsvpDefault, string][] = [
  ['confirmed', 'Standardmäßig zugesagt'],
  ['declined', 'Standardmäßig abgesagt'],
  ['none', 'Keine automatische Rückmeldung'],
]

interface Props {
  defaultPlayers: RsvpDefault
  defaultExtended: RsvpDefault
  requireReason: boolean
  onChangePlayers: (v: RsvpDefault) => void
  onChangeExtended: (v: RsvpDefault) => void
  onChangeRequireReason: (v: boolean) => void
  /** Suffix for radio-group `name` attributes so multiple editors on one page stay independent. */
  idPrefix?: string
}

// RsvpDefaultsEditor renders two independent role defaults (Kader-Spieler /
// Erweiterter Kader) as radio groups plus the "Begründung bei Absage
// erforderlich" checkbox. Conflict lock: a `declined` default is mechanically
// not combinable with `rsvp_require_reason` — one disables the other.
export default function RsvpDefaultsEditor({
  defaultPlayers,
  defaultExtended,
  requireReason,
  onChangePlayers,
  onChangeExtended,
  onChangeRequireReason,
  idPrefix = 'rsvp',
}: Props) {
  const renderGroup = (
    groupLabel: string,
    name: string,
    value: RsvpDefault,
    onChange: (v: RsvpDefault) => void,
  ) => (
    <fieldset>
      <legend className="text-sm font-medium text-brand-text mb-1">{groupLabel}</legend>
      <div className="space-y-1.5">
        {OPTIONS.map(([val, label]) => (
          <label key={val} className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name={`${idPrefix}-${name}`}
              value={val}
              checked={value === val}
              onChange={() => onChange(val)}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">{label}</span>
          </label>
        ))}
      </div>
    </fieldset>
  )

  return (
    <div className="space-y-3 border-t border-brand-border-subtle pt-3">
      <p className="text-sm font-medium text-brand-text-muted">RSVP-Voreinstellung</p>
      {renderGroup('Kader-Spieler', 'players', defaultPlayers, onChangePlayers)}
      {renderGroup('Erweiterter Kader', 'extended', defaultExtended, onChangeExtended)}
      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={requireReason}
          onChange={e => onChangeRequireReason(e.target.checked)}
          className="w-4 h-4 accent-brand-yellow"
        />
        <span className="text-sm text-brand-text">Begründung bei Absage erforderlich</span>
      </label>
    </div>
  )
}
