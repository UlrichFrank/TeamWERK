import { api } from './api'
import { compressImage } from './imageCompress'
import { openBlobNatively } from './openFileNatively'
import { fmtDate, PROOF_MAX_EDGE, PROOF_TARGET_BYTES, type DiaryEntry } from './trainingDiary'

// Bilder werden vor dem Upload clientseitig verkleinert — deutlich enger als
// im Chat, weil ein Nachweis nur lesbar sein muss. Nicht-Bilder (PDF) gehen
// unverändert raus; der Server prüft Typ und Größe ohnehin erneut.
//
// Der Server akzeptiert kein WebP-fremdes Exotenformat: was der Browser nicht
// dekodieren kann (HEIC auf Chrome/Firefox), reicht compressImage unverändert
// durch und der Server antwortet mit HTTP 400. Das ist gewollt sichtbar.
export async function uploadProof(entryId: number, file: File): Promise<DiaryEntry> {
  let blob: Blob = file
  let fileName = file.name

  if (file.type.startsWith('image/')) {
    const result = await compressImage(file, {
      targetBytes: PROOF_TARGET_BYTES,
      maxEdge: PROOF_MAX_EDGE,
    })
    blob = result.blob
    fileName = result.fileName
  }

  const form = new FormData()
  form.append('proof', blob, fileName)
  const { data } = await api.post<DiaryEntry>(`/training-diary/${entryId}/proof`, form)
  return data
}

export async function deleteProof(entryId: number): Promise<void> {
  await api.delete(`/training-diary/${entryId}/proof`)
}

// Endungen der Server-Whitelist (trainingdiary.extByMime). Ein unbekannter Typ
// kann nur aus einer künftigen Whitelist-Erweiterung stammen — dann lieber ohne
// Endung speichern als mit einer falschen.
const PROOF_EXT: Record<string, string> = {
  'image/jpeg': 'jpg',
  'image/png': 'png',
  'image/webp': 'webp',
  'application/pdf': 'pdf',
}

// Lädt den Nachweis über axios (Bearer + Auto-Refresh) und übergibt ihn dem
// nativen Viewer/Download des Geräts. Nötig für PDFs — die kann das Frontend
// nicht inline zeigen — und als Zoom-Ausweg für Bilder auf Mobilgeräten.
// Der Dateiname trägt das Trainingsdatum, damit mehrere Nachweise im
// Download-Ordner unterscheidbar bleiben.
export async function downloadProof(
  entry: Pick<DiaryEntry, 'id' | 'trained_on' | 'proof_mime'>,
): Promise<void> {
  const res = await api.get<Blob>(`/training-diary/${entry.id}/proof`, {
    responseType: 'blob',
  })
  const ext = PROOF_EXT[entry.proof_mime ?? '']
  const stamp = fmtDate(entry.trained_on).replace(/\./g, '-')
  openBlobNatively(res.data, `trainingsnachweis-${stamp}${ext ? `.${ext}` : ''}`)
}
