# Design — member-csv-import-status-teamwerk

## CSV-Mapping (Quelle → Senke)

```
CSV-Header                          DB-Spalte / Verhalten
────────────────────────────────────────────────────────────────────
Status                              IGNORIERT (kein Read, kein Write)
Status TeamWERK                     members.status via normalizeStatus
                                    aktiv|passiv|ausgetreten|anwaerter|
                                    honorar|verletzt|pausiert bleiben.
                                    Alias gekündigt → ausgetreten bleibt.
beitragsfrei                        members.beitragsfrei (1 bei "ja",
                                    sonst 0 — direkt aus dieser Spalte)
Grund für Beitragsfreiheit          members.beitragsfrei_grund (TEXT NULL)
```

## Entscheidungen

### D1 — Alte „Status"-Spalte komplett ignorieren

Heute liest `Import` die Spalte „Status" sowohl für Insert als auch für Update (`handler.go:1814`, `handler.go:1919`, `handler.go:1953`). Wir entfernen jeden Lesezugriff. Konsequenzen:

- Alte CSV-Exports, die ausschließlich „Status" enthalten (kein „Status TeamWERK"), führen beim Anlegen neuer Mitglieder zum Default `aktiv` und beim Update zu **keiner** Status-Änderung. Das ist akzeptiert, weil das aktuelle Mapping ohnehin falsch wäre (Freitext fällt in den `aktiv`-Default).
- Die Spalte „Status" darf weiterhin in der CSV stehen — sie wird nur nicht ausgewertet.

### D2 — Statuswert „gekündigt" bleibt Alias auf `ausgetreten`

Die CSV unterscheidet `gekündigt` und `ausgetreten` semantisch nicht — beide bedeuten „nicht mehr Mitglied". Wir führen keinen neuen Status ein, weil:

- Der `CHECK`-Constraint auf `members.status` (Migration 001:528) wäre sonst zu ändern.
- UI und Filterlogik (z. B. `MembersPage`, `BeitragslaufPage`) müssten den neuen Status zusätzlich kennen.
- Es gibt aktuell keine fachliche Auswertung, die zwischen „gekündigt" und „ausgetreten" trennt.

`normalizeStatus` mappt `gekündigt` und `Vereinswechsel` weiter auf `ausgetreten` (bleibt unverändert).

### D3 — `beitragsfrei` direkt aus eigener CSV-Spalte, Ableitung entfällt

Der heutige Block in `handler.go:1953–1960` leitet `beitragsfrei` aus `Status == "beitragsfrei"` ab. Mit dem neuen CSV-Schema gibt es eine eigene Spalte; die Ableitung fällt ersatzlos weg. Folge fürs Frontend:

- `IMPORT_FIELDS` heute: `{ col: 'status', label: 'Status / Beitragsfrei' }` (kombiniert).
- Neu: drei separate Einträge `status`, `beitragsfrei`, `beitragsfrei_grund` mit eigenständigen Whitelist-Checkboxen.

### D4 — `beitragsfrei_grund` ist gekoppelt an `beitragsfrei`

Wenn `beitragsfrei=false`, ergibt ein Grund keinen Sinn. Die Kopplung erzwingen wir **applikationsseitig** in `UpdateBankdaten` und `PUT /api/members/{id}`:

```
falls beitragsfrei == 0:
    UPDATE members SET beitragsfrei_grund = NULL ...
```

Bewusst **kein** SQL-`CHECK`-Constraint: ein CHECK würde ein eingeschleustes `beitragsfrei_grund` bei `beitragsfrei=0` mit DB-Fehler ablehnen, was die HTTP-Schicht in 500er-Pfade zwingt. Die Applikationsregel ist nachsichtiger („akzeptiere, aber clear vor Schreiben"), liefert weiterhin 204 und ist testbar (siehe `TestUpdateMember_BeitragsfreiFalseClearsGrund`, `TestBankdaten_BeitragsfreiFalseClearsGrund`).

Im UI wird das Eingabefeld bei Checkbox-Off **ausgeblendet** und der Form-Wert lokal geleert; das Backend ist die letzte Verteidigungslinie, falls jemand das Frontend umgeht.

### D5 — Kassierer-Whitelist erweitern: `beitragsfrei` + `beitragsfrei_grund` gemeinsam

Die heutige Spec `kassierer-member-zugriff` schreibt fest: `PUT /api/members/{id}/bank-details` ändert `beitragsfrei` **nicht** (Scenario „Kassierer ändert nur Bankfelder"). Wir lockern das bewusst:

- Die Kopplung `beitragsfrei ↔ beitragsfrei_grund` lässt sich nicht aufteilen — wer den Grund pflegt, muss auch das Flag setzen können.
- Der Bankdaten-Block der UI enthält bereits heute die Checkbox „Beitragsfrei" — Kassierer sehen sie, konnten sie aber bisher faktisch nicht persistent ändern (nur Vorstand/Admin). Mit dieser Änderung wird das UI-Verhalten zur Wahrheit.

Die kombinierte Whitelist nach diesem Change:
`iban, sepa_mandat, sepa_mandat_date, account_holder, street, zip, city, beitragsfrei, beitragsfrei_grund`.

Andere Stammdaten (Name, Status, Rollen, Geburtsdatum, …) bleiben kassierer-unzugänglich.

### D6 — Enrich-Modus für `beitragsfrei_grund`

Der Enrich-Modus überschreibt nie belegte Felder. Für `beitragsfrei_grund` heißt das:

- DB-Wert `NULL` oder leer → CSV-Wert wird übernommen.
- DB-Wert nicht leer → CSV-Wert wird ignoriert (unabhängig davon, ob er identisch oder verschieden ist).

Für `beitragsfrei` (Bool, `NOT NULL DEFAULT 0`) gibt es kein „leer". Damit Enrich kein Rückwärtsschritt wird, bleibt die bestehende Regel: Enrich überschreibt `beitragsfrei` nur, wenn `dbBeitragsfrei == 0` und CSV `"ja"` — also nur das Hochsetzen von `false` auf `true`, nicht das Wegsetzen.

### D7 — Migrationsnummer 007

Höchste Migration ist heute `006_template_id_backfill`. Die Migration trägt die Nummer `007_beitragsfrei_grund` (`.up.sql` + `.down.sql`). Reines `ALTER TABLE … ADD COLUMN` — SQLite-WAL-sicher, kein Backfill nötig (Default `NULL`).

## Nicht-Ziele

- Keine Änderung am CHECK-Constraint von `members.status`.
- Kein neuer Status-Wert „gekündigt".
- Keine Änderung am Match-Algorithmus (Vorname+Nachname+DOB) und am Duplikat-Detektor.
- Keine Änderung an SEPA-Pflichtfeldern oder IBAN-Validierung.
- Keine Änderung an `member_change_drafts` (Bankdaten-Draft-Pfad) — der Grund läuft denselben Weg wie die übrigen Bankfelder.
