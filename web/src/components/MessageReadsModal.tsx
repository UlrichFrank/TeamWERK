import { useEffect, useId, useRef, useState } from "react";
import { X } from "lucide-react";
import { api } from "../lib/api";
import { useEscapeKey } from "../lib/useEscapeKey";
import { useDialogA11y } from "../lib/useDialogA11y";
import { errorMessage } from "../lib/errors";

interface Reader {
  userId: number;
  name: string;
  readAt: string;
}

// Chat-Nachricht oder Mitteilung — beide Leserlisten haben dieselbe Form und
// sind absender-only; nur Route und Leertext unterscheiden sich.
export type ReadsTarget =
  | { kind: "message"; id: number }
  | { kind: "broadcast"; id: number };

interface Props {
  target: ReadsTarget;
  onClose: () => void;
}

const readsUrl = (t: ReadsTarget) =>
  t.kind === "message" ? `/chat/messages/${t.id}/reads` : `/chat/broadcasts/${t.id}/reads`;

// MessageReadsModal zeigt dem Absender, wer seine Nachricht bzw. Mitteilung
// wann gelesen hat. Lädt die Leserliste on-demand (nur der Absender ist berechtigt).
export default function MessageReadsModal({ target, onClose }: Props) {
  const url = readsUrl(target);
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  useEscapeKey(onClose);
  useDialogA11y(dialogRef, true);
  const [readers, setReaders] = useState<Reader[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const r = await api.get(url);
        if (alive) setReaders(r.data ?? []);
      } catch (e) {
        if (alive) setError(errorMessage(e, "Fehler beim Laden"));
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [url]);

  return (
    <div
      className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow transform-gpu p-6 w-full max-w-md max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4 shrink-0">
          <h2 id={titleId} className="text-lg font-bold text-brand-text">Gelesen von</h2>
          <button
            onClick={onClose}
            aria-label="Schließen"
            className="p-1 rounded hover:bg-brand-border-subtle transition-colors"
          >
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        {error && <p className="text-sm text-brand-danger">{error}</p>}

        {!error && !loading && readers.length === 0 && (
          <p className="text-sm text-brand-text-muted">
            Noch niemand hat diese {target.kind === "broadcast" ? "Mitteilung" : "Nachricht"} gelesen.
          </p>
        )}

        {readers.length > 0 && (
          <ul className="border border-brand-border-subtle rounded-md divide-y divide-brand-border-subtle overflow-y-auto">
            {readers.map((r) => (
              <li
                key={r.userId}
                className="flex items-center gap-2 px-3 py-2.5 text-sm"
              >
                <span className="flex-1 text-brand-text truncate">{r.name}</span>
                <span className="text-xs text-brand-text-subtle">
                  {new Date(r.readAt).toLocaleTimeString("de-DE", {
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
