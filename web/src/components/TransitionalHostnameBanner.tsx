import { AlertTriangle } from 'lucide-react'

const ALIAS_HOST = 'internal.team-stuttgart.org'
const PRIMARY_ORIGIN = 'https://teamwerk.team-stuttgart.org'

/**
 * Persistenter, nicht schließbarer Migrations-Hinweis. Erscheint ausschließlich,
 * wenn die App unter dem Übergangs-Alias `internal.team-stuttgart.org` geladen
 * wurde — auf dem Primärhost und in der lokalen Entwicklung rendert die
 * Komponente `null`. Die App bleibt darunter voll funktional; der Banner ist ein
 * In-App-Hinweis, kein harter Block.
 */
export default function TransitionalHostnameBanner() {
  if (window.location.host !== ALIAS_HOST) return null

  const target = `${PRIMARY_ORIGIN}${window.location.pathname}${window.location.search}`

  return (
    <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 bg-brand-info/10 border-b border-brand-info/30 text-brand-text text-sm px-4 py-3">
      <div className="flex items-start sm:items-center gap-2">
        <AlertTriangle className="w-5 h-5 shrink-0 text-brand-info" />
        <p>
          Wir sind umgezogen. Öffne <strong>teamwerk.team-stuttgart.org</strong>,
          installiere die PWA neu und logge dich einmal wieder ein.
        </p>
      </div>
      <a
        href={target}
        className="shrink-0 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed text-center"
      >
        Jetzt wechseln
      </a>
    </div>
  )
}
