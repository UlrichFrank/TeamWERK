## Context

Der Basis-Change `mannschaft-aufgaben-strafen` hat drei kader-scoped Tables etabliert (`penalty_types`, `team_penalties`, `kader_strafenwarte`) plus einen Read-Gate-Helper (`canReadPenalties`: Spieler ∨ Trainer ∨ Erw. Kader, ohne Eltern) in `internal/teams/access.go`. Der Strafen-Tab auf `MeinTeamPage.tsx` zeigt an oberster Stelle die Zeile „Mannschaftskasse: X €" als **reine Summe aller offenen Strafen** (Zeilen 484–487) — das ist keine Kasse, sondern der Nenner der Strafen. Der Tab hat weder ein Konzept von Einzahlungen/Ausgaben noch von unterschiedlichen Einheiten.

Handball-Teams organisieren Strafen in der Praxis in **Euro** oder in **Strichen** — die aktuelle Implementierung kennt nur Cent. Und die reale Mannschaftskasse (Geld auf Konto/Umschlag, Ausgaben für Trainergeschenke, Startgelder, Kabinen-Bier) fehlt komplett.

Dieser Change ergänzt beides: Einheiten für Strafen und ein echtes, entkoppeltes Kassenbuch mit eigener Rolle `Kassenwart`.

## Goals / Non-Goals

**Goals:**
- Strafen wählbar in Euro **oder** Striche pro Kader; Wechsel rechnet um.
- Eigenes Kassenbuch pro Kader (Ledger, Saldo), Rolle `Kassenwart` als per-Kader-Appointment.
- Strafen und Kasse strukturell entkoppeln — kein „Kassenstand"-Wert im Strafen-Tab, keine Buchung, die Strafen mit-modifiziert.
- Eigene Zeile in Listen fett darstellen (kleine UX-Verbesserung).
- Konsistenz mit den Hard Rules (Broadcast, Tests, brand-Tokens, lucide) und dem etablierten kader-scoped Muster aus dem Basis-Change.

**Non-Goals:**
- Kein Audit-Trail (Buchungen werden hart gelöscht wie Strafen).
- Kein konfigurierbarer Wechselkurs — `1 € = 1 Strich` ist fest.
- Keine automatische Kopplung „Buchung schließt Strafe" (bewusst manuell, siehe D3).
- Kein neuer globaler `member_club_functions`-Wert.
- Keine Push/Benachrichtigungen.

## Decisions

### D1 — Einheit pro Kader/Saison, nicht pro Strafe oder pro Katalog-Eintrag
Neue Table `penalty_settings(kader_id PK, unit CHECK IN ('euro','striche') DEFAULT 'euro')`. Der Trainer setzt einmal pro Kader.

**Warum:** Nur so bleiben Summen und Kassenstands-Aggregation eindeutig. „5 € + 2 Striche" ist keine sinnvolle Zahl — die App würde entweder mischen (Nonsense) oder pro Einheit getrennt anzeigen (Komplexität ohne Realgewinn). In der Praxis entscheidet ein Team einmal pro Saison; ein Wechsel mitten in der Saison ist selten, aber möglich (siehe D2).

**Alternative verworfen:** Einheit pro `penalty_types`-Eintrag. Öffnet Mixed-Katalog, macht Kassenstand-Anzeige undefiniert (siehe Explore-Diskussion).

### D2 — Einheiten-Wechsel rechnet um, sperrt nicht
`PUT /api/teams/{id}/penalty-settings` ist ein Massenumrechnungs-Endpoint. In einer TX:
1. `UPDATE penalty_types SET default_amount_cent = ...` — bei →Striche `ceil(default_amount_cent / 100)`, bei →Euro `default_amount_cent * 100`.
2. `UPDATE team_penalties SET amount_cent = ...` — dieselbe Regel für alle Rows dieses Kaders.
3. `UPDATE penalty_settings SET unit = ...`.
4. `Broadcast("penalty-settings")` + `Broadcast("penalties")`.

Rate ist fest: `1 € = 1 Strich`. Euro → Striche rundet **auf** (`ceil(cent / 100)`), damit niemand versehentlich billiger wegkommt; Striche → Euro ist verlustfrei (`n * 100`).

**Preview-Endpoint** `GET /api/teams/{id}/penalty-settings/preview?to=<unit>` (Trainer-only) liefert Delta-Liste + Anzahl aufgerundet, ohne DB-Mutation — das Frontend zeigt vor dem PUT eine Bestätigung mit den betroffenen Rows.

**Warum:** Sperren (`409` bei bestehenden Strafen) fühlt sich benutzerfeindlich an, weil der Wechsel real vorkommt. Umrechnen mit sichtbarer Vorschau ist ehrlich und praktikabel. Aufrunden verhindert stille „Rabatte" durch Rundungsverluste.

**Alternative verworfen:** `409` mit Manual-Reset-Zwang; einstufiges `PUT?dry-run=1` (unüblich in dieser Codebase, `PUT` als Dry-Run ist semantisch schräg).

### D3 — Kasse ist entkoppeltes Ledger, keine Buchung modifiziert Strafen
`team_cashbook_entries(id, kader_id, amount_cent [signed], note, entered_by_member_id, entered_at)`. Positiv = Einnahme, negativ = Ausgabe. Saldo = SQL-`SUM(amount_cent)`.

Eine Buchung modifiziert **nichts** an `team_penalties`. Wenn ein Spieler seine Strafen bezahlt, macht der Kassenwart eine Einzahlungs-Buchung (positiv), und der Strafenwart resettet die offenen Strafen dieses Spielers — zwei separate Aktionen, zwei separate Rollen.

**Warum:** Genau das ist die Entkopplung, die der User wollte. Automatik hier („Buchung mit `penaltyId` schließt Strafe automatisch") würde die entkoppelten Modelle über eine Fremdschlüssel-Hintertür wieder verlöten und das Non-Goal des Basis-Changes aufweichen (keine Zahlungs-Historie). Wenn sich das je als schmerzhaft erweist, ist ein separater Follow-up-Change der richtige Ort.

**Alternative verworfen:** V3 aus der Explore-Skizze (Buchung mit optionalem `penalty_id`, das die Strafe schließt).

### D4 — Kassenwart als per-Kader-Appointment (Sibling zu Strafenwart)
`kader_kassenwarte(kader_id, member_id) PK`, exakt gleiches Muster wie `kader_strafenwarte`. Ernennung durch Trainer.

**Warum:** Konsistenz mit dem im Basis-Change bewusst gewählten Modell. Kein neuer globaler `member_club_functions`-Wert (keine CHECK-Migration, kein neuer JWT-Claim), kein „Kassenwart überall/immer". Fremd-Team-Buchung strukturell ausgeschlossen (D5 baut darauf auf).

### D5 — Write-Gate „Trainer ODER Kassenwart"
Buchen (`POST`/`DELETE` auf `/cashbook`) ist erlaubt, wenn der Caller-Member Trainer **oder** Kassenwart **dieses** Kaders ist. Ernennung (`POST/DELETE /treasurers`) nur Trainer. Beide Gates sind DB-Lookups im Handler (`isTrainerOfTeam`, `isKassenwartOfTeam`), analog zu `isStrafenwartOfTeam` aus dem Basis-Change.

**Warum:** Trainer soll die Kasse auch pflegen können (typischer Fall: Trainer hält die Kasse selbst, kein separater Kassenwart im Team). Reine „Kassenwart-only"-Gates würden den Trainer aussperren und Ernennung erzwingen.

### D6 — Read-Gate für Kasse = Read-Gate für Strafen
`canReadCashbook = canReadPenalties` (Spieler ∨ Trainer ∨ Erw. Kader; **keine Eltern**, keine Außenstehenden). Implementiert als benannter Alias / Wiederverwendung des Helpers in `internal/teams/access.go`.

**Warum:** Der User hat explizit gewählt „wie Strafen, keine Eltern". Konsistenz vermeidet asymmetrische Sichtbarkeiten, die Eltern verwirren („warum sehe ich die Kasse, aber nicht die Strafen?").

### D7 — Kassenbuch-Einträge werden hart gelöscht (kein Status)
Analog zu Strafen: `DELETE /api/teams/{id}/cashbook/{eid}` löscht die Row komplett. Kein „storniert"-Status.

**Warum:** Konsistenz mit dem Basis-Change; „Buchungs-Historie" ist explizit Non-Goal. Wenn Trainer/Kassenwart sich vertippt, korrigiert er per Löschen + Neu.

### D8 — Bold-Me als reine Frontend-Konvention
Vergleich `roster.mymember.id === row.memberId` (bzw. `user.id === row.userId` je nach Kontext) → `font-semibold`. Kein API-Feld, keine Backend-Änderung.

**Warum:** Rein visuell, keine Sicherheits- oder Datenimplikation. Wird in `MeinTeamPage.tsx` in drei Listen angewendet (Roster-Tabs, Strafen-Übersicht, Kassenbuch); als kleiner Utility-Vergleich, keine neue Abstraktion.

### D9 — SSE-Events getrennt pro Aggregat
Broadcasts: `cashbook` (Ledger-Mutation), `treasurers` (Ernennung), `penalty-settings` (Einheiten-Wechsel), und beim Einheiten-Wechsel zusätzlich `penalties` (weil Beträge in Rows mutieren). Frontend abonniert via `useLiveUpdates` und reloadet entsprechend.

**Warum:** Getrennte Events, damit `MeinTeamPage` nur die relevante Sektion nachlädt (Kasse-Buchung soll keinen Roster-Reload triggern). Erfüllt die Broadcast-Hard-Rule und wird vom `broadcast_test.go`-Gate erfasst.

## Risks / Trade-offs

- **Sichtbarkeitsleck bei Kasse** (Eltern sehen den Saldo) → schwerste Fehlerklasse. Mitigation: eigener Endpoint mit `canReadCashbook` (D6), explizite Negativ-Tests (Eltern → 403, Außenstehender → 403), Read-Gate als benannter, getesteter Helper.
- **Massen-Umrechnung schlägt fehl auf halber Strecke** → korrupte Beträge. Mitigation: TX (D2), Test mit Multi-Row-Setup.
- **Fremd-Team-Buchung durch globalen Kassenwart** → durch D4/D5 strukturell ausgeschlossen; Test `TestCashbookCreate_ForeignTeamKassenwart_403`.
- **Wechselkurs ungerecht wahrgenommen** („1 € = 1 Strich ist willkürlich") → real, aber jede andere Rate wäre genauso willkürlich; explizite Preview + Aufrunden macht es transparent. Alternative Raten ausdrücklich Non-Goal.
- **Buchung ohne Strafen-Kopplung fühlt sich für Nutzer inkonsistent an** („warum ändert die Buchung meine Strafenliste nicht?") → Dokumentations-Risiko im UI (Hilfetext im Kasse-Tab). Bewusst akzeptiert (D3).
- **RAM/VPS** → keine neuen Dependencies, drei zusätzliche SQLite-Tables + drei Handler-Dateien. Kein Footprint-Risiko.

## Migration Plan

1. Neue Migration `0NN_mannschaftskasse_und_strafen_einheiten.up.sql` / `.down.sql` (nächste freie Nummer nach `031`): drei Tables — `penalty_settings`, `team_cashbook_entries`, `kader_kassenwarte`. Alle FK auf `kader(id)` bzw. `members(id)` `ON DELETE CASCADE`. Beträge als `INTEGER` (signed für Kassenbuch, unsigned für Strafen).
2. Migration setzt für **alle bestehenden Kader** eine Default-Row in `penalty_settings` mit `unit='euro'` (Backfill, damit `GET` nicht auf `NULL` läuft).
3. Backend-Handler + Gates + Routen + Tests.
4. Frontend (MeinTeamPage) — neuer Tab, Header-Zeile raus, Verwaltungs-Sektion, `useLiveUpdates`, Bold-Me.
5. Rollback: `.down.sql` droppt die drei Tables (rein additiv, kein Datenverlust am Bestand des Basis-Changes).

## Open Questions

- UI-Feinschliff Kassenbuch: Einzahlung/Ausgabe als getrennte Buttons oder als Vorzeichen-Toggle? — Umsetzungsdetail, keine Spec-Auswirkung.
- Anzeige der Einheit in der Kasse: die Kasse ist immer Euro, deshalb keine Einheit-Umschaltung dort. Kein offener Punkt, nur Dokumentation.
- Ob der Einheiten-Wechsel eine SSE-Massenlast triggert (`penalties`-Event → alle Clients reloaden Strafen) — bei Kader-Größe <30 kein Problem. Wenn sich das je als teuer erweist, Sonder-Event `penalties-bulk-updated`. Kein Blocker.
