// Clientseitige Bild-Verkleinerung vor dem Upload.
//
// Ziel: jedes hochgeladene Bild ist ≤ 1 MB (Server erzwingt dasselbe Limit als
// Backstop, > 1 MB → HTTP 413). Strategie: längste Kante auf MAX_EDGE deckeln,
// dann per Canvas als WebP mit iterativ sinkender Qualität exportieren, bis das
// Ergebnis unter dem Ziel liegt. Bereits kleine Dateien (inkl. animierter GIFs)
// bleiben unverändert, damit Animationen nicht zerstört werden.

const TARGET_BYTES = 1 << 20; // 1 MB
const MAX_EDGE = 1920; // px, längste Kante
const QUALITY_STEPS = [0.85, 0.75, 0.65, 0.55, 0.45];

// Default-Ausgabeformate: WebP zuerst (bessere Kompression), JPEG als Fallback
// wenn der Browser WebP nicht liefert (iOS Safari < 16). Aufrufer, deren
// Server-MIME-Filter WebP nicht akzeptiert (z. B. match-reports mit
// image/jpeg+image/png-only), übergeben eine engere Liste via opts.formats.
const DEFAULT_FORMATS: OutputFormat[] = [
  { mime: "image/webp", ext: ".webp" },
  { mime: "image/jpeg", ext: ".jpg" },
];

export interface OutputFormat {
  mime: string;
  ext: string;
}

export interface CompressOptions {
  targetBytes?: number;
  maxEdge?: number;
  formats?: OutputFormat[];
}

export interface CompressResult {
  blob: Blob;
  fileName: string;
}

function fitWithin(
  w: number,
  h: number,
  maxEdge: number,
): { width: number; height: number } {
  if (w <= maxEdge && h <= maxEdge) return { width: w, height: h };
  const scale = maxEdge / Math.max(w, h);
  return { width: Math.round(w * scale), height: Math.round(h * scale) };
}

async function loadBitmap(file: File): Promise<ImageBitmap | HTMLImageElement> {
  if (typeof createImageBitmap === "function") {
    try {
      return await createImageBitmap(file);
    } catch {
      /* Fallback auf <img> unten */
    }
  }
  return await new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error("image load failed"));
    };
    img.src = url;
  });
}

function toBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality));
}

function stripExt(name: string): string {
  const i = name.lastIndexOf(".");
  return i > 0 ? name.slice(0, i) : name;
}

// compressImage verkleinert file auf ≤ targetBytes. Kann bei Nicht-Bild- oder
// nicht darstellbaren Dateien die Originaldatei zurückgeben — der Server prüft
// MIME + Größe erneut. `opts.formats` steuert die Ausgabe-MIME-Reihenfolge
// (Default WebP → JPEG); Aufrufer mit engerer Server-Whitelist übergeben z. B.
// nur JPEG.
export async function compressImage(
  file: File,
  opts: CompressOptions = {},
): Promise<CompressResult> {
  const targetBytes = opts.targetBytes ?? TARGET_BYTES;
  const maxEdge = opts.maxEdge ?? MAX_EDGE;
  const formats = opts.formats ?? DEFAULT_FORMATS;

  // Schon klein genug → unverändert (bewahrt u.a. animierte GIFs).
  if (file.size <= targetBytes) {
    return { blob: file, fileName: file.name };
  }

  let bitmap: ImageBitmap | HTMLImageElement;
  try {
    bitmap = await loadBitmap(file);
  } catch {
    return { blob: file, fileName: file.name };
  }

  const srcW = "width" in bitmap ? bitmap.width : (bitmap as HTMLImageElement).naturalWidth;
  const srcH = "height" in bitmap ? bitmap.height : (bitmap as HTMLImageElement).naturalHeight;
  const { width, height } = fitWithin(srcW, srcH, maxEdge);

  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) return { blob: file, fileName: file.name };
  ctx.drawImage(bitmap as CanvasImageSource, 0, 0, width, height);
  if ("close" in bitmap && typeof bitmap.close === "function") bitmap.close();

  const stem = stripExt(file.name);
  let smallest: { blob: Blob; ext: string } | null = null;
  for (const { mime, ext } of formats) {
    for (const q of QUALITY_STEPS) {
      const blob = await toBlob(canvas, mime, q);
      if (!blob) break; // MIME wird vom Browser nicht encodiert → nächstes Format
      if (!smallest || blob.size < smallest.blob.size) smallest = { blob, ext };
      if (blob.size <= targetBytes) return { blob, fileName: stem + ext };
    }
  }
  // Keine Qualitätsstufe reichte → kleinste Variante (Server-Backstop greift ggf.).
  if (smallest) return { blob: smallest.blob, fileName: stem + smallest.ext };
  return { blob: file, fileName: file.name };
}
