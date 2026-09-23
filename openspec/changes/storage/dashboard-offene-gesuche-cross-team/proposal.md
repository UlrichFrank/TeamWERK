## Why

Folge-Proposal zu `dashboard-offene-gesuche`. Teil 1 zeigt offene Mitfahr-Gesuche nur zu Spielen der **eigenen** Teams. Fahrgemeinschaften bilden sich aber oft **teamübergreifend**: Wenn zwei Mannschaften am selben Tag am selben Ort spielen, lohnt sich eine gemeinsame Fahrt — unabhängig davon, in welchem Team man ist.

Im Datenmodell bekommt jede Mannschaft eine **eigene `games`-Zeile** (kein Dedup), jede mit eigener `venue_id` (FK auf `venues`, nullable). „Gleicher Tag + gleicher Ort" ist also `same date AND same venue_id` über zwei verschiedene `games`-Zeilen. Genau dann sollen auch Gesuche fremder Teams sichtbar werden.

## What Changes

- `queryCarpoolingOpenRequests` (aus Teil 1) wird erweitert: ausgehend von den eigenen nächsten ≤3 Spielen (Anker) werden **kolozierte Fremdspiele** dazugenommen — andere `games`-Zeilen mit gleichem `date` UND gleichem `venue_id` (nur wenn `venue_id IS NOT NULL`).
- **Gruppierung umgestellt** von „pro Spiel" auf **Pool nach (Tag, Venue)**: Ein Block `date · venue.name` listet alle offenen Gesuche aller dort/dann stattfindenden Spiele, teamübergreifend gemischt. Pro Gesuch wird der Spiel-/Team-Kontext ergänzt, damit erkennbar bleibt, zu welchem Anlass jemand sucht.
- **Fallback ohne Venue:** Spiele mit `venue_id IS NULL` matchen nicht über Teamgrenzen und bleiben ein Block pro Spiel (Verhalten wie Teil 1).
- „Offen"-Definition unverändert (kein `confirmed`; `pending` zählt als offen).
- Frontend: Block „Offene Gesuche" rendert Gruppen nach (Tag, Ort) inkl. Spiel-/Team-Kontext je Eintrag.

## Capabilities

### Modified Capabilities

- `dashboard-offene-gesuche`: Anzeige der offenen Gesuche wird um kolozierte Fremdteam-Spiele (gleicher Tag + Venue) erweitert und nach (Tag, Venue) gruppiert.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_CrossTeamSameVenue` | Offenes Gesuch eines Fremdteams an anderem Spiel mit **gleichem** `date`+`venue_id` erscheint im selben (Tag, Venue)-Pool. |
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_CrossTeamDifferentVenue` | Fremdteam-Gesuch mit **anderem** `venue_id` oder anderem Tag erscheint **nicht**. |
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_NullVenueNoCrossMatch` | Hat eines der beiden Spiele `venue_id IS NULL`, erfolgt **kein** Cross-Team-Match (Fallback: Block pro Spiel). |
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_PoolMerge` | Zwei Spiele mit gleichem (Tag, Venue) liefern ihre Gesuche **unter einer** Gruppe gemerged. |

**Garantierte Invariante:** Ein Fremdteam-Gesuch erscheint genau dann, wenn sein Spiel `date` UND `venue_id` (nicht NULL) mit einem der eigenen Anker-Spiele teilt und keine `confirmed`-Paarung darauf existiert.

## Impact

- **Datei:** `internal/dashboard/handler.go` — Query um Kolokations-Join erweitern, Gruppierung auf (Tag, Venue) umstellen, Venue-Name + Spiel-Kontext mitliefern.
- **Datei:** `internal/dashboard/handler_test.go` — Cross-Team-/Pool-/Null-Venue-Tests.
- **Datei:** `web/src/pages/DashboardPage.tsx` — Block nach (Tag, Ort) gruppieren, Kontext je Eintrag anzeigen.
- **Bewusste Datensichtbarkeit:** Erstmals werden Namen von Mitgliedern **fremder Teams** im Dashboard sichtbar — ausschließlich bei Übereinstimmung von Tag + Venue. Bewusst akzeptiert; in `design.md` festgehalten.
- **Datenqualitäts-Abhängigkeit:** Cross-Team-Matching wirkt nur bei gepflegtem `venue_id`.
- **Kein** Schema-/Migrations-Change, **keine** neue Route.
