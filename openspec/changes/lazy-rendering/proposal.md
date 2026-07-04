## Why

Die Datenverkehrs-Analyse hat neben Über-Fetch (`list-endpoint-pagination`) und Voll-Refetch (`scoped-live-updates`, `efficient-data-loading-quickwins`) eine weitere Achse aufgedeckt: das Frontend **rendert und lädt Daten, die gar nicht sichtbar sind** bzw. wirft geladenen Zustand unnötig weg. Diese Verbesserungen kommen **ohne** API-Vertragsänderung aus — sie verändern nur, *wann* und *ob* das Frontend bereits existierende Endpoints aufruft.

Konkrete Fälle aus der Frontend-Analyse:

- **Keine Virtualisierung langer Listen.** Members (bis 1000+), Duty-Slots (bis 500), Chat-Historie (100) werden vollständig ins DOM gerendert, auch wenn nur ein Bruchteil im Viewport liegt — hoher RAM-/Render-Aufwand auf schwachen Mobilgeräten.
- **`VideosPage` setzt bei jedem `video-*`-SSE-Event auf Seite 0 zurück** (`fetchPage(0, true)`), verwirft alle nachgeladenen Seiten und die Scroll-Position und lädt erneut.
- **Eingeklappte/abgeleitete Inhalte werden eager geladen**, z. B. lädt `MeinTeamPage` alle Team-Rosters sequ– auch nicht sichtbarer Teams (Fokus-Filter greift erst clientseitig nach dem Laden).

## What Changes

- **Virtualisiertes Rendering langer Listen:** Members-, Duty-Slot- und Chat-Ansichten rendern nur die sichtbaren (plus Puffer-)Zeilen. Umsetzung ohne neues schweres NPM-Paket (leichtgewichtiges Windowing bzw. hand-rolled Intersection-Observer; RAM-Budget beachten). Kein Endpoint-Change — treibt bei Bedarf nur die vorhandenen „Mehr laden"-Aufrufe aus `list-endpoint-pagination`.
- **Geladene Seiten erhalten statt zurücksetzen:** `VideosPage` (und analoge paginierte Ansichten) verwerfen bei SSE-Events nicht mehr die geladenen Seiten. Neue/aktualisierte Elemente werden in den vorhandenen Bestand eingepflegt bzw. als „neue Einträge verfügbar"-Hinweis angezeigt; die Scroll-Position bleibt erhalten.
- **On-Demand statt eager:** Inhalte, die hinter Aufklapp-/Fokus-Interaktionen liegen (z. B. Rosters nicht-fokussierter Teams in `MeinTeamPage`, Detail-Aufklappungen), werden erst beim Sichtbarwerden geladen — kein Vorab-Laden ganzer Sammlungen, die der Nutzer evtl. nie öffnet.

## Capabilities

### Added Capabilities

- `lazy-rendering`: Frontend rendert/lädt nur sichtbare bzw. angeforderte Inhalte (Virtualisierung, On-Demand-Aufklappen) und bewahrt geladenen Paginierungs-Zustand über Live-Update-Events hinweg.

## Test-Anforderungen

Frontend-Tests (vitest); dieser Change ändert keine HTTP-Route, daher keine Backend-Route→Test-Zeilen.

| Bereich | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| Windowing | `renders_only_visible_rows` | Bei N≫Viewport-Zeilen sind nur die sichtbaren (+ Puffer) im DOM; Scrollen tauscht Zeilen aus. |
| VideosPage | `keeps_loaded_pages_on_sse_event` | Nach einem `video-*`-Event bleiben zuvor geladene Seiten + Scroll-Position erhalten. |
| On-Demand | `roster_loads_only_when_expanded` | Roster eines nicht-fokussierten Teams wird erst beim Aufklappen/Fokus geladen, nicht beim Mount. |

**Garantierte Invariante:** Lazy-Rendering ändert nur, *was gerendert/wann geladen* wird — nie *welche Daten* ein Nutzer sehen darf. Alle Elemente bleiben durch Scrollen/Aufklappen erreichbar; nichts wird dauerhaft ausgeblendet.

## Mess-Anforderungen

Dieser Change senkt primär Render-/RAM-Last und **Request-Zahl**, nicht die Payload einer einzelnen Antwort — die Server-Payload-Baseline (`payload-measurement-harness`) ist daher nur teilweise einschlägig.

| Kennzahl | Werkzeug | Erwartung |
|---|---|---|
| DOM-Knoten langer Listen | Frontend-Test/Profiler | konstant bzgl. Listengröße (nur Viewport gerendert). |
| Requests je `video-*`-Event auf `VideosPage` | Frontend-Test (Request-Spy) | 0 zusätzliche Voll-Refetches; kein Reset auf Seite 0. |
| Requests beim Mount von `MeinTeamPage` | Frontend-Test (Request-Spy) | nur fokussierte/sichtbare Rosters werden geladen. |

## Impact

- **Frontend:** `MembersPage`, `DutyPage`/`DutySlotList`, `ChatPage` (Windowing); `VideosPage` (Seiten erhalten); `MeinTeamPage` (On-Demand-Rosters); ggf. geteilte Windowing-Komponente in `web/src/components/`.
- **Kein** Backend-, Schema-, Migrations- oder API-Change. **Kein** schweres neues NPM-Paket (RAM-Budget VPS/Mobilgerät).
- **Abhängigkeit:** ergänzt `list-endpoint-pagination` (Windowing treibt „Mehr laden") und `scoped-live-updates`/`efficient-data-loading-quickwins` (weniger/gebündelte Events → weniger Reset-Anlässe), ist aber unabhängig lauffähig.
