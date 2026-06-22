## Context

`POST /api/members/import` (`internal/members/handler.go`, `Handler.Import`) verarbeitet eine CSV in drei Modi (`append`, `update`, `enrich`) plus einen Dry-Run (`preview`/`preview=1`). Für Bestandsmitglieder sammelt die Schleife pro Zeile `setClauses`/`setArgs`/`changes` über die Helfer `addChange`/`addNullableChange` und vier Inline-Blöcke (Trikotnummer, SEPA-Mandat, abgeleitetes `beitragsfrei`, IBAN mit MOD-97). Am Ende ein dynamisches `UPDATE members SET … WHERE id=?`. Das Frontend (`web/src/pages/MembersPage.tsx`) hat einen dreistufigen Dialog: (1) Datei + Modus, (2) Vorschau-Report, (3) Ergebnis. Vorschau und Anwendung laden dieselbe in-memory `File` getrennt hoch.

## Goals / Non-Goals

**Goals:**
- Gezielte Spalten-Auswahl beim Update von Bestandsmitgliedern (z.B. nur IBAN).
- Pro-Zeile-Auswahl in der Vorschau, welche Bestandsänderungen tatsächlich geschrieben werden.
- Volle Rückwärtskompatibilität: ohne die neuen Formfelder unverändertes Verhalten.

**Non-Goals:**
- Keine Spalten-/Zeilen-Auswahl für das Anlegen NEUER Mitglieder (`created`).
- Keine Persistenz der getroffenen Auswahl über den Request hinaus.
- Keine Änderung an Matching-Strategie, IBAN-Validierung oder Email-Klassifizierung.

## Decisions

**1. Zwei optionale Formfelder statt JSON-Body.** Der Endpoint ist bereits `multipart/form-data` (Datei-Upload). `fields` (komma-separierte DB-Spalten) und `apply_lines` (komma-separierte Zeilennummern) fügen sich nahtlos ein und bleiben optional → keine Breaking Changes.

**2. Whitelist als `fieldAllowed(column)`-Closure.** Leere/fehlende Whitelist → `nil`-Map → alle Felder erlaubt. Jeder Schreibpfad (`addChange`, `addNullableChange`, die vier Inline-Blöcke) ruft `fieldAllowed(<column>)` als erste Bedingung. Das abgeleitete `beitragsfrei` wird an `fieldAllowed("status")` gekoppelt, da es ausschließlich aus dem Status abgeleitet wird.

**3. Identität Vorschau→Apply über CSV-Zeilennummer.** `ImportRow.Line` existiert bereits. Das Frontend kennt aus der Vorschau die Zeilennummern und sendet beim Apply die angehakten als `apply_lines`. Member-IDs wären robuster, sind aber im Report nicht enthalten und existieren nicht für `created`-Zeilen — die Zeilennummer ist die pragmatisch korrekte, bereits vorhandene Kennung. Annahme: dieselbe Datei wird zwischen Vorschau und Apply nicht ausgetauscht (im UI-Flow garantiert, da dieselbe `File`-Referenz).

**4. `skipped` nur außerhalb des Dry-Runs.** Im Dry-Run zeigt die Vorschau bewusst ALLE Zeilen als `updated` (sonst könnte der Nutzer nichts auswählen). Erst beim Apply gilt: `selected := applyLines == nil || applyLines[lineNum]`. Update-Schreibvorgang nur bei `selected`; eine nicht ausgewählte Zeile mit Änderungen wird als `skipped` gemeldet (neuer Zähler + Status), statt als `unchanged` — so bleibt „nichts zu tun" von „bewusst ausgelassen" unterscheidbar.

**5. Frontend: Feld-Auswahl in Schritt 1, Zeilen-Auswahl in Schritt 2.** Feld-Checkboxen erscheinen nur bei `mode=update`/`enrich`, alle vorausgewählt; ihre Auswahl fließt als `fields` in Vorschau UND Apply, damit die Vorschau exakt das zeigt, was angewendet wird. Zeilen-Checkboxen sitzen an jeder `updated`-Zeile der Vorschau, default angehakt.

## Risks / Trade-offs

- **Datei-Wechsel zwischen Vorschau und Apply** würde Zeilennummern verschieben. Im UI nicht möglich (gleiche `File`), per direktem API-Aufruf theoretisch — akzeptiert, da kein Datenverlust entsteht (es würde lediglich die falsche Zeile übersprungen/geschrieben; ein erneuter Lauf korrigiert).
- **`fields` mit unbekannten Spaltennamen** werden schlicht ignoriert (matchen keinen Schreibpfad) — robust statt fehlerträchtig.
- **Doppelte Quelle der Spaltennamen** (Backend-DB-Spalten ↔ Frontend-Checkbox-Keys) muss konsistent bleiben. Mitigation: zentrale Liste im Frontend als Konstante mit Label↔Spalte, identisch zu den Backend-`column`-Strings; in der Spec dokumentiert.
