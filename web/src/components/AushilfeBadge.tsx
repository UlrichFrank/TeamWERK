// Kennzeichen für den erweiterten Kader — am Termin wie an Diensten „Erw. Kader“
// (Change dienste-erweiterter-kader). Ein Baustein, damit alle Stellen gleich aussehen.
export default function AushilfeBadge({ label = 'Erw. Kader' }: { label?: string }) {
  return (
    <span className="inline-flex items-center rounded-full bg-brand-blue/10 px-2 py-0.5 text-xs font-semibold text-brand-blue border border-brand-blue/30 whitespace-nowrap flex-shrink-0">
      {label}
    </span>
  )
}
