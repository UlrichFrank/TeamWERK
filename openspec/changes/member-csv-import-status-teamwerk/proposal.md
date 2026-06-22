## Why

Die exportierte Mitgliederliste verwendet zwei getrennte Spalten: **„Status"** als Freitext-Begründung (z. B. `Zweitspielrecht, beitragsfrei`, `kein aktiver Sportler mehr`) und **„Status TeamWERK"** als kontrollierter Lebenszyklus-Status (`aktiv|passiv|gekündigt|ausgetreten`). Heute liest der Importer ausschließlich die Spalte „Status" und mappt sie über `normalizeStatus` auf `members.status`. Für die neue CSV bedeutet das: Freitext wie „Zweitspielrecht, beitragsfrei" wird durch den Default-Fallback der Funktion auf `aktiv` gemappt — auch wenn das Mitglied gekündigt ist. Außerdem wird `members.beitragsfrei` aus dem String `Status == "beitragsfrei"` abgeleitet, obwohl die CSV inzwischen eine eigene Spalte `beitragsfrei` führt. Die Spalte **„Grund für Beitragsfreiheit"** wird gar nicht importiert; im Member-Profil fehlt ein passendes Feld.

## What Changes

- **CSV-Spalten-Mapping** im Importer (`POST /api/members/import`):
  - Spalte **„Status"** wird **nicht mehr gelesen** (ersatzlos ignoriert — auch beim Anlegen neuer Mitglieder).
  - Spalte **„Status TeamWERK"** wird über `normalizeStatus` auf `members.status` gemappt. `gekündigt → ausgetreten` bleibt als Alias erhalten; kein neuer Status.
  - Spalte **„beitragsfrei"** wird direkt auf `members.beitragsfrei` gemappt (`"ja"` → `1`, sonst `0`). Die alte Ableitung aus dem Status-Freitext entfällt.
  - Spalte **„Grund für Beitragsfreiheit"** wird auf das neue Feld `members.beitragsfrei_grund` gemappt.
- **Neues Member-Feld** `beitragsfrei_grund TEXT NULL` per Migration 007, freigeschaltet in `GET /api/members/{id}` und im `MemberKontaktTab` (Bankdaten-Abschnitt) editierbar.
- **UI-Koppelung**: Das Eingabefeld „Grund" wird **nur angezeigt**, wenn die Checkbox „Beitragsfrei" gesetzt ist. Beim Deselektieren der Checkbox MUSS der Grund-Wert serverseitig auf `NULL` zurückgesetzt werden, damit keine Datenleichen entstehen.
- **Kassierer-Whitelist** (`PUT /api/members/{id}/bank-details`) wird um das gekoppelte Paar `beitragsfrei` + `beitragsfrei_grund` erweitert, damit Kassierer den Beitragsfrei-Status samt Grund pflegen können (Vorstand/Admin via regulärem `PUT /api/members/{id}` weiterhin auch).
- **Frontend-Importdialog**: `IMPORT_FIELDS` in `MembersPage.tsx` wird in `status`, `beitragsfrei` und `beitragsfrei_grund` aufgeteilt (heute kombiniert als „Status / Beitragsfrei"), damit Feld-Whitelist und Anzeige sauber bleiben.

## Capabilities

### Modified Capabilities

- `members-csv-enrich-mode`: Spalten-Mapping wird umgestellt (Status TeamWERK + beitragsfrei + Grund für Beitragsfreiheit); alte Status-Spalte und die abgeleitete `beitragsfrei`-Heuristik entfallen.
- `member-csv-import-selective`: Whitelist-Spalten-Set wird um `beitragsfrei` und `beitragsfrei_grund` erweitert; die kombinierte Status-/Beitragsfrei-Logik wird aufgelöst.
- `members`: neues Feld `beitragsfrei_grund` im DB-Schema, in GET/PUT-Endpoints und im UI-Bankdaten-Block; Kopplung an `beitragsfrei` (NULL bei `false`).
- `kassierer-member-zugriff`: Bankdaten-Whitelist erweitert um `beitragsfrei` und `beitragsfrei_grund`.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `POST /api/members/import` | `TestImport_StatusTeamWERK_AppendNew` | `Status TeamWERK=passiv` legt Mitglied mit `members.status='passiv'` an. Spalte `Status` mit Freitext beeinflusst weder Status noch beitragsfrei. |
| `POST /api/members/import` | `TestImport_BeitragsfreiSpalte_DirectMap` | `beitragsfrei=ja` setzt `members.beitragsfrei=1`. Leere Zelle setzt `0` (Append) bzw. lässt unverändert (Update). |
| `POST /api/members/import` | `TestImport_BeitragsfreiGrund_Append` | `Grund für Beitragsfreiheit` landet wortwörtlich in `members.beitragsfrei_grund`. |
| `POST /api/members/import` | `TestImport_BeitragsfreiGrund_EnrichLeaves` | Enrich-Modus überschreibt einen bereits gefüllten `beitragsfrei_grund` nicht. |
| `POST /api/members/import` | `TestImport_AlteStatusSpalteWirdIgnoriert` | CSV mit ausschließlich „Status"-Spalte (kein „Status TeamWERK") lässt `members.status` für Bestandsmitglieder unverändert und mappt neue Mitglieder auf den Default `aktiv`. |
| `POST /api/members/import` | `TestImport_GekuendigtBleibtAlias` | `Status TeamWERK=gekündigt` setzt `members.status='ausgetreten'`. |
| `PUT /api/members/{id}` | `TestUpdateMember_BeitragsfreiFalseClearsGrund` | Wird `beitragsfrei=false` gespeichert, MUSS `members.beitragsfrei_grund` serverseitig auf `NULL` gesetzt werden — unabhängig vom übertragenen Grund-Wert. |
| `PUT /api/members/{id}/bank-details` | `TestBankdaten_KassiererPflegtBeitragsfreiGrund` | Kassierer darf `beitragsfrei=true` + `beitragsfrei_grund="…"` setzen; übrige Stammdaten (Name, Status, Rollen) unverändert. |
| `PUT /api/members/{id}/bank-details` | `TestBankdaten_BeitragsfreiFalseClearsGrund` | Übertragenes `beitragsfrei=false` setzt Grund auf `NULL`, auch wenn ein Grund mitgesendet wird. |
| `PUT /api/members/{id}/bank-details` | `TestBankdaten_SpielerForbidden` | Nutzer mit ausschließlich `spieler` erhält HTTP 403. |
| `GET /api/members/{id}` | `TestGetMember_BeitragsfreiGrundField` | Response enthält das Feld `beitragsfrei_grund` (auch wenn `NULL` → JSON `null`/leer). |

**Garantierte Invariante**: `members.beitragsfrei = 0` ⇒ `members.beitragsfrei_grund IS NULL`. Diese Kopplung wird in `UpdateBankdaten` und `PUT /api/members/{id}` durchgesetzt; sie ist nicht durch eine DB-CHECK abgesichert (siehe `design.md`).

## Impact

- **Migration:** `internal/db/migrations/007_beitragsfrei_grund.up.sql` (+ `.down.sql`). Nächste freie Nummer.
- **Backend:**
  - `internal/members/handler.go` — `Import` (Spalten-Mapping umstellen), `UpdateBankdaten` (Whitelist erweitern, Grund-Clearing erzwingen), `Update`/`Get` (neues Feld lesen/schreiben + Clearing-Invariante), `ImportRow.Changes` mit neuen Labels.
  - `internal/members/handler_test.go`, `internal/members/import_test.go`, `internal/members/bankdaten_test.go` — neue Tests pro obiger Tabelle.
- **Frontend:**
  - `web/src/pages/MembersPage.tsx` — `IMPORT_FIELDS` aufspalten, neuer Default für alle drei Boxen.
  - `web/src/components/admin/MemberKontaktTab.tsx` — Grund-Textinput unter der Checkbox; bei Toggle aus → Grund leeren; in Submit nicht mitsenden bzw. `null` schicken.
  - `web/src/pages/MemberDetailPage.tsx` — Typ `Member`/`MemberForm` um `beitragsfrei_grund` erweitern.
- **Specs:** vier Capability-Updates (siehe oben).
- **Keine** neuen Routen, **keine** Berechtigungsänderung außerhalb der Bank-Details-Whitelist.
- **CHECK-Constraint** auf `members.status` bleibt unverändert.
