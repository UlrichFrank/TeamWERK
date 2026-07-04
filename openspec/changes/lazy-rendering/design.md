# Design — lazy-rendering

## Warum ohne API-Change

Dieser Change ist bewusst die frontend-only-Teilmenge der Lazy-Idee. Er ruft ausschließlich **bestehende** Endpoints (inkl. der paginierten aus `list-endpoint-pagination`) und verändert nur Timing/Umfang des Renderings. Das hält ihn risikoarm und unabhängig von Schema-Arbeit (die im Schwester-Change `incremental-sync` steckt).

## Windowing ohne schweres Paket

RAM-Budget (VPS 1 GB, schwache Mobilgeräte) und die Konvention „kein State-Manager" sprechen gegen große Virtualisierungs-Bibliotheken. Zwei zulässige Wege:

- **A:** ein schlankes Windowing (nur sichtbarer Bereich + kleiner Über-/Unter-Puffer), hand-rolled über `IntersectionObserver` bzw. Scroll-Offset-Rechnung.
- **B:** ein kleines, geprüftes Windowing-Utility, falls es das Bundle-/RAM-Budget nicht sprengt (Bundle-Delta in `make metrics` beobachten).

Entscheidung beim ersten umgesetzten Bereich; danach konsistent.

## VideosPage: Seiten erhalten statt Reset

Heute (`VideosPage.tsx`): `useLiveUpdates(video-* → fetchPage(0, true))` verwirft alle Seiten. Neu:

```
video-ready/updated  → betroffenes Element im vorhandenen Bestand patchen (per ID),
                        NICHT Liste zurücksetzen
video-queued         → "N neue Videos" — Hinweis-Chip, Nachladen auf Klick
video-deleted        → Element per ID aus Bestand entfernen
```

Fällt das SSE-Event aus (Buffer-1-Drop), bleibt der Bestand konsistent-genug; ein manuelles Neu-Laden bereinigt. Kein Datenverlust-Risiko, da rein darstellend.

## On-Demand-Aufklappen (MeinTeamPage-Muster)

Heute lädt `MeinTeamPage` `/teams/my` und dann alle Rosters; der Fokus-Filter greift erst clientseitig. Neu: Roster eines Teams wird erst geladen, wenn das Team fokussiert/aufgeklappt ist. Bereits geladene Rosters werden im Komponenten-State behalten (kein Re-Fetch beim erneuten Aufklappen innerhalb der Session).

## Abgrenzung zu incremental-sync

| Aspekt | lazy-rendering (dieser Change) | incremental-sync (#5b) |
|---|---|---|
| API-Vertrag | unverändert | neue Cursor-Parameter |
| Schema | unverändert | `updated_at` + Tombstones |
| Kern | *rendern/laden nur was sichtbar* | *übertragen nur was geändert* |
| Risiko | niedrig (nur Darstellung) | mittel (Delete-Tracking) |

## Invariante: Sichtbarkeit unverändert

Windowing/On-Demand blenden nichts dauerhaft aus. Jede Zeile ist durch Scrollen erreichbar, jedes Detail durch Aufklappen. Der Change fügt keine Filter hinzu und entfernt keine — er verzögert nur das Rendern/Laden.
