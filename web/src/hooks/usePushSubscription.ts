import { useEffect } from 'react'
import { api } from '../lib/api'

function urlBase64ToUint8Array(base64String: string): ArrayBuffer {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = atob(base64)
  const arr = Uint8Array.from([...rawData].map((c) => c.charCodeAt(0)))
  return arr.buffer as ArrayBuffer
}

export function usePushSubscription() {
  useEffect(() => {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) return

    const isIOS = /iphone|ipad|ipod/i.test(navigator.userAgent)
    if (isIOS && !window.matchMedia('(display-mode: standalone)').matches) return

    if (Notification.permission === 'denied') return

    const subscribe = async () => {
      try {
        const { data } = await api.get<{ publicKey: string }>('/push/vapid-public-key')
        const registration = await navigator.serviceWorker.ready
        const subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(data.publicKey),
        })
        const json = subscription.toJSON()
        const keys = json.keys as { p256dh: string; auth: string }
        await api.post('/push/subscribe', {
          endpoint: subscription.endpoint,
          p256dh: keys.p256dh,
          auth: keys.auth,
        })
      } catch (err) {
        // Beobachtbar loggen statt still verwerfen: ein fehlgeschlagenes
        // (Re-)Subscribe — z.B. applicationServerKey-Mismatch (InvalidStateError)
        // oder Netzwerkfehler beim POST /push/subscribe — bleibt sonst
        // unbemerkt und der Nutzer erhält dauerhaft keine Pushes mehr.
        console.warn('[push] subscribe failed', err)
      }
    }

    subscribe()
  }, [])
}
