import { useEffect, useState } from "react";
import { api } from "../lib/api";

// AuthImage lädt ein JWT-geschütztes Bild per axios (Bearer kommt automatisch mit)
// als Blob und zeigt es über eine Object-URL an. Nötig, weil <img src> keine
// Authorization-Header sendet. Muster wie ReportImage in MatchReportFormPage.
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
  const [error, setError] = useState(false);

  useEffect(() => {
    let revoked = false;
    let objectUrl: string | null = null;
    setError(false);
    setSrc(null);
    api
      .get(url, { responseType: "blob" })
      .then((res) => {
        if (revoked) return;
        objectUrl = URL.createObjectURL(res.data as Blob);
        setSrc(objectUrl);
      })
      .catch(() => {
        if (!revoked) setError(true);
      });
    return () => {
      revoked = true;
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
  if (!src) {
    return (
      <div
        className={`${className ?? ""} bg-brand-surface-card animate-pulse`}
        style={{ minHeight: "6rem" }}
        aria-busy="true"
      />
    );
  }
  return <img src={src} alt={alt ?? ""} className={className} onClick={onClick} />;
}
