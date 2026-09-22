# Design

## Context

`internal/matchreports/create.go` implementiert bereits `DELETE /api/match-reports/{id}`
für `draft`/`publish_failed`, gated auf Autor-oder-Admin (`create.go:139-188`). Die
Autorisierungslogik für Freigeber existiert schon in `isReviewer(claims)`
(`create.go:193ff`, genutzt von `Pending`/`Publish`). Siehe proposal.md für die
Motivation.

## Goals / Non-Goals

**Goals:**
- `Delete`-Handler um den State `pending_review` erweitern, mit eigener
  Berechtigungslogik (Freigeber statt Autor).
- Bestehendes Verhalten für `draft`/`publish_failed` unverändert lassen.
- Frontend-Button für Freigeber auf der Detailseite.

**Non-Goals:**
- Kein Reject-Workflow, der den Bericht zurück an den Autor gibt (state bleibt
  `pending_review` → dann `deleted`, nicht `pending_review` → `draft`).
- Kein Soft-Delete / keine Papierkorb-Funktion — Löschen ist endgültig, wie bei
  `draft` heute schon.
- Kein Löschen-Button in der Listenübersicht (`MatchReportListPage.tsx`).

## Decisions

**Berechtigung getrennt nach State, nicht eine gemeinsame Bedingung.** Der bestehende
Check `authorID != claims.UserID && claims.Role != auth.RoleAdmin` gilt weiterhin
exakt für `draft`/`publish_failed`. Für `pending_review` wird eine zweite Verzweigung
mit `isReviewer(claims)` eingezogen. Alternative wäre eine einzige kombinierte
Bedingung (Autor ODER Freigeber, für alle drei States) gewesen — verworfen, weil das
den Autor auch für `pending_review` löschberechtigt machen würde, was der Proposal
explizit ausschließt (Autor hat mit Submit die Verfügung abgegeben).

**Bild-Aufräumen wiederverwenden.** `removeAllImageFiles(id)` wird unverändert auch
für den `pending_review`-Zweig aufgerufen — dieselbe Funktion, kein neuer Code.

**Frontend-Button-Platzierung.** Der neue Button erscheint in der bestehenden
Button-Reihe (`canEdit`-Block) neben „Veröffentlichen", sichtbar wenn
`report.state === 'pending_review' && isReviewer`. Kein neuer State/Reducer nötig —
folgt demselben Muster wie `deleteDraft`.

## Risks / Trade-offs

- [Freigeber löscht versehentlich einen eingereichten Bericht, der eigentlich nur
  überarbeitet werden sollte] → Bestätigungsdialog mit eindeutigem Text
  („Eingereichten Bericht endgültig löschen? Der Autor muss ggf. neu beginnen.");
  gleiche Mitigation wie beim bestehenden Draft-Löschen.
- [Zwei Freigeber agieren gleichzeitig: einer publisht, der andere löscht] → beide
  Operationen laufen über einfache `UPDATE`/`DELETE ... WHERE id=? AND state=?`-Muster
  bzw. das bestehende `SELECT`-dann-`DELETE` mit State-Check; ein Race führt im
  schlimmsten Fall zu einem 409 beim zweiten Request (Bericht existiert nicht mehr
  oder ist nicht mehr im erwarteten State) — kein Datenverlust über das gewollte
  Löschen hinaus, kein neues Verhalten gegenüber dem bestehenden Draft-Delete.

## Migration Plan

Keine Migration nötig (kein Schema-Änderung, State-Wert `pending_review` existiert
bereits im CHECK-Constraint). Reiner Code-Deploy.
