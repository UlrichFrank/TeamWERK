# Design — Halber Beitrag bei Ein-/Austritt und im ersten Abrechnungsjahr

## Kontext

Der Beitragslauf berechnet pro Mitglied serverseitig den Jahresbeitrag
(`internal/beitragslauf/compute.go` → `computeItem`). Preview, Export-Daten (für die
clientseitige `pain.008`) und Confirm gehen alle durch `buildPreview` → denselben
`computeItem`-Pfad. **Eine** Änderung an `computeItem` wirkt daher konsistent auf alle drei
Wege.

## Entscheidung: Halbierungsregel

Ein Mitglied zahlt **50 %** (exakt `betrag_cent / 2`, ganzzahlig), wenn **eine** Bedingung
greift; sonst 100 %. Ermäßigungen stapeln nicht.

```
  HALB if …                                          Datenquelle
  ─────────────────────────────────────────────────────────────────
  R4  Saison.is_inaugural = 1   (Verein-Erstjahr)    seasons.is_inaugural (NEU)
  R3  start_date ≤ join_date ≤ end_date              members.join_date  (existiert)
  R2  status=ausgetreten ∧ exit_date ∈ Saisonfenster members.exit_date  (NEU)
```

**Exakte Halbierung:** `betrag_cent / 2` (Integer-Division). Beiträge sind in der Praxis
volle Euro (z. B. 22600 ct → 11300 ct), teilen also glatt. Sollte je ein ungerader
Cent-Betrag auftreten, schneidet die Integer-Division ab (Verein erhält nie mehr als die
exakte Hälfte). Bewusst **kein** Round-half-up.

**Kein Stacking:** Ein bool `half` genügt; mehrere zutreffende Gründe ergeben weiterhin nur
eine Halbierung. Der angezeigte Grund folgt der Priorität R4 → R3 → R2 (Erstjahr ist
dominanter Erklärungsgrund).

## Entscheidung: Inklusion unterjähriger Austritte

Heute filtert die SQL `WHERE status NOT IN ('ausgetreten','honorar','anwaerter')` Austritte
schon beim Laden weg. Das muss aufweichen, damit R2 greifen kann:

```
  Laden:   WHERE status NOT IN ('honorar','anwaerter')      (ausgetreten bleibt drin)
  Compute: status=ausgetreten
             └ exit_date ∈ [start,end]  → EINBEZIEHEN, Kategorie wie aktiv, HALB
             └ sonst                    → exclStatusInaktiv (wie bisher ausgeschlossen)
```

**Kategorie eines unterjährigen Austritts:** abgeleitet aus `home_club_id` (bleibt am
Datensatz) → `aktiv_mit` / `aktiv_ohne`, danach halbiert. Begründung: Das Mitglied war bis
zum Austritt aktiv; eine historische Statusverfolgung gibt es nicht. `pausiert`/`passiv`
ohne Austritt bleiben unverändert.

Honorar/Anwärter bleiben **immer** ausgeschlossen (keine Datumsausnahme).

## Entscheidung: `is_inaugural` als Saison-Flag

Das „erste Abrechnungsjahr" ist das **erste des Vereins** (einmalige Startkonzession), nicht
das erste je Mitglied. Ein per-Saison-Flag ist:

- **reproduzierbar** über Preview → Export → Confirm (alle lesen dieselbe Saison),
- **selbst-auslaufend** (nur die eine Saison trägt es),
- **kontrolliert** durch den Admin in der Saisonverwaltung.

Alternative „erste Saison ohne Protokoll erkennen" wurde verworfen (fragil, abhängig vom
Dateisystem-Zustand).

## Entscheidung: Pflicht-Eintrittsdatum + Bestands-Backfill

```
  join_date:
    Bestand   Migration backfillt NULL → impliziter Tag VOR erstem Saisonstart
              ⇒ R3 greift nie ⇒ voller Beitrag (Erstjahr ohnehin halb via R4)
    Zukunft   App-Validierung: Pflichtfeld bei Anlage/Bearbeitung
  exit_date:
    Pflicht (App-Validierung), sobald status = ausgetreten gesetzt wird
    DB-Spalte bleibt nullbar (kein Zwangs-Backfill von Altbestand)
```

**Backfill-Sentinel:** Startdatum der frühesten Saison **minus 1 Tag** (bzw. ein fixes
frühes Datum, falls keine Saison existiert). Bewusst **nicht** `created_at` — die
Datensätze entstanden bei Systemeinführung (≈ jetzt) und lägen sonst *im* aktuellen
Saisonfenster, was R3 fälschlich auslösen würde.

## Datumsvergleich

`join_date`/`exit_date` und `start_date`/`end_date` werden als ISO-Datum (`YYYY-MM-DD`,
auf 10 Zeichen geschnitten) verglichen. Saisonfenster ist **inklusive** beider Grenzen
(`start ≤ d ≤ end`). `loadSeason` liefert künftig zusätzlich `end_date` und `is_inaugural`.

## Auswirkungen auf Datenfluss

```
  buildPreview ─ loadSeason → (label, start, end, is_inaugural)
               ─ LoadMembersForLauf → inkl. join_date, exit_date, ausgetretene
               └ computeItem(m, saetze, season)
                    ├ Inklusion (ausgetreten-Sonderfall)
                    ├ Kategorie + voller Betrag (wie bisher)
                    └ half? → betrag/=2, half_reason setzen
  Preview JSON  : + "half": bool, + "half_reason": "eintritt|austritt|erstjahr"
  ExportData    : nutzt halbierten it.BetragCent automatisch (pain.008 stimmt)
  Confirm       : Betrag kommt aus Client (= halbierter Preview-Wert) → Protokoll stimmt
```

## Test-Strategie

Tabellengetriebene Compute-Tests (reine Funktion) + Handler-Tests über `testutil`.
`TestPreview_NeumitgliedZahltVollenBeitrag` wird zu „…ZahltHalbenBeitrag" umgekehrt.
Siehe `tasks.md` → `## Test-Anforderungen`.
