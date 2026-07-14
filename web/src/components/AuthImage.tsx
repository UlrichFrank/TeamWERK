import { useEffect, useState } from "react";
import { api } from "../lib/api";

// AuthImage lädt ein JWT-geschütztes Bild per axios (Bearer kommt automatisch mit)
// als Blob und zeigt es über eine Object-URL an. Nötig, weil <img src> keine
// Authorization-Header sendet. Muster wie ReportImage in MatchReportFormPage.
//
// Um Layout-Shifts zu vermeiden (relevant im Chat-Verlauf, wo Bild-Loads sonst
// die Scroll-Position verschieben), wird das Bild vor dem Rendern per
// `Image()` in den natürlichen Dimensionen probiert; die Aspect-Ratio setzen
// wir dann sowohl auf den Placeholder als auch auf das finale `<img>`. Nach
// dem Bild-Load ändert sich die Zeilenhöhe damit nicht mehr.
export default function AuthImage({
  url,
  alt,
  className,
  onClick,
}: {
  url: string;
  alt?: string;
  className?: string;
  onClick?: () => void;
}) {
  const [src, setSrc] = useState<string | null>(null);
  const [dims, setDims] = useState<{ w: number; h: number } | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    let cancelled = false;
    let objectUrl: string | null = null;
    setError(false);
    setSrc(null);
    setDims(null);
    api
      .get(url, { responseType: "blob" })
      .then((res) => {
        if (cancelled) return;
        const created = URL.createObjectURL(res.data as Blob);
        objectUrl = created;
        const probe = new Image();
        probe.onload = () => {
          if (cancelled) return;
          setDims({ w: probe.naturalWidth, h: probe.naturalHeight });
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
  }, [url]);

  if (error) {
    return (
      <div className="text-xs text-brand-text-muted py-2">
        Bild nicht verfügbar
      </div>
    );
  }

  const style = dims
    ? { aspectRatio: `${dims.w} / ${dims.h}` }
    : { minHeight: "6rem" };

  if (!src || !dims) {
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
