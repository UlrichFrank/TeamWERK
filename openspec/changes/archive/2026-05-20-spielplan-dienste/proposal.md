# Spielplan & automatische Dienstgenerierung

## Problem

Für jedes Heimspiel müssen heute 3+ Duty Slots manuell angelegt werden (Aufbau, Bewirtung, Abbau).
Bei mehreren Mannschaften und ~15 Heimspielen pro Saison ist das fehleranfällig und zeitaufwendig.
Außerdem gibt es keine Übersicht, welche Spieltage besetzt sind und welche nicht.

## Lösung

1. **Spielplan-Verwaltung** — Heimspiele manuell erfassen (Name, Datum, Uhrzeit, Mannschaft).
2. **Dienst-Vorlage** — Eine globale Heimspiel-Vorlage definiert, welche Diensttypen mit welcher
   Zeitverschiebung und Personenanzahl automatisch generiert werden.
3. **Auto-Generierung** — Beim Anlegen eines Heimspiels werden die Duty Slots per Vorlage erzeugt
   und erhalten automatisch die `team_id` des spielenden Teams.
4. **Kalender-Ansicht** — Monatliche Übersicht aller Heimspiele mit Besetzungsampel pro Spieltag.
5. **Spieltag-Detail** — Zeitleiste der generierten Dienste mit Besetzungsstand.

## Scope

**In scope:**
- Datenmodell: `games`, `game_templates`, `game_template_items`
- Admin-UI: Spielplan-Seite (Kalender + Detail), Template-Konfiguration
- Auto-Generierung der Duty Slots beim Spielanlegen (mit Vorschau)
- `source`-Feld auf `games` für spätere Import-Erweiterung (H4A / Handball360)
- Verknüpfung `duty_slots.game_id` (nullable, bestehende Slots kompatibel)

**Out of scope:**
- Import aus Handball4All / Handball360 (separates Feature)
- Auswärtsspiele (keine Dienste nötig, werden nicht verwaltet)
- Spielergebnis-Erfassung

## Rollen & Zugriff

| Aktion                          | admin | trainer |
|---------------------------------|-------|---------|
| Spiel anlegen / bearbeiten      | ✓     | —       |
| Template konfigurieren          | ✓     | —       |
| Spielplan-Kalender anzeigen     | ✓     | ✓       |
| Spieltag-Detail anzeigen        | ✓     | ✓       |

## Datenmodell (Übersicht)

```
games
├── id, team_id, opponent, date, time
├── is_home (nur Heimspiele werden verwaltet)
├── season_id, source ("manual")
└── template_id → game_templates

game_templates
├── id, name ("Heimspiel Standard")
└── game_duration_minutes (default 90)

game_template_items
├── template_id, duty_type_id
├── anchor ("start" | "end")
├── offset_minutes (negativ = vor Anpfiff)
├── duration_minutes
├── slots_count
└── role_desc

duty_slots (bestehend, erweitert)
└── game_id (nullable FK → games)  ← neu
```

## UX-Skizze

**Kalender:**
- Monatliche Grid-Ansicht, Spieltage farbcodiert nach Besetzungsgrad
- Mehrere Spiele pro Tag werden gestapelt angezeigt (z.B. [A1] ●●● / [B2] ●●○)
- Klick auf Spieltag → Detailansicht

**Spieltag-Detail:**
- Zeitleiste mit Diensten (Aufbau → Bewirtung → Abbau)
- Balken-Anzeige: `████░ 2/3` für Besetzungsstand
- Button „+ Dienst anlegen" für manuelle Ergänzungen

**Spielanlegen-Dialog:**
- Felder: Datum, Uhrzeit, Gegner, Mannschaft
- Vorschau der zu generierenden Slots
- Bestätigen / Anpassen / Überspringen

## Migrationspfad (Zukunft)

Das `source`-Feld auf `games` ist von Beginn an vorbereitet.
Wenn Handball4All oder Handball360 angebunden werden soll, wird ein separater
Import-Mechanismus das gleiche `games`-Modell befüllen — ohne Schema-Änderung.
