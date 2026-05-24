import { useState, useEffect } from 'react'
import { formatOffset, parseOffset } from '../lib/time'

export default function OffsetInput({ value, onChange, className }: {
  value: number
  onChange: (v: number) => void
  className?: string
}) {
  const [str, setStr] = useState(formatOffset(value))

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
