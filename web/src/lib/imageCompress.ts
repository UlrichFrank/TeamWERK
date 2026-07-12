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
// MIME + Größe erneut.
export async function compressImage(
  file: File,
  targetBytes = TARGET_BYTES,
  maxEdge = MAX_EDGE,
): Promise<CompressResult> {
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

  // WebP bewahrt Transparenz (PNG) und komprimiert gut; vom Server erlaubt.
  // Fallback auf JPEG, weil iOS Safari erst ab Version 16 WebP per canvas.toBlob
  // liefert und dort intermittierend `null` zurückgibt — sonst würde ein 3–8 MB
  // Kamerafoto ungekürzt weitergereicht und der Server mit 413 ablehnen.
  const stem = stripExt(file.name);
  const attempts: { mime: string; ext: string }[] = [
    { mime: "image/webp", ext: ".webp" },
    { mime: "image/jpeg", ext: ".jpg" },
  ];
  let smallest: { blob: Blob; ext: string } | null = null;
  for (const { mime, ext } of attempts) {
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
