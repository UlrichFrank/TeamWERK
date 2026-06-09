## Why

Der bestehende Mitglieder-CSV-Import ignoriert 10 vorhandene DB-Felder (Adresse, Beitrittsdatum, IBAN, SEPA-Mandat, Kontoinhaber) und verknüpft Eltern-Accounts nicht automatisch — ein kompletter Datenimport aus der Vereinsverwaltung erfordert damit manuelles Nacharbeiten für jeden einzelnen Datensatz. Eine aktuelle Mitgliederliste mit 181 Einträgen liegt vor und soll vollständig importiert werden.

## What Changes

- **Adressfelder importieren**: `Adresse`, `PLZ`, `Ort` aus CSV → `street`, `zip`, `city` in `members`
- **Beitrittsdatum importieren**: `Mitglied seit` → `join_date`
- **Bankdaten importieren**: `IBAN`, `Kontoinhaber`, `SEPA Mandat` → `iban`, `account_holder`, `sepa_mandat`
- **IBAN-Validierung**: MOD-97-Prüfsumme vor dem Speichern; ungültige IBANs werden als Warning im Report gemeldet, der Rest des Datensatzes wird trotzdem importiert
- **Email-Klassifizierung**: `Email`- und `Email 2`-Spalten werden per Heuristik (Alter + Vorname im lokalen Teil der Adresse) als eigene Email oder Eltern-Email klassifiziert und entsprechend verknüpft (`members.user_id` bzw. `family_links`)
- **Preview-Modus**: neuer `mode=preview` führt die komplette Import-Logik aus, schreibt aber nichts in die DB — gibt denselben Report zurück wie `mode=update`, damit der Admin die geplanten Änderungen vor dem Anwenden prüfen kann

## Capabilities

### New Capabilities

- `member-csv-import-enhanced`: Erweiterter CSV-Import für Mitglieder — neue Felder, IBAN-Validierung, Email-Klassifizierung mit Eltern-Verknüpfung und Preview-Modus

### Modified Capabilities

*(keine bestehenden Specs betroffen — der bisherige Import ist nicht spezifiziert)*

## Impact

- **Backend**: `internal/members/handler.go` — `Import`-Funktion und `applyLinkUpdates` erweitern; neue `validateIBAN`-Hilfsfunktion (stdlib only: `math/big`)
- **Frontend**: `web/src/pages/MembersPage.tsx` oder Import-Modal — Preview-Button ergänzen, der Report mit IBAN-Warnings anzeigen
- **API**: `POST /api/members/import` — neuer `mode=preview` Parameter; `ImportRow` erhält neue Felder für IBAN-Warnings
- **Keine neuen Abhängigkeiten** — MOD-97 ist Pure-Go mit `math/big` aus der Stdlib
