// Albumbilder einer Chat-Nachricht oder Mitteilung (chat-mehrere-bilder).
// url ist der relative Abrufpfad ohne /api-Prefix (axios ergänzt ihn).
export interface ChatMediaItem {
  id: number;
  url: string;
  width?: number;
  height?: number;
}

// Höchstzahl Bilder je Nachricht bzw. Mitteilung — dieselbe Grenze prüft der
// Server (internal/chat/album.go, maxAlbumSize).
export const MAX_ALBUM_SIZE = 10;

export const ALBUM_LIMIT_TOAST = `Höchstens ${MAX_ALBUM_SIZE} Bilder je Nachricht`;

interface WithMedia {
  media?: ChatMediaItem[];
  mediaId: number | null;
  mediaUrl: string | null;
  mediaWidth?: number;
  mediaHeight?: number;
}

// albumOf liefert die Albumbilder; fehlt `media` (Antwort eines älteren
// Servers aus dem Cache), wird das Einzelbild aus den Altfeldern gebildet.
export function albumOf(obj: WithMedia): ChatMediaItem[] {
  if (obj.media) return obj.media;
  if (obj.mediaId !== null && obj.mediaUrl) {
    return [
      {
        id: obj.mediaId,
        url: obj.mediaUrl,
        width: obj.mediaWidth,
        height: obj.mediaHeight,
      },
    ];
  }
  return [];
}

// Ein noch nicht gesendetes Bild inkl. lokaler Vorschau-URL.
export interface PendingImage {
  file: File;
  previewUrl: string;
}

// addPendingImages hängt Bilddateien an die bestehende Auswahl an, bis zur
// Grenze; Nicht-Bilder werden ignoriert. `overflow` meldet, ob Dateien wegen
// der Grenze verworfen wurden (→ Toast beim Aufrufer).
export function addPendingImages(
  prev: PendingImage[],
  files: File[],
): { next: PendingImage[]; overflow: boolean } {
  const images = files.filter((f) => f.type.startsWith("image/"));
  const room = Math.max(0, MAX_ALBUM_SIZE - prev.length);
  const taken = images.slice(0, room);
  return {
    next: [
      ...prev,
      ...taken.map((file) => ({ file, previewUrl: URL.createObjectURL(file) })),
    ],
    overflow: images.length > room,
  };
}

export function revokePending(images: PendingImage[]): void {
  for (const img of images) URL.revokeObjectURL(img.previewUrl);
}

// uploadSequentially lädt die Bilder strikt nacheinander hoch (nicht parallel:
// jedes compressImage dekodiert in ein Canvas, zehn parallele Decodes von
// Handy-Fotos sind auf älteren iPhones ein Speicherrisiko). onProgress meldet
// das gerade laufende Bild (1-basiert). Scheitert ein Upload, bricht der Lauf
// ab und liefert null.
export async function uploadSequentially(
  images: PendingImage[],
  upload: (file: File) => Promise<number | null>,
  onProgress?: (k: number, n: number) => void,
): Promise<number[] | null> {
  const ids: number[] = [];
  for (let i = 0; i < images.length; i++) {
    onProgress?.(i + 1, images.length);
    const id = await upload(images[i].file);
    if (id === null) return null;
    ids.push(id);
  }
  return ids;
}
