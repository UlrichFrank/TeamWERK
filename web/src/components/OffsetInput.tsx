import { useState, useEffect } from 'react'
import { formatOffset, parseOffset } from '../lib/time'

export default function OffsetInput({ value, onChange, className }: {
  value: number
  onChange: (v: number) => void
  className?: string
}) {
  const [str, setStr] = useState(formatOffset(value))

  // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
  useEffect(() => { setStr(formatOffset(value)) }, [value])

  return (
    <input
      type="text"
      value={str}
      placeholder="0"
      onChange={e => setStr(e.target.value)}
      onBlur={() => {
        const mins = parseOffset(str)
        onChange(mins)
        setStr(formatOffset(mins))
      }}
      className={className}
    />
  )
}
