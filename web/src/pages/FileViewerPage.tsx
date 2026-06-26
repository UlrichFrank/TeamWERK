import { useParams } from 'react-router-dom'
import FileViewer from '../components/FileViewer'

export default function FileViewerPage() {
  const { fileId } = useParams()
  const id = Number(fileId)

  if (!fileId || Number.isNaN(id)) {
    return <p className="text-sm text-brand-danger">Ungültige Datei-ID.</p>
  }

  return <FileViewer source="file" fileId={id} fallbackPath="/dokumente" />
}
