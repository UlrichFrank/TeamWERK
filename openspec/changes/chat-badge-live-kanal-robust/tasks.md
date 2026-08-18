## 1. Gemeinsame SSE-Basis

- [x] 1.1 `web/src/hooks/useEventStream.ts` anlegen: Hook `useEventStream(path, onEvent, identity)`, der die EventSource an die Nutzer-Identität bindet, bei `readyState === CLOSED` mit Backoff neu verbindet, `onopen` den Backoff zurücksetzt und beim Unmount Verbindung **und** Timer aufräumt.
- [x] 1.2 Backoff als reine, exportierte Funktion `nextBackoffMs(attempt)` (1 s → 2 s → 4 s → 8 s → 16 s → 30 s, danach konstant) — testbar ohne Render.
- [x] 1.3 Sofortiger Reconnect-Versuch mit zurückgesetztem Backoff, wenn das Dokument wieder sichtbar wird oder der Browser `online` meldet.
- [x] 1.4 Tests `web/src/hooks/useEventStream.test.ts` (neben dem Hook, wie `useLiveUpdates.test.tsx`): Backoff-Folge inkl. Obergrenze; Wiederaufbau nach `CLOSED`; transienter Fehler bleibt beim Browser; kein Reconnect nach Unmount; Neuaufbau bei Identitätswechsel; Aufräumen von Timer und Listenern.

## 2. Chat-Kanal auf die Basis umstellen

- [x] 2.1 `useChatEvents` auf `useEventStream` umstellen: URL exakt `/api/chat/events` **ohne** `?token=`, Bindung an `user` statt `[]`.
- [x] 2.2 Test: Der Hook öffnet keine URL mit Query-Parameter (Regressionsschutz gegen den Token im Log) und baut bei Identitätswechsel neu auf.

## 3. Globalen Kanal auf die Basis umstellen

- [ ] 3.1 `useLiveUpdates` auf `useEventStream` umstellen; Coalescing (300 ms je Event-Typ) und `invalidateReferenceCache` unverändert erhalten.
- [ ] 3.2 Test: Coalescing-Verhalten bleibt unverändert; nach einem Verbindungsabbruch werden erneut Events verarbeitet.

## 4. Zähler-Aktualität in AppShell

- [ ] 4.1 `chatUnread` um den Zustand „schon einmal erfolgreich geladen" ergänzen; Anzeigestellen zeigen vor dem ersten Erfolg keinen Badge. Der App-Icon-Badge-Effekt bleibt unberührt.
- [ ] 4.2 `loadChatUnread`: fehlgeschlagener Versuch lässt den zuletzt bekannten Wert stehen und wird beim nächsten Auslöser nachgeholt (kein Sprung auf `0`, kein Fehlerdialog).
- [ ] 4.3 Refetch bei `visibilitychange` (sichtbar) und `online` registrieren, beim Unmount abmelden.
- [ ] 4.4 Tests in `web/src/components/__tests__/AppShell.unreadRefresh.test.tsx`: Refetch beim Sichtbarwerden; Refetch bei `online`; gescheiterter Start-Load zeigt keinen Badge und wird nachgeholt; erfolgreiche `0` zeigt keinen Badge; Listener werden beim Unmount entfernt.

## 5. Verifikation

- [ ] 5.1 `pnpm -C web test` und `pnpm -C web lint` grün; bestehende `AppShell.navBadges`-Tests unverändert grün.
- [ ] 5.2 `openspec validate chat-badge-live-kanal-robust --strict` grün.
- [ ] 5.3 Manuelle Gegenprobe laut `design.md` — Migration Plan (PWA in den Hintergrund, Nachricht senden, zurückholen).
