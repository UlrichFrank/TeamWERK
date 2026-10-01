import { X } from "lucide-react";
import type { PendingImage } from "../lib/chatMedia";

// Vorschauleiste der noch nicht gesendeten Bilder (Chat-Eingabe und
// Mitteilungs-Dialog): Miniaturen in Auswahlreihenfolge, jede einzeln
// entfernbar. Scrollt bei vielen Bildern horizontal statt umzubrechen.
export default function PendingImageStrip({
  images,
  onRemove,
  disabled = false,
  className = "",
}: {
  images: PendingImage[];
  onRemove: (index: number) => void;
  disabled?: boolean;
  className?: string;
}) {
  if (images.length === 0) return null;
  return (
    <ul
      className={`flex gap-2 overflow-x-auto ${className}`}
      aria-label="Ausgewählte Bilder"
    >
      {images.map((img, i) => (
        <li key={img.previewUrl} className="relative shrink-0">
          <img
            src={img.previewUrl}
            alt={`Vorschau ${i + 1}`}
            className="h-16 w-16 object-cover rounded-md border border-brand-border-subtle"
          />
          <button
            type="button"
            onClick={() => onRemove(i)}
            disabled={disabled}
            aria-label={`Bild ${i + 1} entfernen`}
            className="absolute -top-1.5 -right-1.5 bg-white rounded-full border border-brand-border p-0.5 text-brand-text-muted hover:text-brand-text disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <X className="w-3 h-3" />
          </button>
        </li>
      ))}
    </ul>
  );
}
