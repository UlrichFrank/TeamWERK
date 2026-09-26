// Google-Cast-Sender-Anbindung (video-tv-streaming).
//
// Lädt das Cast-SDK von Google **erst bei Bedarf** — kein passiver Google-Ping
// beim Seitenaufruf, DSGVO-neutral im Default-Zustand. Aufrufsite:
// `loadCastSDK()` beim ersten Klick auf den Cast-Button.
//
// SRI (Subresource Integrity) wird BEWUSST NICHT gesetzt: Google versioniert
// `cast_sender.js` serverseitig ohne stabile Hashes und pusht regelmäßig
// Firmware-Kompat-Updates; ein fixer `integrity=`-Hash würde bei jedem
// Google-Update die Cast-Integration brechen. Restrisiko: kompromittierter
// gstatic-Endpunkt liefert bösartiges JS. Mitigation: Skript wird nur nach
// expliziter User-Aktion geladen (keine passive Angriffsfläche).
//
// KEIN `crossorigin`-Attribut: es macht den Abruf zu einem CORS-Request, und
// gstatic liefert für `cast_sender.js` keinen `Access-Control-Allow-Origin` —
// Chrome verwirft das Skript dann (gemessen 09/2026). Die nachgeladenen
// Skripte (`cast_framework.js`, `eureka/clank/…`) kommen ebenfalls von
// www.gstatic.com und brauchen denselben CSP-Eintrag `script-src`
// (`internal/app/security_headers.go` + `deploy/nginx-teamwerk.conf`).

declare global {
  interface Window {
    // Cast-SDK-Ready-Callback (siehe Google-Docs).
    __onGCastApiAvailable?: (available: boolean) => void
    chrome?: {
      cast?: {
        media: {
          DEFAULT_MEDIA_RECEIVER_APP_ID: string
          MediaInfo: new (contentId: string, contentType: string) => CastMediaInfo
          LoadRequest: new (mediaInfo: CastMediaInfo) => unknown
          StreamType: { BUFFERED: string }
        }
        AutoJoinPolicy: { ORIGIN_SCOPED: string }
      }
    }
    cast?: {
      framework: {
        CastContext: {
          getInstance: () => CastContextInstance
        }
      }
    }
  }
}

interface CastMediaInfo {
  streamType?: string
}

interface CastContextInstance {
  setOptions: (opts: { receiverApplicationId: string; autoJoinPolicy: string }) => void
  requestSession: () => Promise<CastSession>
}

interface CastSession {
  loadMedia: (req: unknown) => Promise<void>
}

const CAST_SDK_URL =
  'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1'

// Ergebnis eines Ladeversuchs. Die beiden Fehlschläge sind bewusst getrennt:
//   - 'unavailable': Skript geladen, aber der Browser kann nicht casten
//     (Safari, Firefox, Chrome auf iOS) — endgültig, ein zweiter Versuch ändert nichts.
//   - 'load_failed': Skript kam nicht an (CSP, Adblocker, Netz) — der Browser
//     könnte casten, ein erneuter Versuch ist sinnvoll.
// Beides als ein `false` zusammenzufassen hat einen CSP-Fehler lange als
// „Browser kann nicht casten" getarnt: der Button verschwand kommentarlos.
export type CastLoadResult = 'available' | 'unavailable' | 'load_failed'

// cast_sender.js lädt cast_framework.js selbst nach. Scheitert DAS (z. B. an
// der CSP), feuert weder `onerror` unseres Skripts noch `__onGCastApiAvailable`
// — ohne Timeout hinge der Klick für immer.
export const CAST_LOAD_TIMEOUT_MS = 10_000

let sdkPromise: Promise<CastLoadResult> | null = null

// loadCastSDK injiziert das Google-Cast-Sender-Skript und resolvt, sobald die
// Cast-Framework-API im Browser verfügbar (oder abschließend als unverfügbar
// markiert) ist. Single-Flight über modul-lokales Promise — mehrfacher Aufruf
// lädt das Skript nicht mehrfach. Nach 'load_failed' wird der Cache verworfen,
// damit ein erneuter Klick das Skript neu anfordert.
export function loadCastSDK(): Promise<CastLoadResult> {
  if (sdkPromise) return sdkPromise
  sdkPromise = new Promise((resolve) => {
    const s = document.createElement('script')
    const fail = () => {
      clearTimeout(timer)
      s.remove()
      sdkPromise = null
      resolve('load_failed')
    }
    const timer = setTimeout(fail, CAST_LOAD_TIMEOUT_MS)
    window.__onGCastApiAvailable = (available) => {
      if (!available) {
        clearTimeout(timer)
        resolve('unavailable')
        return
      }
      // `true` heißt nur „der Browser kann casten" — ob cast_framework.js
      // tatsächlich ankam, sagt der Callback nicht (bei CSP-Block kommt trotzdem
      // `true`, gemessen). Fehlt das Framework, ist es ein Ladefehler.
      if (isCastAvailable()) {
        clearTimeout(timer)
        resolve('available')
      } else {
        fail()
      }
    }
    s.src = CAST_SDK_URL
    // Weder `integrity=` noch `crossorigin` (siehe Modul-Kommentar).
    s.async = true
    s.onerror = fail
    document.head.appendChild(s)
  })
  return sdkPromise
}

// startCastSession startet eine Cast-Session zum Default-Media-Receiver und
// lädt die gegebene HLS-Master-URL (inkl. ?st=-Token) als BUFFERED-Stream.
// Aufrufer stellt sicher, dass `loadCastSDK()` vorher erfolgreich resolved hat.
export async function startCastSession(masterURL: string): Promise<void> {
  const cc = window.chrome?.cast
  const framework = window.cast?.framework
  if (!cc || !framework) {
    throw new Error('Cast SDK not available')
  }
  const context = framework.CastContext.getInstance()
  context.setOptions({
    receiverApplicationId: cc.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
    autoJoinPolicy: cc.AutoJoinPolicy.ORIGIN_SCOPED,
  })
  const session = await context.requestSession()
  const media = new cc.media.MediaInfo(masterURL, 'application/vnd.apple.mpegurl')
  media.streamType = cc.media.StreamType.BUFFERED
  await session.loadMedia(new cc.media.LoadRequest(media))
}

// isCastAvailable prüft synchron, ob die Cast-Framework-API bereits injiziert
// wurde. Wird vom CastButton nach `loadCastSDK()`-Auflösung benutzt, um sein
// Rendering zu entscheiden.
export function isCastAvailable(): boolean {
  return Boolean(window.chrome?.cast && window.cast?.framework)
}
