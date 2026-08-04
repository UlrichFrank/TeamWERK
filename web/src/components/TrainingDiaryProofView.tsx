import { useState } from 'react'
import { Download, FileText } from 'lucide-react'
import AuthImage from './AuthImage'
import ImageLightbox from './ImageLightbox'
import { isImageMime, type DiaryEntry } from '../lib/trainingDiary'
import { downloadProof } from '../lib/trainingDiaryProof'

// Darstellung eines VORHANDENEN Nachweises (proof_status === 'present') für
// alle Lesenden: Eigentümer, Eltern, Trainer, sportliche Leitung. Die Zustände
// 'none' und 'purged' bleiben bei den Aufrufern — deren Texte unterscheiden sich
// je Sicht (Erfassung vs. Mannschaftsübersicht).
//
// Bilder erscheinen als Vorschau und öffnen im Vollbild; alles andere (PDF)
// bekommt einen Download — inline anzeigen kann das Frontend es nicht.
export default function TrainingDiaryProofView({
  entry,
  thumbClassName,
  fileLabel = 'Nachweis herunterladen',
}: {
  entry: DiaryEntry
  thumbClassName: string
  fileLabel?: string
}) {
  const [zoomed, setZoomed] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const url = `/training-diary/${entry.id}/proof`

  async function handleDownload() {
    setBusy(true)
    setError('')
    try {
      await downloadProof(entry)
    } catch {
      setError('Der Nachweis konnte nicht geladen werden.')
    } finally {
      setBusy(false)
    }
  }

  if (isImageMime(entry.proof_mime)) {
    return (
      <>
        <button
          type="button"
          onClick={() => setZoomed(true)}
          aria-label="Nachweis vergrößern"
          className="block cursor-zoom-in rounded-md focus:outline-none focus:ring-2 focus:ring-brand-yellow"
        >
          <AuthImage url={url} alt="Trainingsnachweis" className={thumbClassName} />
        </button>
        {zoomed && (
          <ImageLightbox url={url} alt="Trainingsnachweis" onClose={() => setZoomed(false)} />
        )}
      </>
    )
  }

  return (
    <>
      <button
        type="button"
        onClick={handleDownload}
        disabled={busy}
        className="inline-flex items-center gap-1 text-sm text-brand-text hover:underline disabled:opacity-40 disabled:cursor-not-allowed"
      >
        <FileText className="w-4 h-4" />
        {fileLabel}
        <Download className="w-4 h-4" />
      </button>
      {error && <p className="text-xs text-brand-danger">{error}</p>}
    </>
  )
}
