// Kennzeichen für den erweiterten Kader. Am Termin heißt es „Erw. Kader“
// (Zugehörigkeit), an Diensten „Aushilfe“ (freiwillig, keine Dienstpflicht) —
// Change dienste-erweiterter-kader. Ein Baustein, damit beide gleich aussehen.
export default function AushilfeBadge({ label = 'Aushilfe' }: { label?: string }) {
  return (
    <span className="inline-flex items-center rounded-full bg-brand-blue/10 px-2 py-0.5 text-xs font-semibold text-brand-blue border border-brand-blue/30 whitespace-nowrap flex-shrink-0">
      {label}
    </span>
  )
}
