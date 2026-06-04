## Why

Im Handball gibt es neben der regulären Saisonphase Qualifikationszeiträume, in denen eine Mannschaft in veränderter Zusammensetzung (anderer Spielerkader, teils hochgezogene Jugendspieler, ggf. anderer Trainer) antritt. Diese Qualifikationskader existieren parallel zum regulären Kader derselben Altersklasse/Geschlecht und müssen separat verwaltbar sein.

## What Changes

- Die `kader`-Tabelle erhält zwei neue Felder: `type` (`regular` | `qualification`) und `is_active`
- Der bestehende UNIQUE-Constraint auf `(season_id, age_class, gender, team_number)` wird durch zwei partielle Unique-Indizes ersetzt, die jeweils nur einen aktiven Kader pro Typ erlauben
- Alle bestehenden Kader werden per Migration auf `type='regular'`, `is_active=1` gesetzt
- Inaktive Kader (`is_active=0`) bleiben als historische Datensätze erhalten
- Im Saisons-Tab der Admin-Einstellungen kann pro Altersklasse/Geschlecht ein optionaler Qualifikationskader aktiviert werden
- Neuen Qualifikationskader anlegen: Name + Altersklasse + Geschlecht, Spieler/Trainer danach separat befüllen

## Capabilities

### New Capabilities

- `qualifikations-kader`: Verwaltung paralleler Qualifikationskader neben regulären Kadern einer Saison, inkl. Aktivierungssteuerung pro Altersklasse/Geschlecht im Admin-UI

### Modified Capabilities

*(keine bestehenden Specs betroffen — Kader-Verwaltung war bislang nicht formal spezifiziert)*

## Impact

- **DB-Schema:** Migration 015 — `ALTER TABLE kader`, neuer UNIQUE-Index (partiell)
- **Backend:** `internal/kader/handler.go` — Kader-Erstellung, Listing, Aktivierungsendpunkt
- **API:** Neuer Endpunkt `PUT /api/admin/kader/:id/activate` sowie `type`-Feld in Kader-Responses
- **Frontend:** `web/src/pages/AdminSaisonPage.tsx` (Saisons-Tab) — Auswahl aktiver Kader pro Altersklasse; `web/src/pages/AdminKaderPage.tsx` (Kader-Übersicht) filtert auf `is_active=1`
- **Keine neuen Abhängigkeiten**
