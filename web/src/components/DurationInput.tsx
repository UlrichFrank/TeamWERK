import { useState, useEffect } from 'react'
import { formatDuration, parseDuration } from '../lib/time'

export default function DurationInput({ value, onChange, className, placeholder = 'z.B. 1h 30min' }: {
  value: number
  onChange: (v: number) => void
  className?: string
  placeholder?: string
}) {
  const [str, setStr] = useState(formatDuration(value))

  useEffect(() => { setStr(formatDuration(value)) }, [value])

  return (
    <input
      type="text"
      value={str}
      placeholder={placeholder}
      onChange={e => setStr(e.target.value)}
      onBlur={() => {
        const mins = parseDuration(str)
        onChange(mins)
        setStr(formatDuration(mins))
      }}
      className={className}
    />
  )
}
