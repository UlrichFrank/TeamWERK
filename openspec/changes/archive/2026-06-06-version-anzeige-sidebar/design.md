## Context

Die App sendet bereits die `buildHash`-Version des Servers via SSE (`__version:<hash>`). Der bestehende `useVersionCheck`-Hook hört darauf, speichert aber die Version nur lokal in der Closure und gibt sie nicht nach außen. Ein Bug im `onerror`-Handler verhindert zuverlässige Reconnection nach Server-Neustarts.

## Goals / Non-Goals

**Goals:**
- `useVersionCheck` gibt neben `updateAvailable` auch `version` zurück (den ersten empfangenen Hash)
- Sidebar-Footer zeigt `v <hash>` unterhalb einer Trennlinie an
- SSE-Reconnection funktioniert zuverlässig nach Deployment

**Non-Goals:**
- Kein neuer Backend-Endpunkt
- Keine Version ins Vite-Bundle einbacken
- Keine Anzeige auf Mobile (Sidebar ist dort ein Overlay — die Version ist dort nicht nötig)

## Decisions

**useVersionCheck gibt Objekt zurück statt boolean**

```ts
// vorher
function useVersionCheck(): boolean

// nachher
function useVersionCheck(): { updateAvailable: boolean; version: string | null }
```

Alle bisherigen Konsumenten (`App.tsx`) werden angepasst. `AppShell.tsx` nutzt `version` für die Anzeige.

**onerror-Handler entfernen**

Der Browser reconnectet SSE automatisch gemäß Spec, solange `es.close()` nicht aufgerufen wird. Der bestehende Handler ruft `es.close()` bei `readyState === CLOSED` auf — was passiert wenn der Server während des Neustarts kurzzeitig unerreichbar ist. Lösung: Handler ersatzlos entfernen. Das Cleanup beim Unmount (`return () => es.close()`) bleibt bestehen.

**Version im DEV-Modus**

Im DEV-Modus (`import.meta.env.DEV`) gibt der Hook `version: null` zurück — die AppShell zeigt dann nichts an.

## Risks / Trade-offs

- [Risiko] Der Hash ist kryptisch (`a3f9c12`) — Nutzer können damit nichts anfangen → Mitigation: das ist gewollt, reine Kontrollinfo für Admins
- [Trade-off] Die Version erscheint erst nach dem ersten SSE-Event, nicht sofort beim Laden → akzeptabel, da SSE sehr schnell verbindet
