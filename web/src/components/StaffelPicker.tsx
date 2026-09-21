import { useEffect, useRef, useState } from 'react'
import { Check, Pencil, X } from 'lucide-react'
import { api } from '../lib/api'
import { HEADER_FIELD } from '../lib/buttonStyles'

interface KatalogEntry {
  code: string
  name: string
  org: string
}

interface Props {
  value: string
  /** Wird mit dem neuen Code aufgerufen; leerer String entfernt die Zuordnung. */
  onSave: (code: string) => Promise<void>
}

/**
 * Staffel-Zuordnung eines Kaders.
 *
 * Der Katalog kommt aus der Handball4All-Schnittstelle und wird deshalb **erst
 * beim Öffnen** geladen, nicht beim Rendern der Kaderliste: ein Abruf nach
 * außen je Kaderkarte wäre bei neun Mannschaften neun überflüssige Aufrufe.
 *
 * Ist der Katalog nicht erreichbar, bleibt die Freitext-Eingabe nutzbar — die
 * Validierung läuft ohnehin serverseitig und für beide Wege gleich.
 */
export default function StaffelPicker({ value, onSave }: Props) {
  const [open, setOpen] = useState(false)
  const [katalog, setKatalog] = useState<KatalogEntry[] | null>(null)
  const [katalogFehler, setKatalogFehler] = useState(false)
  const [entwurf, setEntwurf] = useState(value)
  const [fehler, setFehler] = useState('')
  const [speichert, setSpeichert] = useState(false)
  const geladen = useRef(false)

  useEffect(() => { setEntwurf(value) }, [value])

  useEffect(() => {
    if (!open || geladen.current) return
    geladen.current = true
    api
      .get<KatalogEntry[]>('/bwhv/staffel-katalog')
      .then((r) => setKatalog(r.data))
      .catch(() => setKatalogFehler(true))
  }, [open])

  const speichern = async (code: string) => {
    setSpeichert(true)
    setFehler('')
    try {
      await onSave(code)
      setOpen(false)
    } catch (e) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
      setFehler(msg ?? 'Speichern fehlgeschlagen')
    } finally {
      setSpeichert(false)
    }
  }

  if (!open) {
    return (
      <span className="flex items-center gap-2">
        <span className="text-xs text-brand-text-muted font-medium">Staffel:</span>
        {value ? (
          <span className="text-xs bg-brand-border-subtle text-brand-text px-2 py-0.5 rounded-full font-medium">
            {value}
          </span>
        ) : (
          <span className="text-xs text-brand-text-subtle">nicht zugeordnet</span>
        )}
        <button
          onClick={() => setOpen(true)}
          className="inline-flex items-center gap-1 text-xs text-brand-text-muted hover:text-brand-text transition-colors"
          aria-label="Staffel zuordnen"
        >
          <Pencil className="w-3 h-3" />
          {value ? 'ändern' : 'zuordnen'}
        </button>
      </span>
    )
  }

  return (
    <div className="w-full">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs text-brand-text-muted font-medium">Staffel:</span>

        {katalog && katalog.length > 0 && (
          <select
            className={HEADER_FIELD}
            value={katalog.some((k) => k.code === entwurf) ? entwurf : ''}
            onChange={(e) => setEntwurf(e.target.value)}
            aria-label="Staffel aus Katalog wählen"
          >
            <option value="">— aus Katalog wählen —</option>
            {katalog.map((k) => (
              <option key={`${k.org}-${k.code}`} value={k.code}>
                {k.code} · {k.name}
              </option>
            ))}
          </select>
        )}

        {/* Freitext ist immer verfügbar, auch wenn der Katalog nicht lädt. */}
        <input
          className={HEADER_FIELD}
          value={entwurf}
          onChange={(e) => setEntwurf(e.target.value)}
          placeholder="oder Code eingeben, z.B. mB-RL-BW"
          aria-label="Staffelcode eingeben"
        />

        <button
          onClick={() => speichern(entwurf)}
          disabled={speichert}
          className="inline-flex items-center gap-1 px-2 h-8 sm:h-[30px] rounded-md bg-brand-yellow text-brand-black text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
        >
          <Check className="w-3 h-3" /> Speichern
        </button>
        {value && (
          <button
            onClick={() => speichern('')}
            disabled={speichert}
            className="inline-flex items-center gap-1 px-2 h-8 sm:h-[30px] rounded-md border border-brand-border text-xs text-brand-danger hover:bg-brand-danger-light transition-colors disabled:opacity-40"
          >
            Entfernen
          </button>
        )}
        <button
          onClick={() => { setOpen(false); setEntwurf(value); setFehler('') }}
          className="inline-flex items-center justify-center w-8 h-8 sm:h-[30px] rounded-md border border-brand-border text-brand-text-muted hover:text-brand-text transition-colors"
          aria-label="Abbrechen"
        >
          <X className="w-3 h-3" />
        </button>
      </div>

      {katalogFehler && (
        <p className="text-xs text-brand-text-muted mt-1">
          Der Staffel-Katalog ist gerade nicht erreichbar — die Eingabe von Hand funktioniert
          trotzdem.
        </p>
      )}
      {fehler && <p className="text-xs text-brand-danger mt-1">{fehler}</p>}
    </div>
  )
}
