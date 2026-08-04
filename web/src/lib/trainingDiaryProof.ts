import { api } from './api'
import { compressImage } from './imageCompress'
import { PROOF_MAX_EDGE, PROOF_TARGET_BYTES, type DiaryEntry } from './trainingDiary'

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
