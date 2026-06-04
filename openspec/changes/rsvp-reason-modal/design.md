## Context

`TerminePage.tsx` zeigt Trainings- und Spieltermine und erlaubt RSVP (Zusagen/Vielleicht/Absagen). Die bisherige Implementierung hat zwei Probleme:

1. Für "Vielleicht" und "Absagen" existiert ein Inline-`<input>`-Feld unter den Buttons, das aber auf `reasons[key]` zugreift — einem State, der nie deklariert wurde. Begründungen werden daher nie übermittelt.
2. Das UX-Muster (Feld unter Buttons → dann klicken) lädt dazu ein, die Begründung wegzulassen.

Das Backend akzeptiert `reason` bereits im Request-Body — es braucht keine Backend-Änderungen.

## Goals / Non-Goals

**Goals:**
- Begründung bei "Vielleicht" und "Absagen" ist Pflichtfeld und wird zuverlässig übermittelt
- Modal-Muster: Nutzer muss aktiv bestätigen, bevor die RSVP gesendet wird
- Abbrechen lässt den bisherigen RSVP-Status unverändert
- Gilt für eigene RSVPs und Eltern-Kind-RSVPs, für Training und Spiele

**Non-Goals:**
- Keine Begründung für "Zusagen" (kein Bedarf)
- Kein Backend-Umbau
- Kein Refactoring der Kartenstruktur jenseits der RSVP-Buttons

## Decisions

### Modal statt Inline-Feld

**Entscheidung:** Ein einzelnes, seitenweites Modal wird beim Klick auf "Vielleicht"/"Absagen" geöffnet.

**Begründung:** Ein Inline-Feld unterhalb der Buttons ist optisch leicht zu übersehen und trennt Eingabe von Aktion. Ein Modal erzwingt den Fokus auf die Begründung und macht die Pflichtfeld-Semantik deutlich. Da immer nur ein Modal gleichzeitig offen sein kann, reicht ein einziger `pendingRSVP`-State für die gesamte Seite.

**Alternativen:** Inline-Feld reparieren und als Pflichtfeld markieren → schlechtere UX, unklare Beziehung zwischen Feld und Button.

### Bestehende `pendingRSVP`-State-Variable weiterverwenden

**Entscheidung:** `pendingRSVP` (bereits deklariert, Typ `{ kind, id, status, memberId? }`) und `modalReason` (ebenfalls deklariert) werden direkt verwendet. Kein neuer State.

**Begründung:** Der State ist bereits vorhanden; das spart Refactoring und zeigt, dass die ursprüngliche Intention richtig war — nur die Modal-UI fehlt.

### OK-Button disabled bis Begründung nicht leer

**Entscheidung:** `disabled={modalReason.trim() === ''}` auf dem OK-Button.

**Begründung:** Einfachste Implementierung eines Pflichtfelds ohne Validierungsbibliothek. Kein separater `submitted`-State nötig.

### Keine separate Komponenten-Datei

**Entscheidung:** Das Modal wird als JSX direkt am Ende von `TerminePage.tsx` gerendert.

**Begründung:** Das Modal ist ausschließlich an den lokalen State der Seite gebunden (`pendingRSVP`, `modalReason`, Dispatch-Funktionen). Eine Auslagerung in eine eigene Datei würde nur Prop-Drilling erzeugen ohne Mehrwert.

## Risks / Trade-offs

- **Begründung verpflichtend auch für "Vielleicht"** → Könnte für Nutzer ungewohnt sein. Trade-off: Konsistenz der UX und Planungssicherheit für Trainer überwiegen.
- **Modal blockiert Seiten-Scrolling** → Mitigiert durch Standard-Overlay-Pattern (`fixed inset-0 z-50`), das bereits im Projekt für andere Modals verwendet wird.
