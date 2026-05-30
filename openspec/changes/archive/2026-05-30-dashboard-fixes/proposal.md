## Why

Das bestehende Dashboard hat drei kleine aber nutzungsrelevante Lücken: Der Begriff „Nächste Spiele" passt nicht zu generischen Events, Klicks auf Events führen ins Leere statt zum Kalender, und die Dienstkonto-Soll-Berechnung für Elternteile ist pauschal (5 × Kinder) statt aus echten Saisondaten abgeleitet.

## What Changes

- **Umbenennung:** Accordion-Sektion „Nächste Spiele" → „Nächste Events" im DashboardPage
- **Kalender-Jump:** Klick auf ein Event in der Nächste-Events-Sektion navigiert zu `/kalender?date=YYYY-MM-DD`; KalenderPage initialisiert `year`/`month` aus dem `?date`-Query-Param
- **Kader-Feld `games_per_season`:** Neues Integer-Feld auf der `kader`-Tabelle (saison-spezifisch), editierbar in AdminKaderPage rechts neben Altersklasse; Migration 012
- **Neue Dienstkonto-Formel:** `soll` für Elternteil wird dynamisch aus Kader-Daten und aktiven Spielplan-Templates berechnet statt pauschal `5 × Anzahl Kinder`
- **Erklärtext angepasst:** DutyAccountTile zeigt für Elternteile „Ziel: {soll} Dienste (Saison {name})" statt bisheriger Formel-Erklärung

## Capabilities

### New Capabilities

- `kalender-date-param`: KalenderPage akzeptiert `?date=YYYY-MM-DD` URL-Parameter zur direkten Monatsnavigation
- `kader-games-per-season`: Kader-Einträge speichern die Anzahl Saisonspiele; editierbar durch Admin/Vorstand im Kader-UI
- `dienstkonto-dynamische-soll-formel`: Soll-Berechnung für Elternteile basiert auf Kader-Spielanzahl, Template-Slot-Summen und Elternteil-Anzahl pro Kind

### Modified Capabilities

- `duties`: Soll-Berechnung im Dashboard-Handler ändert sich; kein Verhaltensbruch für Trainer/Admin/Spieler

## Impact

**Backend:**
- `internal/db/migrations/012_kader_games_per_season.up/.down.sql`
- `internal/dashboard/handler.go`: `computeSoll()`-Funktion ersetzt durch neue Logik mit JOIN auf `kader`, `kader_members`, `game_templates`, `game_template_items`, `family_links`
- `internal/config/handler.go` oder neuer `internal/kader/handler.go`: PUT-Endpoint für `games_per_season`

**Frontend:**
- `web/src/pages/DashboardPage.tsx`: Accordion-Titel + Link-Ziel + Erklärtext
- `web/src/pages/KalenderPage.tsx`: `useSearchParams` + State-Initialisierung
- `web/src/pages/AdminKaderPage.tsx`: Neues Input-Feld für `games_per_season`

**Keine Breaking Changes** für externe Clients. Keine neuen Abhängigkeiten.
