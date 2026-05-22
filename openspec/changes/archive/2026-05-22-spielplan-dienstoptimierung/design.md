## Context

`applyBehavior()` und `loadSameDayContext()` sind in `internal/games/handler.go` vollständig implementiert und funktional. Das Problem ist, dass sie nicht überall aufgerufen werden:

**PreviewSlots** (`GET /admin/duty-templates/{id}/preview`): Optimierung wird nur angewendet wenn `game_id` übergeben wird (Regenerierung). Für neue Spiele fehlt ein `date`-Parameter — `allGameTimes` bleibt leer, `applyBehavior` wird nicht aufgerufen.

**CreateGame** (`POST /admin/games`): Empfängt die bereits vom Frontend berechneten Slots (nach PreviewSlots), ruft aber `applyBehavior` nicht selbst auf. Da Preview nicht optimiert war, werden unoptimierte Slots gespeichert. Außerdem: `template_id` wird im Request-Struct akzeptiert, aber nie in die `games`-Tabelle gespeichert.

**RegenerateSlots** (`POST /admin/games/{id}/regenerate`): Ruft `loadSameDayContext` und `applyBehavior` korrekt auf — aber empfängt die Slots aus dem Request-Body statt aus einem Template. `SpieltagDetailPage` verwendet eine nicht existierende URL (`GET /admin/game-template/preview`), weshalb der Regenerierungs-Flow komplett defekt ist.

## Goals / Non-Goals

**Goals:**
- Dienstoptimierung wird bei neuen Spielen korrekt im Preview angezeigt
- `games.template_id` wird gespeichert und in `GetGame` zurückgegeben
- Regenerierungs-Dialog in SpieltagDetailPage funktioniert: Template-Auswahl, Live-Preview, Fehlerfall
- RegenerateSlots generiert Slots server-seitig aus Template (keine Slots im Request-Body mehr nötig)

**Non-Goals:**
- Optimierung für Auswärtsspiele oder generische Events (nur Heimspiele zählen als Kontext — bestehende Logik bleibt)
- Änderung der `applyBehavior`-Logik selbst
- Duty-Slots nachträglich bei UpdateGame neu berechnen

## Decisions

### D1: PreviewSlots erhält `date`-Parameter statt Neuschreiben

**Problem**: Preview für neue Spiele hat keinen Same-Day-Kontext.  
**Entscheidung**: Neuer optionaler Query-Parameter `date`. Wenn gegeben (ohne `game_id`): `loadSameDayContext` mit `date` + aktiver Season aufrufen, dann die Zeit des neuen Spiels selbst in `allGameTimes` einfügen (sorted), damit mehrere gleichzeitige neue Spiele korrekt zählen.  
**Alternative**: Neuen separaten Preview-Endpunkt für neue Spiele — abgelehnt, da unnötige Duplizierung.

### D2: RegenerateSlots lädt Template aus DB statt Slots aus Body

**Problem**: Aktuell übergibt das Frontend die Slots explizit im Request-Body. Das ist umständlich und fehleranfällig.  
**Entscheidung**: RegenerateSlots empfängt nur `template_id` im Body, lädt Template-Items selbst aus DB, wendet `applyBehavior` an, generiert Slots. Kein `slots`-Array im Request mehr nötig.  
**Vorteil**: Optimierung immer server-seitig korrekt; Frontend muss keine Slot-Daten zwischenspeichern.  
**Alternative**: Slots weiter aus Body — abgelehnt, da Preview-Daten im Frontend aufwändig zu halten sind.

### D3: CreateGame — Slots weiter aus Body, template_id nur speichern

**Problem**: CreateGame soll `template_id` speichern.  
**Entscheidung**: Slots kommen weiterhin aus dem Request-Body (berechnet durch PreviewSlots-Fix). Nur `template_id` wird zusätzlich in der `games`-Tabelle gespeichert. Keine doppelte Slot-Generierung im CreateGame.  
**Warum nicht template-basiert wie RegenerateSlots**: Bei neuen Spielen muss der User die Preview-Slots bestätigen (er wählt Template aus und sieht Vorschau). Die optimierten Slots kommen dann vom Fix in PreviewSlots.

### D4: Migration 004 — nullable `template_id`

`ALTER TABLE games ADD COLUMN template_id INTEGER REFERENCES game_templates(id)` — nullable. Bestehende Spiele ohne Template-Zuordnung bleiben gültig.

## Risks / Trade-offs

- **Regen-Dialog ohne gespeichertes Template**: Nutzer muss Template manuell wählen. Fehlerfall zeigt klare Meldung: "Kein Template gespeichert – bitte Template wählen". Kein silent failure.
- **Preview-Date-Parameter**: Wenn `date` und `game_id` beide übergeben werden, hat `game_id` Vorrang (bestehende Logik). Das ist korrekt für Regenerierung.
- **Alte Spiele ohne `template_id`**: Im Regen-Dialog ist kein Template vorausgewählt — der Nutzer muss es manuell auswählen. Akzeptabel.

## Migration Plan

1. Migration 004 SQL: `ALTER TABLE games ADD COLUMN template_id INTEGER REFERENCES game_templates(id);`
2. Backend-Änderungen: `handler.go` — PreviewSlots, CreateGame, GetGame, RegenerateSlots
3. Frontend-Änderungen: SpielplanPage, SpieltagDetailPage
4. Deploy: `make deploy` führt `migrate up` automatisch aus

Rollback: `make migrate-down` entfernt `template_id`-Spalte (via Down-Migration DROP COLUMN oder Tabellen-Rebuild).

## Open Questions

- Soll `RegenerateSlots` auch `template_id` in `games` updaten, wenn der Nutzer beim Regenerieren ein anderes Template wählt? → **Ja**, auf Wunsch des Users: beim Regenerieren gewähltes Template wird als neues `template_id` gespeichert.
