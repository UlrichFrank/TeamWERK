## Why

Ein Heimspieltag wird von genau einem Verein **ausgerichtet** — der stellt Kasse, Bewirtung und Hallenaufsicht. Welche Dienste an einem Spieltag überhaupt anfallen, hängt davon ab, wer ausrichtet: richtet ein Partnerverein aus, entfällt für uns der Kuchendienst; richten wir aus, entfällt umgekehrt nichts. Heute kennt das System diesen Begriff nicht. Der Auto-Regen (`internal/games/regen.go`) erzeugt die Slots einer Heim-Vorlage unterschiedslos für jeden Heimspieltag, und der Vorstand räumt die nicht zutreffenden Dienste hinterher von Hand weg — bei jedem Spielplanlauf erneut, weil der nächste Regen sie wieder anlegt.

Das lässt sich mit den vorhandenen Mitteln nicht abbilden: `team_ids` und `rotation_enabled` auf `game_template_items` filtern über **Mannschaften**, nicht über den ausrichtenden Verein, und `games.template_id` schaltet nur ganze Vorlagen um — für jede Ausrichter-Kombination eine eigene Heim-Vorlage zu pflegen würde alle gemeinsamen Dienstzeilen vervielfachen.

## What Changes

- **Neue vereinsweite Ausrichter-Liste.** Editierbare Liste von Vereinen (Name, aktiv, Sortierung) mit **genau einem als Default markierten** Eintrag. Gepflegt unter `/einstellungen` in der neuen Kachel **Ausrichter**.
- **Der Tab „Bewirtung" heißt künftig „Heimspieltage"** und trägt zwei Kacheln: **Bewirtung** (die bestehenden Felder „Kuchen je Spiel" und „Max. Kuchen pro Mannschaft") und **Ausrichter** (die neue Liste). Beide steuern die automatische Dienst-Generierung bei Heimspielen — der Tab bündelt sie unter dem gemeinsamen fachlichen Dach.
- **Der Ausrichter ist eine Eigenschaft des Spieltags**, nicht des einzelnen Spiels und nicht der Halle: ein Wert pro `(date, season_id)`. Er gilt für alle Termine dieses Tages — bereits bestehende wie später angelegte.
- **Auflösung ist nie leer:** `ausrichter(tag) = spieltag_ausrichter[tag] ?? default_ausrichter`. Dadurch braucht weder der H4A-Import noch die Massenanlage noch die Bestandsmigration eine Sonderbehandlung — es gibt keinen Zustand „Tag ohne Ausrichter".
- **Neues optionales Feld `ausrichter_id` auf Vorlagen-Items** (`game_template_items`), nur auf Vorlagen mit `template_type='heim'` zulässig. `NULL` (Default) = die Zeile gilt immer; gesetzt = die Zeile erzeugt nur an Spieltagen Slots, deren Ausrichter übereinstimmt. Bestehende Vorlagen tragen `NULL` und verhalten sich unverändert.
- **Gate im Auto-Regen an zwei Stellen:** in `regenGameItems` (erzeugt die Slots) **und** in `buildRotationPlan` (berechnet den Kuchenbedarf des Tages). Nur so rechnet die Rotation nicht Bedarf für Dienste, die das Gate danach verwirft.
- **Drei Schreibpfade für den Tages-Ausrichter**, alle mit derselben Vorschau: Tagesansicht im Kalender, Termin-Wizard (als erkennbar tagesbezogenes Feld) und die Massen-Dienstregeneration (je Tag wählbar).
- **Vorschau vor jedem ausrichter-bedingten Schreiben.** Eine Änderung kann Dienste mit bestehenden Zusagen löschen. Vor dem Commit zeigt ein Dry-Run die Bilanz („löscht N Dienste, M Zusagen") — derselbe Codepfad wie beim Massenlauf (Rollback statt Commit), kein Client-Nachbau.
- **Löschen eines Ausrichters ist erlaubt.** Betroffene Spieltage fallen auf den Default zurück. Vorlagen-Zeilen mit diesem Ausrichter werden im selben bestätigten Schritt **mitgelöscht** statt entkoppelt — ein `SET NULL` würde sie still auf „gilt immer" heben und damit mehr Dienste erzeugen als vorher.

**Nicht Teil dieses Changes:** keine Heuristik und kein zusätzlicher Schritt im H4A-Import (der Import nutzt schlicht den geltenden Tages-Ausrichter bzw. den Default); keine Zurechnung von Diensten an Mitglieder des Ausrichters (der Ausrichter filtert nur, er weist nicht zu); kein Ausrichter je Halle; keine Vorschau beim Ändern des Default-Ausrichters in den Einstellungen.

## Capabilities

### New Capabilities
- `heimspieltag-ausrichter`: Vereinsweite Ausrichter-Liste mit Default, Ausrichter je Spieltag inklusive Auflösungsregel, Ausrichter-Bindung auf Vorlagen-Items und das daraus folgende Gate im Auto-Regen — samt Vorschau vor ausrichter-bedingten Löschungen.

### Modified Capabilities
- `bewirtungsrotation`: `buildRotationPlan` berücksichtigt beim Bedarf nur noch Heimspiele, deren rotations-aktives Item das Ausrichter-Gate des Tages passiert. Der Einstellungen-Tab wird von „Bewirtung" zu „Heimspieltage" umbenannt und in zwei Kacheln gegliedert; die beiden bestehenden Felder bleiben inhaltlich unverändert.
- `duty-bulk-regen`: Der Massenlauf zeigt und setzt je Tag im Zeitraum den Ausrichter; Preview und Apply weisen die Wirkung in der bestehenden Bilanz aus.

## Impact

- **DB:** neue Migration `047_*` — Tabellen `ausrichter` (mit Partial-Unique-Index auf `is_default`) und `spieltag_ausrichter` (`PRIMARY KEY (date, season_id)`), Spalte `game_template_items.ausrichter_id` (nullable, `ON DELETE RESTRICT`), Seed-Zeile für den Default-Ausrichter.
- **Backend:** `internal/settings/ausrichter.go` (Foundation-Layer: Liste, CRUD und `ResolveAusrichterForDay`, damit `internal/games` es innerhalb seiner `tx` lesen darf — gleiches Muster wie `GetBewirtungVerhaeltnis`), `internal/settings/handler.go` (Routen der Liste), neu `internal/games/ausrichter_handler.go` (Preview/Apply für den Tages-Ausrichter — muss in `games` liegen, weil es den unexportierten `runAutoRegen` braucht, wie schon beim H4A-Import), `internal/games/regen.go` (Gate in `regenGameItems` + `buildRotationPlan`, Auflösung einmal je `regenSingleDay`), `internal/games/handler.go` (Vorlagen-Item-CRUD um `ausrichter_id`), `internal/games/bulkregen_handler.go` (Tages-Ausrichter in Request/Response), `internal/app/router.go`.
- **Frontend:** `web/src/pages/AdminSettingsPage.tsx` (Tab-Umbenennung + zwei Kacheln + Ausrichter-CRUD nach dem Muster von `StammvereineTab`), `web/src/components/DutyTemplateItemFields.tsx` und beide Vorlagen-Editoren (Ausrichter-Auswahl, nur bei `template_type='heim'`), `web/src/pages/KalenderPage.tsx` (Tages-Picker, Wizard-Feld, Vorschau-Modal, Ausrichter-Spalte im Massenlauf-Dialog).
- **Bestand:** Kein Verhaltenswechsel beim Deploy — heute trägt kein Vorlagen-Item eine Ausrichter-Bindung, das Gate ist also zunächst für alle Zeilen offen. Erst das Setzen eines `ausrichter_id` auf einer Vorlagen-Zeile ändert etwas, und das läuft über die Vorschau.
- **Abhängigkeit:** setzt den Change `bewirtung-cap-global` voraus (Item-Feld heißt dort bereits `rotation_enabled`, der Tab trägt zwei Felder). Migrationsnummer vor dem Schreiben erneut prüfen.
- **RAM/VPS:** unkritisch — zwei kleine Tabellen, ein zusätzlicher `SELECT` je Regen-Tag.

## Test-Anforderungen

| Route | Testname | Erwartung |
|---|---|---|
| `GET /api/ausrichter` | `TestListAusrichter_Happy` | `200`, Liste inkl. `is_default`-Markierung |
| `GET /api/ausrichter` | `TestListAusrichter_Unauthenticated` | `401` |
| `POST /api/ausrichter` | `TestCreateAusrichter_Happy` | `201`, `settings-changed`-Broadcast |
| `POST /api/ausrichter` | `TestCreateAusrichter_DuplicateName` | `409`, nichts geschrieben |
| `POST /api/ausrichter` | `TestCreateAusrichter_NichtVorstand` | `403` |
| `PUT /api/ausrichter/{id}` | `TestSetDefaultAusrichter_VorherigerVerliertDefault` | `200`, genau eine Zeile mit `is_default=1` |
| `DELETE /api/ausrichter/{id}` | `TestDeleteAusrichter_SpieltageFallenAufDefault` | `200`, betroffene `spieltag_ausrichter` auf `NULL`, Auflösung liefert Default |
| `DELETE /api/ausrichter/{id}` | `TestDeleteAusrichter_VorlagenZeilenWerdenMitgeloescht` | `200`, Items mit dieser Bindung sind weg (kein `SET NULL`) |
| `DELETE /api/ausrichter/{id}` | `TestDeleteAusrichter_DefaultNichtLoeschbar` | `409 default_ausrichter_undeletable`, nichts geschrieben |
| `GET /api/game-days/{date}/host` | `TestGetGameDayHost_FaelltAufDefault` | `200`, `is_explicit=false`, Default-ID |
| `POST /api/game-days/host/preview` | `TestPreviewGameDayHost_ZeigtBilanzOhneZuSchreiben` | `200` mit Bilanz, DB unverändert |
| `POST /api/game-days/host/apply` | `TestApplyGameDayHost_Happy` | `200`, `spieltag_ausrichter` gesetzt, Regen gelaufen, Broadcast |
| `POST /api/game-days/host/apply` | `TestApplyGameDayHost_UnbekannterAusrichter` | `400`, nichts geschrieben |
| `POST /api/game-days/host/apply` | `TestApplyGameDayHost_NichtBerechtigt` | `403` |
| `POST /api/game-templates` (Item) | `TestTemplateItem_AusrichterNurBeiHeimVorlage` | `400 ausrichter_requires_heim_template` |
| `POST /api/duty-slots/bulk-regen/preview` | `TestBulkRegenPreview_HostOverrideWirktUndSchreibtNicht` | `200`, Tageszeile weist neuen Ausrichter aus, DB unverändert |

**Garantierte Invarianten (Unit-Tests in `internal/games/regen_test.go` bzw. `internal/settings/ausrichter_test.go`):**

- **Auflösung ist total:** `ResolveAusrichterForDay` liefert für jeden Tag einen Ausrichter — expliziter Wert, sonst Default. Ein `NULL`-Eintrag in `spieltag_ausrichter` verhält sich wie „kein Eintrag".
- **Genau ein Default:** Über alle Schreibpfade bleibt die Anzahl Zeilen mit `is_default=1` exakt 1 (mechanisch durch Partial-Unique-Index abgesichert).
- **Gate ist additiv:** Ein Vorlagen-Item mit `ausrichter_id IS NULL` erzeugt an jedem Spieltag dieselben Slots wie vor diesem Change (Charakterisierung gegen die Bestandslogik).
- **Gate greift vor der Bedarfsrechnung:** Ist das rotations-aktive Item am Tag ausgegated, ist der Kuchenbedarf `0` — nicht „Bedarf berechnet, Slots verworfen".
- **Gate wirkt nur bei `event_type='heim'`:** Auswärts- und generische Termine bleiben unberührt.
- **Preview schreibt nicht:** Nach `preview` sind `duty_slots`, `duty_assignments` und `spieltag_ausrichter` bitgleich zum Zustand davor.
- **Zusagen-Erhalt:** Ändert sich der Ausrichter so, dass eine Vorlagen-Zeile weiterhin gilt, überlebt die Zusage den Regen unverändert (bestehendes `restoreAssignments`-Matching bleibt gültig).
