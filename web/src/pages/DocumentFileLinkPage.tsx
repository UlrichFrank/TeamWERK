import { Navigate, useParams } from 'react-router-dom'

// Deep-Link-Kompatibilität: alte /dokumente/datei/:fileId-Links leiten auf
// den neuen In-App-Viewer um.
export default function DocumentFileLinkPage() {
  const { fileId } = useParams()
  if (!fileId) return <Navigate to="/dokumente" replace />
  return <Navigate to={`/dokumente/anzeigen/${fileId}`} replace />
}
