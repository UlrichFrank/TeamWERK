import { useEffect, useState } from "react";
import { api } from "../lib/api";

// AuthImage lädt ein JWT-geschütztes Bild per axios (Bearer kommt automatisch mit)
// als Blob und zeigt es über eine Object-URL an. Nötig, weil <img src> keine
// Authorization-Header sendet. Muster wie ReportImage in MatchReportFormPage.
//
// Um Layout-Shifts zu vermeiden (relevant im Chat-Verlauf, wo Bild-Loads sonst
// die Scroll-Position verschieben) gilt eine Zwei-Wege-Strategie:
//
// 1. **Bevorzugt**: naturalWidth/naturalHeight kommen vom Server (aus der
//    media-Tabelle, per Upload-Probe befüllt) → aspect-ratio ab dem ersten
//    Frame gesetzt, kein Client-Preload nötig, KEIN Shift.
// 2. **Fallback**: fehlen die Server-Dims (Bestand vor Backfill, unlesbarer
//    Header, andere AuthImage-Aufrufer), wird das geladene Blob per `Image()`
//    lokal gemessen und aspect-ratio nachträglich gesetzt. Ein einmaliger
//    Placeholder→Bild-Shift bleibt möglich, wird aber danach eingefroren.
export default function AuthImage({
  url,
  alt,
  className,
  onClick,
  naturalWidth,
  naturalHeight,
}: {
  url: string;
  alt?: string;
  className?: string;
  onClick?: () => void;
  naturalWidth?: number;
  naturalHeight?: number;
}) {
  const [src, setSrc] = useState<string | null>(null);
  const [probedDims, setProbedDims] = useState<{ w: number; h: number } | null>(
    null,
  );
  const [error, setError] = useState(false);

  const serverDims =
    naturalWidth && naturalHeight
      ? { w: naturalWidth, h: naturalHeight }
      : null;

  useEffect(() => {
    let cancelled = false;
    let objectUrl: string | null = null;
    setError(false);
    setSrc(null);
    setProbedDims(null);
    api
      .get(url, { responseType: "blob" })
      .then((res) => {
        if (cancelled) return;
        const created = URL.createObjectURL(res.data as Blob);
        objectUrl = created;
        // Server-Dims vorhanden → Blob direkt anzeigen, kein Preload nötig.
        if (naturalWidth && naturalHeight) {
          setSrc(created);
          return;
        }
        // Sonst: lokal messen, dann anzeigen.
        const probe = new Image();
        probe.onload = () => {
          if (cancelled) return;
          setProbedDims({ w: probe.naturalWidth, h: probe.naturalHeight });
          setSrc(created);
        };
        probe.onerror = () => {
          if (cancelled) return;
          setError(true);
        };
        probe.src = created;
      })
      .catch(() => {
        if (!cancelled) setError(true);
      });
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [url, naturalWidth, naturalHeight]);

  if (error) {
    return (
      <div className="text-xs text-brand-text-muted py-2">
        Bild nicht verfügbar
      </div>
    );
  }

  const dims = serverDims ?? probedDims;
  const style = dims
    ? { aspectRatio: `${dims.w} / ${dims.h}` }
    : { minHeight: "6rem" };

  if (!src) {
    return (
      <div
        className={`${className ?? ""} bg-brand-surface-card animate-pulse`}
        style={style}
        aria-busy="true"
      />
    );
  }
  return (
    <img
      src={src}
      alt={alt ?? ""}
      className={className}
      style={style}
      onClick={onClick}
    />
  );
}
