## Context

Die KalenderPage zeigt pro Kalendertag Pills für Trainings und Spiele. Bisher navigiert ein Klick auf ein Spiel-Pill immer zur Spieltag-Detailseite (`/kalender/{id}`), die Dienst-Slots verwaltet. Nutzer (insbesondere Trainer und Admins) wollen aber auch direkt Spieldaten (Datum, Uhrzeit, Gegner, Teams) korrigieren können, ohne den Kalender-Kontext zu verlassen.

Die Mitfahrgelegenheiten-Seite verwendet bereits einen „Team | Meine"-Toggle als Vorlage für das visuelle Muster.

## Goals / Non-Goals

**Goals:**
- Toggle „Spiel | Dienst" im KalenderPage-Header steuert das Klickverhalten von Spiel-Pills
- Dienst-Modus (default): bestehende Navigation zu `/kalender/{id}` bleibt unverändert
- Spiel-Modus: Klick öffnet `GameModal` (Edit für admin/trainer, Read-only sonst)
- `GameModal` erlaubt das Bearbeiten von Datum, Uhrzeit, Gegner und Teams via `PUT /admin/games/{id}`
- Sonstiges-Filter im Spiel-Modus deaktiviert (visuell + funktional) und automatisch abgewählt

**Non-Goals:**
- Dienstverwaltung im `GameModal` (keine Slot-Generierung, keine Assignments)
- `event_type` ist nicht editierbar (Backend unterstützt es nicht)
- Neue API-Endpunkte
- Veränderung des Dienst-Modus-Verhaltens

## Decisions

### 1. Toggle-State als lokaler React-State

Der `viewMode`-State (`'spiel' | 'dienst'`) bleibt lokal in `KalenderPage` — kein URL-Parameter, kein globaler Store.

**Rationale:** Der Toggle ist eine reine UI-Präferenz ohne tiefere Navigation. Ein URL-Parameter würde Boilerplate für keinen realen Mehrwert erzeugen (kein Deeplink-Bedarf).

**Alternative:** `?mode=spiel` im URL → abgelehnt, da kein Nutzer diesen Link teilen würde.

### 2. GameModal als eigenständige Komponente

`GameModal` wird als neue Datei `web/src/components/GameModal.tsx` angelegt (kein Inline-JSX in `KalenderPage`).

**Rationale:** Die Komponente ist eigenständig testbar, kann später von anderen Seiten genutzt werden (z.B. SpielplanPage), und hält `KalenderPage` überschaubar.

### 3. Rollenbasiertes Edit/Read-only im selben Modal

Dieselbe `GameModal`-Komponente rendert entweder ein Formular oder eine Read-only-Ansicht, gesteuert durch ein `editable`-Prop (berechnet aus `user.role`).

**Rationale:** Vermeidet zwei separate Komponenten mit nahezu identischem Markup.

### 4. Sonstiges-Filter: disable, nicht verstecken

Im Spiel-Modus wird der Sonstiges-Button visuell deaktiviert (`opacity-40 cursor-not-allowed`) und beim Moduswechsel automatisch aus dem aktiven Filter-Set entfernt. Er bleibt sichtbar.

**Rationale:** Verstecken würde das Layout verschieben; ein dauerhaft sichtbarer, gesperrter Button kommuniziert klar, dass der Typ in diesem Modus nicht verfügbar ist.

## Risks / Trade-offs

- **Parallele Edits:** Zwei Trainer könnten dasselbe Spiel gleichzeitig bearbeiten → last-write-wins (SQLite-Transaktionen, kein Optimistic Locking). Risiko ist gering bei der Nutzerzahl.
- **Kein SSE-Broadcast nach GameEdit:** `PUT /admin/games/{id}` ruft bisher kein `hub.Broadcast` auf. Nach dem Speichern soll der Kalender neu laden — dies erfordert einen manuellen Reload-Trigger (oder SSE-Ergänzung im Backend, falls gewünscht). Zunächst: `refetch` nach erfolgreichem PUT.

## Open Questions

*(keine)*
