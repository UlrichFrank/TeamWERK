## Context

Die Dashboard-Seite (`web/src/pages/DashboardPage.tsx`) rendert vier Akkordeon-Sektionen mit jeweils eigener Zeilen-Struktur. Die Sektionen teilen sich keinen visuellen Standard, obwohl drei davon konzeptionell „Termin-bezogene Liste pro Tag" sind. Im Akkordeon stehen die Sektionen heute in der Reihenfolge `Termine → Dienste → Team → Fahrgemeinschaften`. Die Team-Sektion ist navigationslastig (Link zur Mannschaftsseite) und nicht termingebunden.

Backend-seitig liefert `GET /api/dashboard` für jede Zusage in `carpoolingConfirmed[].paarungen[]` heute nur `paarungId` und `partnerName`. Ein Treffpunkt ist im Datenmodell **pro Mitfahr-Eintrag** (Bieter- und Sucher-Eintrag jeweils separat) hinterlegt, aber nicht pro Paarung.

## Goals / Non-Goals

**Goals:**
- Eine einzige, wiederverwendbare Zeilen-Komponente für Termine, Dienste, Fahrgemeinschaften
- Identisches Spaltenraster über die drei terminbezogenen Sektionen
- Reihenfolge der Sektionen spiegelt Priorität wider (Aktionen vor Navigation)
- Partner-Treffpunkt sichtbar bei bestätigten Fahrgemeinschaften

**Non-Goals:**
- Keine Änderung an Mein Team (andere Struktur, kein Datum)
- Keine Änderung an Dashboard-Reload-Logik, Auth, Section-Toggle-State
- Keine neue Capability — nur additive Erweiterung einer bestehenden
- Keine Mobile-spezifische Sonderbehandlung (Termine-Layout funktioniert auf beiden Breakpoints schon)

## Decisions

### Spaltenraster (vier Spalten, fix)

```
┌──────────┬─────┬────────────────────────────────┬─────┐
│ w-10     │ w-4 │ flex-1 min-w-0                 │ w-4 │
│ Wochentg │ Ico │ Zeile 1: text-sm font-medium    │  →  │
│ Tag.Mon  │     │ Zeile 2: text-xs text-muted     │     │
└──────────┴─────┴────────────────────────────────┴─────┘
   gap-3        gap-1.5
```

Identische Tailwind-Klassen in allen drei Sektionen → optisch eine durchgehende Tabelle, obwohl es vier getrennte Akkordeon-Karten sind.

### Inhalts-Mapping pro Sektion

| Sektion          | Titel (Zeile 1)        | Subtitel (Zeile 2)                        | Icon |
|------------------|------------------------|--------------------------------------------|------|
| Termine          | `e.title`              | `e.teamName · e.time`                      | `Dumbbell`/`Home`/`Plane` |
| Dienste-Slot     | `s.dutyTypeName`       | `opponent · s.eventTime`                   | `Check` |
| Dienste-Fallback | `N offene Dienste`     | `opponent`                                 | `Info` (lucide) |
| Fahrt-Zusage     | `p.partnerName`        | `opponent · partnerTreffpunkt` *bzw.* `opponent` | `Check` |
| Fahrt-Gesuch     | `req.requesterName`    | `{plaetze} Plätze · treffpunkt` *bzw.* `{plaetze} Plätze · gameTitle` | `Search` |

Falls der zweite Token (`partnerTreffpunkt` bzw. `treffpunkt`) leer ist, fällt er ersatzlos weg. Der erste Token bleibt immer gesetzt — Zeile 2 ist nie leer.

### `partnerTreffpunkt` — Quelle der Wahrheit

Aus Sicht des Nutzers liefert das Backend immer den Treffpunkt der **Gegenseite**:

```
Ist mein Eintrag der Bieter-Teil der Paarung → partnerTreffpunkt = sucher.treffpunkt
Ist mein Eintrag der Sucher-Teil der Paarung → partnerTreffpunkt = bieter.treffpunkt
```

Begründung: Den eigenen Treffpunkt kennt der User; relevant ist die Pickup-/Treff-Info, die er vom Partner braucht. Konsistent mit `partnerName`.

Bei Kinder-Paarungen (über `family_links`) gilt dieselbe Logik aus Sicht des Kindes — dessen Seite zählt als „eigene Seite".

**Implementierung:** SQL-Subquery wählt die jeweils gegenüberliegende `mitfahrgelegenheiten.treffpunkt` über `mitfahrt_paarungen.bieter_id` / `sucher_id`. Bei NULL → leerer String im JSON.

### Sortierung Fahrgemeinschaften

Die flache Liste mischt Zusagen und Gesuche und sortiert chronologisch nach Datum (aufsteigend). Innerhalb eines Datums: Zusagen vor Gesuchen (Beobachtungs- vor Handlungs-Items).

### Reihenfolge der Sektionen

`Meine Termine → Meine Dienste → Fahrgemeinschaften → Mein Team`

`openSections`-Defaults bleiben `{ termine: true, dienste: true, fahrt: true, team: true }`. Section-IDs werden nicht umbenannt — nur die JSX-Reihenfolge geändert.

### Komponentenstruktur

Neue Datei-interne Komponente `DashboardRow` in `DashboardPage.tsx` (kein Extract in `components/`, da dashboard-spezifisch):

```ts
function DashboardRow({
  to, dateISO, icon, title, subtitle, badge,
}: {
  to: string
  dateISO: string
  icon: React.ReactNode
  title: string
  subtitle: string | React.ReactNode
  badge?: React.ReactNode  // optional, z. B. ExtendedBadge für Termine
})
```

Alle drei Sektions-Komponenten konstruieren ihre Zeilen über `DashboardRow`. Wenn die Sektion „kein-Eintrag"-Fall hat, bleibt der bisherige `<p>`-Hinweis erhalten.

## Risks / Trade-offs

- **Redundantes Datum**: bei Dienste-Slots steht das Spiel-Datum auf jeder Slot-Zeile, obwohl alle dasselbe Spiel betreffen. → Vom User explizit akzeptiert für maximale Spalten-Treue.
- **Verlust der Kategorisierung in Fahrt**: Zusagen vs. Gesuche unterscheiden sich nur noch über das Icon. Mitigation: deutliche Icon-Wahl (`Check` mit `text-brand-success` vs. `Search` mit `text-brand-text-muted`).
- **Backend-Diff sehr klein, aber dual-mode-Subquery muss „eigene Seite" über `user_id` und `family_links`-Kinder bestimmen**. Bestehende Logik in `queryCarpoolingConfirmed` macht das schon — `partnerTreffpunkt`-Auswahl reitet auf demselben CASE-Branch.
- **API-Vertrag**: zusätzliches Feld in einer bestehenden Liste. Additiv, nicht-breaking; alte Clients ignorieren es.

## Migration Plan

Keine DB-Migration. Reiner Code-Change. Deploy via `make deploy`.

Backend-Änderung MUSS zuerst deployt werden (oder gemeinsam), damit das Frontend `partnerTreffpunkt` immer findet (auch wenn leer ist die Property dann undefined → Subtitel fällt auf „opponent only" zurück; tolerant).
