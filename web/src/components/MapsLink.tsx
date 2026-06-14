import { MapPin } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'

interface Venue {
  name: string
  street: string
  city: string
  postal_code: string
}

interface MapsLinkProps {
  venue: Venue | null | undefined
  className?: string
}

function resolveUrl(query: string, provider: 'auto' | 'google' | 'apple'): string {
  const useApple =
    provider === 'apple' ||
    (provider === 'auto' && /iPhone|iPad|iPod|Macintosh/.test(navigator.userAgent))
  const base = useApple ? 'https://maps.apple.com/' : 'https://maps.google.com/'
  return `${base}?q=${query}`
}

export default function MapsLink({ venue, className = '' }: MapsLinkProps) {
  const { mapsProvider } = useAuth()
  if (!venue) return null

  const query = encodeURIComponent(`${venue.street} ${venue.postal_code} ${venue.city}`)
  const url = resolveUrl(query, mapsProvider)

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={`Navigation zu ${venue.name}`}
      className={`inline-flex items-center gap-1 text-brand-text-muted hover:text-brand-text transition-colors ${className}`}
    >
      <MapPin className="w-4 h-4 flex-shrink-0" />
      <span className="text-sm">{venue.name}</span>
    </a>
  )
}
