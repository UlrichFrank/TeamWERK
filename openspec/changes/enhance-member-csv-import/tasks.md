## 1. Backend — IBAN-Validierung

- [x] 1.1 Hilfsfunktion `validateIBAN(s string) (bool, string)` in `internal/members/handler.go` implementieren (MOD-97 via `math/big`, Längenprüfung für DE-IBANs)
- [x] 1.2 `ImportRow`-Struct um Feld `IBANWarning string` erweitern und in JSON-Response einbinden
- [x] 1.3 IBAN-Validierung im Import-Flow aufrufen: bei ungültiger IBAN `IBANWarning` setzen und IBAN nicht in `setClauses`/INSERT aufnehmen

## 2. Backend — Fehlende Felder importieren

- [x] 2.1 Spalten-Aliases erweitern: `Mitglied seit` → `join_date` in `columnAliases`-Map
- [x] 2.2 INSERT für neue Mitglieder um `street`, `zip`, `city`, `join_date`, `iban` (wenn gültig), `account_holder`, `sepa_mandat` erweitern
- [x] 2.3 UPDATE-Logik (`addChange`/`addNullableChange`) für dieselben sieben Felder ergänzen; SEPA-Normalisierung (`"vorliegend"` → `"1"`) einbauen
- [x] 2.4 DB-Lookup-Query (`SELECT ... FROM members`) um die neuen Felder erweitern, damit der Änderungsvergleich funktioniert

## 3. Backend — Email-Klassifizierung und Verknüpfung

- [x] 3.1 Hilfsfunktion `classifyEmail(email, firstName string, dob string) string` implementieren: gibt `"eigen"`, `"eltern"` oder `"kind-eigen"` zurück
- [x] 3.2 `applyLinkUpdates` anpassen: statt `Benutzer_Email`/`Erziehungsberechtigter*_Email` die CSV-Spalten `Email` und `Email 2` per `classifyEmail` verarbeiten
- [x] 3.3 Für `"eigen"`: User suchen und `members.user_id` setzen (nur wenn noch nicht gesetzt)
- [x] 3.4 Für `"eltern"`: User suchen und `family_links`-Eintrag anlegen (wenn noch nicht vorhanden)
- [x] 3.5 Für `"kind-eigen"` oder User nicht gefunden: Notiz im Report-`Changes`-Array hinzufügen

## 4. Backend — Preview-Modus

- [x] 4.1 `mode=preview` als gültigen Wert in der Mode-Validierung ergänzen
- [x] 4.2 `dryRun bool`-Parameter durch die Import-Funktion führen: alle `ExecContext`-Aufrufe (INSERT/UPDATE members, INSERT family_links, UPDATE user_id) bei `dryRun=true` überspringen
- [x] 4.3 Sicherstellen dass der Report bei `mode=preview` identisch befüllt wird wie bei `mode=update`

## 5. Frontend — Preview-Flow im Import-Modal

- [x] 5.1 Import-Modal um zweistufigen Flow erweitern: erst „Vorschau" (`mode=preview`), dann „Anwenden" (`mode=update`) mit derselben Datei
- [x] 5.2 Preview-Report anzeigen: Zusammenfassung (X neu, Y aktualisiert, Z Fehler) mit aufklappbarer Detailliste der Änderungen pro Zeile
- [x] 5.3 IBAN-Warnings im Report hervorheben (z.B. gelbes Warn-Icon mit Tooltip)
- [x] 5.4 „Anwenden"-Button erst nach erfolgreichem Preview aktivieren; nach dem Anwenden Modal schließen und Mitgliederliste neu laden
