import AuthImage from "./AuthImage";
import type { ChatMediaItem } from "../lib/chatMedia";

// Sichtbare Kacheln eines Albums; ab dem fünften Bild trägt die vierte Kachel
// den Hinweis „+N" auf die übrigen.
const VISIBLE_TILES = 4;

// ChatImageGrid zeigt die Bilder einer Nachricht bzw. Mitteilung.
//
// Ein Bild erscheint wie bisher in natürlicher Größe (AuthImage mit
// Server-Dimensionen). Ab zwei Bildern entsteht ein Raster mit FESTER
// Geometrie: feste Breite, Kacheln mit fixem Seitenverhältnis, Bilder per
// object-cover beschnitten. Die Höhe hängt damit nur an der Bildanzahl, nie am
// Ladezustand — Pflicht in der shrink-to-fit-Sprechblase (Chat-Bild-Gotcha in
// docs/agent/06-gotchas.md), und unabhängig davon, ob der Server Dimensionen
// kennt. Die Lightbox zeigt immer das ganze Bild.
export default function ChatImageGrid({
  media,
  onOpen,
  className = "",
  alt = "Bild",
}: {
  media: ChatMediaItem[];
  onOpen: (index: number) => void;
  className?: string;
  alt?: string;
}) {
  if (media.length === 0) return null;

  if (media.length === 1) {
    const m = media[0];
    return (
      <AuthImage
        url={m.url}
        alt={alt}
        className={`${className} max-w-full rounded-lg cursor-pointer`}
        onClick={() => onOpen(0)}
        naturalWidth={m.width}
        naturalHeight={m.height}
      />
    );
  }

  const visible = media.slice(0, VISIBLE_TILES);
  const hidden = media.length - visible.length;

  return (
    <div
      className={`${className} grid grid-cols-2 gap-1 w-64 max-w-full`}
      data-testid="chat-image-grid"
    >
      {visible.map((m, i) => {
        // Bei drei Bildern steht das erste breit über den beiden anderen.
        const wide = media.length === 3 && i === 0;
        const showMore = hidden > 0 && i === VISIBLE_TILES - 1;
        return (
          <button
            key={m.id}
            type="button"
            onClick={() => onOpen(i)}
            aria-label={
              showMore
                ? `${alt} ${i + 1} von ${media.length}, ${hidden} weitere`
                : `${alt} ${i + 1} von ${media.length}`
            }
            data-testid="chat-image-tile"
            className={`relative overflow-hidden rounded-md bg-brand-surface-card ${
              wide ? "col-span-2 aspect-2/1" : "aspect-square"
            }`}
          >
            <AuthImage
              url={m.url}
              alt={alt}
              fill
              className="absolute inset-0 w-full h-full object-cover"
            />
            {showMore && (
              <span className="absolute inset-0 flex items-center justify-center bg-brand-black/50 text-white text-xl font-semibold">
                +{hidden}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
