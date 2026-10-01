import { useEffect, useRef } from "react";
import { ChevronLeft, ChevronRight, X } from "lucide-react";
import { useDialogA11y } from "../lib/useDialogA11y";
import AuthImage from "./AuthImage";

// Vollbildansicht eines Chat-Albums: blättert per Pfeil-Buttons und ←/→,
// schließt per Hintergrund-Klick, X und ESC. Bei genau einem Bild entfallen
// Pfeile und Zähler. Das Bild lädt über AuthImage ein zweites Mal (eigene
// Object-URL) — dieselbe Begründung wie bei ImageLightbox.
export default function AlbumLightbox({
  urls,
  index,
  onIndexChange,
  onClose,
}: {
  urls: string[];
  index: number;
  onIndexChange: (index: number) => void;
  onClose: () => void;
}) {
  const dialogRef = useRef<HTMLDivElement>(null);
  useDialogA11y(dialogRef, true);
  const n = urls.length;
  const hasPrev = index > 0;
  const hasNext = index < n - 1;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
      else if (e.key === "ArrowLeft" && index > 0) onIndexChange(index - 1);
      else if (e.key === "ArrowRight" && index < n - 1) onIndexChange(index + 1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [index, n, onClose, onIndexChange]);

  const navBtn =
    "absolute top-1/2 -translate-y-1/2 text-white hover:text-brand-yellow transition-colors disabled:opacity-30 disabled:cursor-not-allowed p-2";

  return (
    <div
      ref={dialogRef}
      role="dialog"
      aria-modal="true"
      aria-label="Bild"
      className="fixed inset-0 z-50 bg-brand-black/80 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <button
        type="button"
        onClick={onClose}
        aria-label="Schließen"
        className="absolute top-4 right-4 text-white hover:text-brand-yellow transition-colors"
      >
        <X className="w-7 h-7" />
      </button>
      {n > 1 && (
        <>
          <span className="absolute top-5 left-1/2 -translate-x-1/2 text-white text-sm">
            {index + 1} / {n}
          </span>
          <button
            type="button"
            aria-label="Vorheriges Bild"
            disabled={!hasPrev}
            onClick={(e) => {
              e.stopPropagation();
              if (hasPrev) onIndexChange(index - 1);
            }}
            className={`${navBtn} left-2`}
          >
            <ChevronLeft className="w-8 h-8" />
          </button>
          <button
            type="button"
            aria-label="Nächstes Bild"
            disabled={!hasNext}
            onClick={(e) => {
              e.stopPropagation();
              if (hasNext) onIndexChange(index + 1);
            }}
            className={`${navBtn} right-2`}
          >
            <ChevronRight className="w-8 h-8" />
          </button>
        </>
      )}
      <AuthImage
        key={urls[index]}
        url={urls[index]}
        alt={n > 1 ? `Bild ${index + 1} von ${n}` : "Bild"}
        className="max-h-[90vh] max-w-full object-contain rounded-lg"
      />
    </div>
  );
}
