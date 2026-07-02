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
// expliziter User-Aktion geladen (keine passive Angriffsfläche) und trägt
// `crossorigin="anonymous"` (verhindert Cookie-Leak an gstatic).

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

let sdkPromise: Promise<boolean> | null = null

// loadCastSDK injiziert das Google-Cast-Sender-Skript und resolvt, sobald die
// Cast-Framework-API im Browser verfügbar (oder abschließend als unverfügbar
// markiert) ist. Single-Flight über modul-lokales Promise — mehrfacher Aufruf
// lädt das Skript nicht mehrfach.
export function loadCastSDK(): Promise<boolean> {
  if (sdkPromise) return sdkPromise
  sdkPromise = new Promise((resolve) => {
    window.__onGCastApiAvailable = (available) => resolve(available)
    const s = document.createElement('script')
    s.src = CAST_SDK_URL
    // Cookie-Leak an gstatic verhindern. Kein `integrity=` (siehe Modul-Kommentar).
    s.crossOrigin = 'anonymous'
    s.async = true
    s.onerror = () => resolve(false)
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
