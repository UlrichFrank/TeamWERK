export type RsvpDefault = 'confirmed' | 'declined' | 'none'

const CONFLICT_TOOLTIP = 'Nicht mit „Standardmäßig abgesagt“ kombinierbar'

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
  const anyDeclined = defaultPlayers === 'declined' || defaultExtended === 'declined'

  const renderGroup = (
    groupLabel: string,
    name: string,
    value: RsvpDefault,
    onChange: (v: RsvpDefault) => void,
  ) => (
    <fieldset>
      <legend className="text-sm font-medium text-brand-text mb-1">{groupLabel}</legend>
      <div className="space-y-1.5">
        {OPTIONS.map(([val, label]) => {
          const declinedLocked = val === 'declined' && requireReason
          return (
            <label
              key={val}
              className={`flex items-center gap-2 ${declinedLocked ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}`}
              title={declinedLocked ? CONFLICT_TOOLTIP : undefined}
            >
              <input
                type="radio"
                name={`${idPrefix}-${name}`}
                value={val}
                checked={value === val}
                disabled={declinedLocked}
                onChange={() => onChange(val)}
                className="w-4 h-4 accent-brand-yellow"
              />
              <span className="text-sm text-brand-text">{label}</span>
            </label>
          )
        })}
      </div>
    </fieldset>
  )

  return (
    <div className="space-y-3 border-t border-brand-border-subtle pt-3">
      <p className="text-sm font-medium text-brand-text-muted">RSVP-Voreinstellung</p>
      {renderGroup('Kader-Spieler', 'players', defaultPlayers, onChangePlayers)}
      {renderGroup('Erweiterter Kader', 'extended', defaultExtended, onChangeExtended)}
      <label
        className={`flex items-center gap-2 ${anyDeclined ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}`}
        title={anyDeclined ? CONFLICT_TOOLTIP : undefined}
      >
        <input
          type="checkbox"
          checked={requireReason}
          disabled={anyDeclined}
          onChange={e => onChangeRequireReason(e.target.checked)}
          className="w-4 h-4 accent-brand-yellow"
        />
        <span className="text-sm text-brand-text">Begründung bei Absage erforderlich</span>
      </label>
    </div>
  )
}
