import { MapPin } from 'lucide-react'

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

export default function MapsLink({ venue, className = '' }: MapsLinkProps) {
  if (!venue) return null

  const query = encodeURIComponent(`${venue.street} ${venue.postal_code} ${venue.city}`)
  const url = `https://maps.google.com/?q=${query}`

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
