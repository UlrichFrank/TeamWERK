# Design

## Context

Siehe proposal.md. `duty_accounts` wird an vier Stellen berührt: Claim (`INSERT OR IGNORE`),
DeleteGame (Neuberechnung von `ist`), DeleteUser (expliziter `DELETE`) und die beiden
Lese-Routen. Die Dienst-Bilanz (`internal/dutyfairness`) ist davon vollständig unabhängig.

## Goals / Non-Goals

**Goals:** Kein Anwendungscode berührt `duty_accounts` mehr; die beiden unbenutzten Routen sind weg.

**Non-Goals:**
- `DROP TABLE duty_accounts` in diesem Change (Decision 1).
- `duty_season_targets` / `PUT /api/seasons/{id}/duty-targets`: ebenfalls schreib-only und
  ohne Leser, aber nicht Teil des Befunds; eigener Aufräum-Change.

## Decisions

1. **Tabelle erst im zweiten Schritt droppen.** Laut `docs/agent/10-deployment.md` sind
   Migrationen additiv, damit `make deploy-rollback` ohne `migrate down` funktioniert. Das
   Vorgänger-Binary schreibt `duty_accounts` in DeleteGame (Fehler → HTTP 500 beim Löschen
   eines Termins mit erledigten Diensten) und DeleteUser (Fehler → Nutzer nicht löschbar).
   Ein Drop im selben Deploy machte den Rollback also funktional kaputt. Die Folge-Migration
   `DROP TABLE duty_accounts` ist gefahrlos, sobald kein Binary mit Zugriff mehr zurückgerollt
   werden kann.
2. **DeleteUser verlässt sich auf `ON DELETE CASCADE`.** `duty_accounts.user_id` trägt
   `REFERENCES users(id) ON DELETE CASCADE`, `foreign_keys=ON` ist an jeder Verbindung gesetzt
   — der explizite `DELETE` ist redundant und fällt mit dem Drop ohnehin weg.
3. **Kein Ersatz für den CSV-Export.** Er hatte keinen Aufrufer in der Oberfläche; wer eine
   Auswertung braucht, bekommt sie aus der Vorstands-Rangliste.

## Risks / Trade-offs

- [Externer Aufrufer der entfernten Routen] → keiner bekannt; Frontend und Desktop-Tool nutzen sie nicht. → 404 statt falscher Zahl.
- [Tabelle bleibt zeitweise als tote Struktur] → bewusst, bis zur Folge-Migration.

## Migration Plan

Normaler Deploy. Rollback per `make deploy-rollback` bleibt funktionsfähig, weil die Tabelle
existiert. Danach Folge-Change mit `DROP TABLE duty_accounts`.
