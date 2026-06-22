## Why

Der Mitglieder-CSV-Import (`mode=update`/`enrich`) ist heute „alles oder nichts": Wer nur die IBAN nachpflegen will, übernimmt zwangsläufig auch jede andere abweichende Spalte, und wer eine Vorschau prüft, kann einzelne fehlerhafte Zeilen nicht gezielt aussparen. Beim Import echter Vereinsexporte (in denen einzelne Spalten veraltet oder einzelne Zeilen fragwürdig sind) führt das zu ungewolltem Überschreiben sauberer Bestandsdaten.

## What Changes

- **Feld-Auswahl:** Neues optionales Formfeld `fields` (komma-separierte DB-Spalten). Nur gelistete Spalten werden bei **Bestandsmitgliedern** aktualisiert; leer/abwesend = alle Felder (rückwärtskompatibel). `status`-Auswahl steuert mit das abgeleitete `beitragsfrei`. Greift nur auf den UPDATE-Pfad — neu angelegte Mitglieder (`created`) bekommen weiter alle CSV-Felder.
- **Mitglieder-Auswahl:** Neues optionales Formfeld `apply_lines` (komma-separierte CSV-Zeilennummern). Außerhalb des Dry-Runs werden nur Updates dieser Zeilen geschrieben; leer/abwesend = alle. Identität Vorschau→Apply über die CSV-Zeilennummer (Frontend sendet dieselbe Datei erneut).
- **Neuer Status `skipped`:** Bewusst abgewählte Bestandsmitglieder (hätten Änderungen, sind aber nicht in `apply_lines`) werden im Apply-Lauf als `skipped` gemeldet; `ImportReport` bekommt einen `skipped`-Zähler. Die Vorschau (Dry-Run) zeigt weiterhin **alle** Zeilen mit ihren Änderungen als `updated`.
- **Frontend:** Feld-Checkboxen in Schritt 1 (nur bei `mode=update`/`enrich`, alle vorausgewählt); pro `updated`-Zeile in der Vorschau eine Checkbox (default angehakt); `skipped` bekommt eigenes Icon/Farbe/Badge.

Keine Breaking Changes: ohne `fields`/`apply_lines` verhält sich die Route exakt wie bisher.

## Capabilities

### New Capabilities
- `member-csv-import-selective`: Selektive Steuerung des Mitglieder-CSV-Imports über Feld-Whitelist (`fields`) und Zeilen-Auswahl (`apply_lines`) inkl. Report-Status `skipped`.

### Modified Capabilities
<!-- Keine bestehende Requirement wird umgeschrieben; member-csv-import-enhanced und members-csv-enrich-mode bleiben gültig und werden additiv durch die neue Whitelist/Zeilen-Auswahl ergänzt. -->

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `POST /api/members/import` | `TestImport_FieldsWhitelist_NurIBAN` | Mit `fields=iban` und abweichendem Status+IBAN wird nur `iban` geschrieben; `status` bleibt unverändert. |
| `POST /api/members/import` | `TestImport_FieldsWhitelist_StatusSteuertBeitragsfrei` | Ohne `status` in `fields` bleibt `beitragsfrei` unverändert, auch wenn die CSV einen `beitragsfrei`-Status liefert. |
| `POST /api/members/import` | `TestImport_LeeresFields_AlleFelder` | Ohne `fields` werden wie bisher alle nichtleeren, abweichenden Werte übernommen (Regression). |
| `POST /api/members/import` | `TestImport_FieldsWhitelist_NeuesMitgliedAllFelder` | Im `update`-Modus bekommt ein neu angelegtes Mitglied trotz `fields=iban` alle CSV-Felder. |
| `POST /api/members/import` | `TestImport_ApplyLines_NurAusgewaehlteZeile` | Mit `apply_lines=2` (ohne Dry-Run) wird nur Zeile 2 geschrieben; Mitglied aus Zeile 3 bleibt unverändert. |
| `POST /api/members/import` | `TestImport_ApplyLines_AbgewaehltIstSkipped` | Eine abgewählte Zeile mit Änderungen erhält Status `skipped`; `skipped`-Zähler steigt; DB unverändert. |
| `POST /api/members/import` | `TestImport_ApplyLines_DryRunIgnoriert` | Mit `mode=preview` und gesetztem `apply_lines` werden alle Zeilen als `updated` gemeldet, nichts geschrieben. |

**Garantierte Invariante:** Ein Bestandsmitglied-Feld F wird genau dann durch den Import verändert, wenn (a) die CSV einen nichtleeren, abweichenden Wert für F liefert, (b) F im Modus erlaubt ist (kein `enrich`-Überschreiben), (c) F (bzw. bei `beitragsfrei` die Spalte `status`) in `fields` enthalten oder `fields` leer ist, UND (d) kein Dry-Run läuft und die Zeile in `apply_lines` enthalten oder `apply_lines` leer ist. Andernfalls bleibt F unverändert.

## Impact

- **API:** `POST /api/members/import` — zwei neue optionale Formfelder (`fields`, `apply_lines`), erweiterter `ImportReport` (`skipped`-Zähler), neuer `ImportRow.status = "skipped"`. Auth-Tier unverändert (Vorstand).
- **Backend:** `internal/members/handler.go` (`Import`) — Whitelist-Filter in `addChange`/`addNullableChange` und den Inline-Blöcken (jersey/sepa/beitragsfrei/iban), `selected`-Gate im Apply, `skipped`-Reporting.
- **Frontend:** `web/src/pages/MembersPage.tsx` — Feld-Checkboxen (Schritt 1), Zeilen-Checkboxen (Vorschau), `skipped`-Darstellung; `ImportReport`-Typ.
- **Tests:** `internal/members/import_test.go` — neue Fälle für Feld-Whitelist, `apply_lines`, `skipped`.
