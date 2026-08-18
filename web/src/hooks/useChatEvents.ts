import { useAuth } from '../contexts/AuthContext'
import { useEventStream } from './useEventStream'

// Chat-Ereigniskanal. Die Verbindungsführung (Reconnect, Bindung an die
// Identität, Aufräumen) liegt in useEventStream — hier bleibt nur die Route.
//
// Kein `?token=` in der URL: `/api/chat/events` hängt serverseitig an
// auth.CookieMiddleware und wertet den Query-Parameter gar nicht aus. Der Token
// wurde also wirkungslos mitgeschickt und landete dabei in nginx-Access- und
// Proxy-Logs — genau das, was die Spec sse-live-updates für /api/events schon
// verbietet und was dieser später gebaute Kanal nie übernommen hatte.
export function useChatEvents(onEvent: (eventType: string) => void) {
  const { user } = useAuth()
  useEventStream('/api/chat/events', onEvent, user?.id ?? null)
}
