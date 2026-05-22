## Why

Aktuell gibt es exakt eine globale Spielplan-Vorlage (`is_active=1`). Heim- und Auswärtsspiele sowie Turniere erfordern aber unterschiedliche Dienstpläne (z.B. kein Auf-/Abbau bei Auswärtsspielen). Trainer und Admins sollen je Vorlage steuern können, für welchen Spieltyp sie gilt, und mehrere Vorlagen parallel verwalten.

## What Changes

- **Mehrere Dienstplan-Vorlagen** statt einer einzigen aktiven: `game_templates` unterstützt n Einträge, `is_active` entfällt
- **Vorlagen-Typ** (`template_type`): `heim`, `auswärts`, `generisch` — je Vorlage einstellbar
- **Umbenennung** aller User-facing Bezeichnungen von „Spiel-Vorlage" zu „Dienstplan-Vorlage" (UI-Labels, Menüpunkt, Route)
- **REST-API-Umbenennung**: `/api/admin/game-template` → `/api/admin/duty-templates` (**BREAKING**)
- **Neue Listenansicht** (`/admin/dienstplan-vorlagen`) nach dem Muster der Mitglieder-Seite: Tabelle + Detailseite
- **Löschen** einer Vorlage direkt aus der Tabelle
- Slot-Generierung (`CreateGame`, `RegenerateSlots`, `PreviewSlots`) wählt die passende Vorlage anhand `template_type` und Spieltyp (`is_home`)
- **Spielplanung**: Beim Anlegen eines Spiels muss der Typ (Heim/Auswärts) gewählt werden — die passende Vorlage wird automatisch verwendet. Alternativ kann ein **generisches Event** (z. B. Turnier, Trainingslager) mit eigenem Namen eingeplant werden; dabei wird eine `generisch`-Vorlage gewählt

## Capabilities

### New Capabilities

- `duty-templates`: Verwaltung mehrerer Dienstplan-Vorlagen mit Typ (heim/auswärts/generisch), tabellarische Übersicht, Detailseite, Löschen

### Modified Capabilities

- `games` (Spielplanung): Beim Anlegen eines Spiels Pflichtfeld Heimspiel/Auswärtsspiel; generische Events (z. B. Turnier) mit freiem Namen und `generisch`-Vorlage anlegbar

## Impact

- **DB**: Migration: `game_templates` erhält `template_type TEXT CHECK('heim','auswärts','generisch')`, `is_active`-Spalte bleibt für rückwärtskompatible Migration (wird aber nicht mehr für Single-Select verwendet)
- **Backend**: `internal/games/handler.go` — alle Endpunkte unter neuem Pfad, Slot-Generierung sucht Vorlage nach Typ; alte Pfade können wegfallen (Breaking Change, kein Public-API)
- **Frontend**: `AdminGameTemplatePage.tsx` → aufgeteilt in `AdminDutyTemplatesPage.tsx` (Liste) + `AdminDutyTemplateDetailPage.tsx` (Detail); Route `/admin/spielplan-template` → `/admin/dienstplan-vorlagen` + `/admin/dienstplan-vorlagen/:id`; Nav-Eintrag anpassen
- **Slot-Auswahl-Logik**: Bei `is_home=true` → Vorlage Typ `heim` (fallback `generisch`); `is_home=false` → `auswärts` (fallback `generisch`); generisches Event → immer `generisch`
- **Spielanlegen-UI**: Auswahl Heimspiel/Auswärtsspiel als Pflichtfeld; separater Pfad „Generisches Event" mit Freitext-Name und automatischer Auswahl einer `generisch`-Vorlage
