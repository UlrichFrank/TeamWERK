import { useState, useEffect, useId } from 'react'
import { hoursToDisplay, parseHoursInput } from '../lib/duration'

// Dieselben Presets wie die "Dauer"-Datalist auf /diensttypen (AdminDutyTypesPage).
const HOURS_PRESETS = ['30min', '45min', '1h', '1h 15min', '1h 30min', '1h 45min', '2h', '2h 30min', '3h']

/**
 * Freitext-Eingabe für `hours_value` (REAL, Stunden) mit Presets-Datalist —
 * dasselbe Zustands-Muster wie OffsetInput/DurationInput: eine lokale
 * String-Repräsentation, die erst beim Verlassen des Felds geparst und an
 * `onChange` gemeldet wird, damit "1h 3" beim Tippen nicht zwischenzeitlich
 * kaputt normalisiert wird. `useId()` macht die Datalist-ID pro Instanz
 * eindeutig, weil mehrere Zeilen (Vorlagen-Editor) gleichzeitig auf der
 * Seite stehen können.
 */
export default function HoursInput({ value, onChange, className, placeholder = 'z.B. 1h 30min' }: {
  value: number
  onChange: (v: number) => void
  className?: string
  placeholder?: string
}) {
  const listId = useId()
  const [str, setStr] = useState(hoursToDisplay(value))

  // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
  useEffect(() => { setStr(hoursToDisplay(value)) }, [value])

  return (
    <>
      <input
        type="text"
        list={listId}
        value={str}
        placeholder={placeholder}
        onChange={e => setStr(e.target.value)}
        onBlur={() => {
          const hours = parseHoursInput(str)
          onChange(hours)
          setStr(hoursToDisplay(hours))
        }}
        className={className}
      />
      <datalist id={listId}>
        {HOURS_PRESETS.map(v => <option key={v} value={v} />)}
      </datalist>
    </>
  )
}
