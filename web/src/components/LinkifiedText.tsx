import { Fragment, type MouseEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { inAppPath, isDocumentLink, splitLinks } from '../lib/linkify'

const MOBILE_QUERY = '(max-width: 639px)'

function isMobileViewport(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia(MOBILE_QUERY).matches
}

function isModifiedClick(e: MouseEvent): boolean {
  return e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey
}

interface Props {
  text: string
  linkClassName?: string
}

/**
 * Freitext mit klickbaren URLs (Chat, Termin-Hinweis).
 *
 * - Fremde URLs öffnen in einem neuen Tab.
 * - App-Links (dieselbe Origin) navigieren **innerhalb** der SPA. Ein
 *   `target="_blank"` würde in der iOS-Homescreen-PWA aus der App heraus in ein
 *   Safari-Blatt ohne Sitzung springen; ein voller Seitenwechsel verlöre den
 *   App-Zustand und damit einen sauberen Rückweg.
 * - Ausnahme Dokument-Links am Desktop: `/dokumente/datei/:id` ersetzt dort den
 *   Tab durch den nativen PDF-Viewer (`DocumentFileLinkPage`). Deshalb ein neuer
 *   Tab — die App bleibt im Ursprungstab stehen, statt dass ein installiertes
 *   Desktop-PWA-Fenster ohne Zurück-Knopf auf der Datei endet. Mobil läuft der
 *   Link in den In-App-Viewer, dessen „Zurück" auf den Termin zurückführt.
 *
 * Klicks stoppen die Propagation: der Text steht oft in klickbaren Karten.
 */
export default function LinkifiedText({ text, linkClassName = 'underline break-all' }: Props) {
  const navigate = useNavigate()

  return (
    <>
      {splitLinks(text).map((part, i) => {
        if (part.kind === 'text') return <Fragment key={i}>{part.value}</Fragment>

        const path = inAppPath(part.href, window.location.origin)
        if (path === null) {
          return (
            <a
              key={i}
              href={part.href}
              target="_blank"
              rel="noopener noreferrer"
              className={linkClassName}
              onClick={(e) => e.stopPropagation()}
            >
              {part.href}
            </a>
          )
        }

        return (
          <a
            key={i}
            href={part.href}
            className={linkClassName}
            onClick={(e) => {
              e.stopPropagation()
              // Strg/Cmd/Mittelklick: dem Browser überlassen (neuer Tab/Fenster).
              if (isModifiedClick(e)) return
              e.preventDefault()
              if (isDocumentLink(path) && !isMobileViewport()) {
                window.open(path, '_blank', 'noopener')
                return
              }
              navigate(path)
            }}
          >
            {part.href}
          </a>
        )
      })}
    </>
  )
}
