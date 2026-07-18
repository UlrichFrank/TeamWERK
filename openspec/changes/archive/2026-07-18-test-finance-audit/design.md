## Context

Welle 2. Finance-Package `internal/beitragslauf`. Routen (`router.go`, Tier
`RequireClubFunction("vorstand","kassierer")`): `POST /fee-run/confirm`, `GET /fee-run/protocol`,
`POST /fee-run/export-data`, `GET /fee-run/preview`. Zero-Knowledge: der Server sieht keine
Klartext-IBAN; das Beitragslauf-Protokoll ist eine append-only Textdatei pro Saison unter
`BEITRAGSLAUF_DIR` (kein DB-Tabelle). Scope wurde durch zwei Detail-Recherchen gegen den echten
Code verifiziert; Autorisierung ist bereits über `internal/permissions/matrix_test.go` geprüft.

## Goals / Non-Goals

**Goals:**

- Das fachliche Verhalten von `confirm`/`protocol` festnageln — insbesondere die Sicherheits-
  Invariante „**keine IBAN/Klartext-Bankdaten im Protokoll**" und Append-Only.
- Die ungetesteten member-400-Pfade von `export-data` und die fehlende Halbierungs-Zelle schließen.
- Keine bestehenden Tests duplizieren (Vereins-SEPA-400, Fälligkeits-400, die drei Halbierungs-
  Bedingungen und die `aktiv_ohne`-Austritts-Variante sind bereits abgedeckt).

**Non-Goals:**

- Keine Authz-403-Tests pro Route (Persona-Matrix deckt das ab).
- Keine Geschäftslogik-Änderung; kein Refactor.
- `auth`-Fehlerpfade (Roadmap 5.2) sind bewusst NICHT in diesem Change (separat).

## Decisions

**D1 — Body-Substring-Assertions zur Branch-Isolierung.** `export-data` hat mehrere 400-Quellen
(ungültiger Body, fehlende Vereins-SEPA, Fälligkeit, ausgeschlossenes/unbekanntes Mitglied) mit
je eigener Meldung. Jeder 400-Test prüft zusätzlich den Meldungs-Substring
(`"ausgeschlossen oder unbekannt"` vs `"ungültiger Body"` …), damit er wirklich den Ziel-Branch
trifft und nicht zufällig aus einem anderen Grund 400 liefert (vgl. Welle-1-False-Green-Lehre).

**D2 — „ohne Mandat" vs „ohne Bankdaten" sauber trennen.** `insertMember` legt den
`member_sensitive`-Envelope nur an, wenn `iban != ""`. „ohne Bankdaten" = `iban=""`;
„ohne Mandat" = `sepaMandat=0` bei gesetzter IBAN. Nicht mischen, sonst ist die Ausschluss-
Ursache mehrdeutig.

**D3 — Halbierungs-Restfall isolieren.** Für `aktiv_mit` + Austritt muss `join_date`
**vor** dem Saisonfenster liegen (sonst gewinnt Priorität `eintritt` über `austritt`) und
`is_inaugural=0` bleiben (sonst `erstjahr`). `exit_date` inklusive im Fenster. Assertion prüft
`kategorie=aktiv_mit`, `half_reason=austritt`, `betrag_cent=4800` (halber aktiv_mit-Satz 9600) —
der Betrag unterscheidet die Zelle von der bereits getesteten `aktiv_ohne`-Variante (11300).

**D4 — Protokoll-Datei direkt gegenlesen.** `setupSrv` liefert das Temp-`dir`; Confirm-Tests
lesen die Datei (`beitragslauf_<label mit / → ->.txt`) direkt und prüfen Format + Abwesenheit der
IBAN. `protocol`-Tests gehen zusätzlich über den HTTP-Weg (Content-Type, Body).

## Risks / Trade-offs

- **Kein echter Bug erwartet** in diesem Bereich (anders als Welle 1 `checkAntiEscalation`) — die
  Tests sichern Bestandsverhalten. Sollte ein Test wider Erwarten einen Fail-Open zeigen (z.B.
  IBAN doch im Protokoll), greift `test-strategy` D3: Fix zuerst, dann Test.
- **`testutil.Post` kann kein syntaktisch kaputtes JSON senden** (encodet selbst) → für den
  „ungültiger Body"-400 ein typ-fehlpassendes Feld (`saison_id:"abc"`) oder ein manuell gebauter
  Request mit `strings.NewReader`.
