import { useEffect, useState } from 'react'

export function useCompactHeader(threshold = 950) {
  const [compact, setCompact] = useState(() => window.innerWidth < threshold)

  useEffect(() => {
    const handler = () => setCompact(window.innerWidth < threshold)
    window.addEventListener('resize', handler, { passive: true })
    return () => window.removeEventListener('resize', handler)
  }, [threshold])

  return compact
}
