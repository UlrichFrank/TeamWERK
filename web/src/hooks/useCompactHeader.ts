import { useEffect, useRef, useState } from 'react'

export function useCompactHeader(threshold = 450) {
  const ref = useRef<HTMLDivElement>(null)
  const [compact, setCompact] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const observer = new ResizeObserver(entries => {
      setCompact(entries[0].contentRect.width < threshold)
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [threshold])

  return { ref, compact }
}
