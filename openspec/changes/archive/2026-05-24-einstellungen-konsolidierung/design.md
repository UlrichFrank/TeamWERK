## Context

Drei bestehende Pages werden zu einer zusammengeführt:

| Alte Page | Route | Größe | Inhalt |
|-----------|-------|-------|--------|
| `AdminClubPage` | `/admin/verein` | 45 Z. | Formular: Vereinsname + Adresse |
| `AdminSeasonsPage` | `/admin/saisons` | 184 Z. | Liste + inline Anlegen-Formular |
| `AdminAgeClassRulesPage` | `/admin/altersklassen` | 166 Z. | Tabelle mit inline-Edit pro Zeile |

Das Saison-Modal-Muster kommt von `AdminDutyTypesPage`: Button oben rechts öffnet Create-Modal, Bearbeiten-Button pro Zeile öffnet Edit-Modal via `EditModal`-Komponente.

## Goals / Non-Goals

**Goals:**
- Einheitlicher Einstiegspunkt `/admin/einstellungen`
- Saisons: Modal-Muster (anlegen + bearbeiten) wie Diensttypen
- Altersklassen und Verein: Inhalt und Logik unverändert, nur neue Heimat
- Alte Routen leiten weiter (keine toten Links)

**Non-Goals:**
- Altersklassen auf Modal umstellen (inline ist dort OK — es gibt keine Liste, nur fixe Zeilen)
- Neue Felder an bestehenden Entitäten
- Validierung über das bereits vorhandene Maß hinaus

## Decisions

### D1: Tab-Navigation (nicht Accordion, nicht Sections)

Drei klar getrennte Bereiche, jeder mit eigener API-Logik. Tabs ermöglichen:
- Gezieltes Laden (nur aktiver Tab lädt Daten)
- Saubere URL-Addressierbarkeit via `?tab=saisons`
- Auf Mobile: horizontale Tab-Leiste (3 Tabs passen gut)

```
┌─────────────────────────────────────────────┐
│ Einstellungen                               │
│                                             │
│ [Verein] [Saisons] [Altersklassen]          │
│ ─────────────────                           │
│                                             │
│  <Tab-Inhalt>                               │
│                                             │
└─────────────────────────────────────────────┘
```

Aktiver Tab wird via `?tab=verein|saisons|altersklassen` im URL gespeichert — direktes Ansteuern via alter Routen möglich (Redirect setzt den Tab-Parameter).

### D2: Saison-Bearbeiten — eigener `PUT /api/admin/seasons/{id}`

Aktuell existiert nur `PUT /api/admin/seasons/{id}/activate`. Ein allgemeines PUT für name/start_date/end_date ist minimal und sauber. Constraint: Aktive Saison darf bearbeitet werden (Name/Datum ändern schadet nicht), aber der Admin sieht einen Hinweis im Modal.

### D3: Alte Routen als React-Router-Redirects, nicht 301

Alle Links im System (Sidebar, potenzielle Deep-Links) bleiben funktionsfähig. Da dies eine SPA ist, reicht `<Navigate to="/admin/einstellungen?tab=..." replace />` in App.tsx — kein Backend-Change nötig.

### D4: Saison-Modal-Felder

| Feld | Create | Edit |
|------|--------|------|
| Saison (Preset-Dropdown) | ✓ (auto-füllt Name+Datum) | — |
| Name | ✓ | ✓ |
| Startdatum | ✓ | ✓ |
| Enddatum | ✓ | ✓ |
| Hinweis wenn aktiv | — | ✓ (readonly Info-Badge) |

„Aktivieren" und „Löschen" bleiben als Buttons in der Zeile (nicht im Modal).

### D5: Daten laden

Jeder Tab lädt seine Daten beim ersten Aktivieren (lazy), cached im State der Page. Tab-Wechsel ohne erneuten API-Call (außer explizitem Refresh nach Mutation).

## Risks / Trade-offs

- **Seitenrefs in E-Mails / externen Links**: Falls jemand `/admin/saisons` geleseztzt hat, funktionieren Redirects. Mitigation: 3 Redirects in App.tsx.
- **Page-Größe**: Alle drei Bereiche in einer Datei → ~350-400 Zeilen. Akzeptabel bei Tab-Struktur; Subkomponenten für jeden Tab halten es lesbar.
- **Saison bearbeiten mit aktiver Saison**: Datum rückwirkend ändern kann bestehende Slots außerhalb der Saison lassen. Mitigation: Hinweis-Text im Modal, keine technische Sperre.

## Migration Plan

1. Backend: `PUT /api/admin/seasons/{id}` deployen
2. Frontend: neue `AdminSettingsPage` deployen, alte Pages entfernen, Routen/Redirects setzen
3. Nav-Eintrag aktualisieren (AppShell)
4. Alte Page-Dateien löschen
