## Context

`TrainingsDetailPage.tsx` hat aktuell drei Bereiche: Session-Info-Karte, Rückmeldungen-Karte (aus `session.responses`), Anwesenheits-Karte (aus `GET /attendances`, nur Trainer + vergangene Sessions). Die Attendance-API gibt `rsvp_status` bereits mit zurück — sie ist damit eine vollständige Datenquelle für die vereinte Tabelle.

## Goals / Non-Goals

**Goals:**
- Eine Tabelle für RSVP + Anwesenheit (Trainer)
- Stat-Badges im Session-Header
- Auto-save Anwesenheit, Fehler-Rollback
- Kommentar-Indikator + Tooltip/Tap

**Non-Goals:**
- Inline-RSVP-Bearbeitung für Spieler (bleibt in TrainingsPage oder per separatem Change)
- Optimistic locking bei gleichzeitiger Bearbeitung (last-write-wins, bewusst akzeptiert — siehe Memory)
- Separate Tooltip-Bibliothek

## Decisions

### 1. Datenquelle: attendances API für Trainer

**Entscheidung:** Sobald `isTrainer === true`, wird `GET /attendances` als einzige Quelle für die Tabelle genutzt — inklusive des RSVP-Status. `session.responses` wird für Trainer nicht mehr separat verarbeitet.

**Begründung:** Attendances gibt alle Teammitglieder zurück (auch ohne Rückmeldung), enthält `rsvp_status`, und vermeidet das Mergen zweier Listen. Für vergangene Sessions ist ohnehin schon ein Attendances-Call nötig; für zukünftige Sessions gibt er einfach `present: null` zurück.

**Konsequenz:** Trainer sehen immer alle Teammitglieder (auch ohne Rückmeldung). Anwesend-Checkbox erscheint nur bei `isPast`. No-RSVP-Badge im Header nur für Trainer (da nur sie die Gesamtzahl kennen).

### 2. Kommentar-Tooltip: CSS-only via Tailwind `group`

**Entscheidung:** Kein externes Tooltip-Package. Die RSVP-Zelle wird in ein `relative group`-Element gewrapped. Das Tooltip-Div hat `absolute hidden group-hover:block` Klassen. Auf Mobile ersetzen wir `group-hover:` durch einen `onClick`-State pro Zeile (`showReasonId: number | null`).

**Begründung:** Minimaler Code, keine Dependency. Das Pattern ist im Projekt bereits für Dropdown-Menüs bekannt. Mobile-Detection via CSS `@media (hover: none)` ist unzuverlässig; ein explizites Click-State ist robuster.

**Umsetzung im Detail:**
```
Desktop:  <div className="group relative">
            <MessageCircle w-3 h-3 />
            <div className="hidden group-hover:block absolute z-10 ...tooltip styles...">
              {reason}
            </div>
          </div>

Mobile:   onClick={() => setShowReasonId(id === showReasonId ? null : id)}
          {showReasonId === row.member_id && <p className="text-xs ...>{reason}</p>}
```

### 3. Auto-save: sofortiger POST, kein Debounce

**Entscheidung:** Jeder Checkbox-Toggle feuert unmittelbar `POST /training-sessions/{id}/attendances` mit dem vollständigen aktuellen `attendanceMap`. Kein Debounce.

**Begründung:** Checkboxes werden von Menschen geklickt — Doppelklick in <300ms ist unwahrscheinlich. Sofortiger Save ist konzeptionell ehrlicher als verzögerter Save. Bei Fehler: lokale Checkbox zurücksetzen, Fehler-Banner im Kartenfuß zeigen (`attendanceError: string | null`).

### 4. Stat-Badges Farben

| Badge | Farbe | Tailwind |
|-------|-------|----------|
| ✓ N   | grün  | `bg-green-100 text-green-700` (ausnahmsweise, da kein brand-success Token vorhanden) |
| ✗ N   | rot   | `bg-brand-danger-light text-brand-danger` |
| ? N   | grau  | `bg-brand-border-subtle text-brand-text-muted` |
| – N   | grau  | `bg-brand-border-subtle text-brand-text-muted` (nur Trainer) |

## Risks / Trade-offs

- **useLiveUpdates + Auto-save**: Nach jedem POST /attendances broadcastet das Backend `"trainings"`. Das löst `load()` aus (Session-Daten), aber nicht `loadAttendances()`. Damit beeinflussen eigene Saves den Checkbox-State nicht. Fremde Saves sind erst nach manuellem Reload sichtbar (acceptable).
- **Attendances-Call auch für zukünftige Sessions**: Minimal mehr Load, vernachlässigbar.
