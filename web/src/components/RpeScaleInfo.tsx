import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'

// Erklärung der RPE-Skala (Rate of Perceived Exertion, 1–10). Standardmäßig
// eingeklappt: wer die Skala kennt, soll das Formular nicht erst zuscrollen
// müssen; wer sie zum ersten Mal sieht, findet die Erklärung direkt daneben.
const STEPS: { range: string; label: string; description: string }[] = [
  { range: '1–2', label: 'sehr leicht', description: 'lockeres Auslaufen, kaum Anstrengung' },
  { range: '3–4', label: 'leicht', description: 'du könntest dich problemlos unterhalten' },
  { range: '5–6', label: 'mittel', description: 'spürbar fordernd, Sprechen wird kürzer' },
  { range: '7–8', label: 'hart', description: 'schwere Beine, nur noch Stichworte möglich' },
  { range: '9–10', label: 'maximal', description: 'Vollgas, kaum durchzuhalten' },
]

export default function RpeScaleInfo() {
  const [open, setOpen] = useState(false)

  return (
    <div className="mt-2">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        aria-expanded={open}
        className="inline-flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
      >
        {open ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
        Was bedeutet die Skala?
      </button>
      {open && (
        <div className="mt-2 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
          <dl className="space-y-1">
            {STEPS.map(step => (
              <div key={step.range} className="flex gap-2">
                <dt className="w-12 shrink-0 font-medium tabular-nums">{step.range}</dt>
                <dd>
                  <span className="font-medium">{step.label}</span>
                  <span className="text-brand-text-muted"> — {step.description}</span>
                </dd>
              </div>
            ))}
          </dl>
          <p className="mt-3 text-brand-text-muted">
            Schätz einfach, wie anstrengend es sich angefühlt hat — es gibt kein richtig oder falsch.
          </p>
        </div>
      )}
    </div>
  )
}
