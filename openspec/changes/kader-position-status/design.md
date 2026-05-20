## Context

AdminKaderPage zeigt derzeit Member pro Kader, aber keine Position-Übersicht. Member haben bereits ein `positions: string[]`-Array mit Positionen, die sie spielen können (z.B. `["Linksaußen", "Rechtsaußen"]`). Die neuen Position-Status-Indikatoren sollen ohne API-Änderungen entstehen, nur durch Client-seitige Aggregation.

## Goals / Non-Goals

**Goals:**
- Visuelle Position-Besetzungs-Übersicht pro Kader anzeigen
- Sehr kompakt (minimal Platz-Verbrauch) zwischen Jahrgänge-Toggle und Trainer-Suche
- Farb-/Kreis-Semantik für schnelle Überblicks-Diagnostik

**Non-Goals:**
- Keine Position-Management-UI (Hinzufügen/Bearbeiten von Positionen auf Members)
- Keine Position-Constraints oder Regeln erzwingen (z.B. "mind. 1 Torwart")
- Keine API-Änderungen
- Keine Backend-Logik (reine Frontend-Aggregation)

## Decisions

### Daten-Aggregation (Client-seitig)

**Decision:** Positionen auf dem Client zählen, nicht vom Backend.

**Rationale:** Member-Objekte enthalten bereits `positions`, keine neue API-Query nötig. Berechnung in React ist trivial.

**Alternatives considered:**
- Backend-Endpoint für Position-Summary: komplexer, unnötig für diese Use-Case
- Position-Counts in Kader-Response: würde API-Response aufblasen

### Positions-Konstanten

**Decision:** Harte Konstante der 7 Handball-Positionen mit Abkürzungen im Frontend:
```
{
  name: "Torwart",
  abbr: "TW",
  color: { 0: "red", 1: "yellow", 2: "green", 3: "blue" }
}
```

**Rationale:** Die Position-Namen sind stabiler (Sportregeln), Abkürzungen sind Standard in Handball. Keine Datenbank-Abhängigkeit.

### Visuelle Render-Strategie

**Decision:** `<PositionStatus members={kader.members} />` Komponente mit inline-Styling:

```
TW ⭕  LA 🟡  RA 🟢  RL ⭕  RM 🟢  RR 🟡  KL 🟢
       🟢            🟢                    🟢

Layout pro Position:
- Abkürzung (TW, LA, etc.) + vertikaler Kreis-Stapel nebeneinander
- Kreise pro Position VERTIKAL gestapelt direkt rechts der Abkürzung
- Kreise: 14px Durchmesser
- Jede [Abkürzung + Stapel]-Einheit wiederholt sich horizontal mit gap-3 zwischen Positionen
- Kreise vertikal: gap-1 zwischen den Kreisen
```

**Struktur:**
```
Position-Wrapper (flex row, gap-3)
├── Position 1
│   ├── Abkürzung (inline)
│   └── Kreis-Stapel (flex column, gap-1)
├── Position 2
│   ├── Abkürzung (inline)
│   └── Kreis-Stapel (flex column, gap-1)
├── ...
```

**Rationale:**
- Sehr kompakt, wenig Platz-Verbrauch
- Abkürzung + Stapel zusammen halten zusammengehörige Daten visuell
- Vertikale Stapelung pro Position zeigt Besetzungsanzahl auf einen Blick
- Tailwind Utility-Classes für Styling (wir verwenden bereits Tailwind)
- Keine neue CSS-Datei nötig

### Positionierung

**Decision:** `<PositionStatus />` zwischen Mode-Toggle (Gemischt/Dediziert) und Trainer-Suche.

```
┌──────────────────────┐
│ Jahrgänge: [Toggle]  │  ← existiert
│ TW LA RA...          │  ← neu
│ Trainer: [Suchen]    │  ← existiert
└──────────────────────┘
```

**Rationale:** Logische Reihenfolge: Spiel-Konfiguration → Position-Status → Trainers/Members

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| Position-Namen könnten in Zukunft ändern | Aktuell stabil; falls nötig, Backend-Konstanten exportieren + API ändern |
| Sehr kleine Kreise sind auf Mobile schwer zu sehen | akzeptabel für Admin-Seite; nicht critical |
| Keine Fehlerbehandlung wenn `positions` leer | Default: alle Positionen als "0" (rot) |

## Migration Plan

- Feature ist rein additiv (neuer UI-Block)
- Kein Rollout-Risiko
- Deployment: normales Frontend-Build, keine DB-Migration
