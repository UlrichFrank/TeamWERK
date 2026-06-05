## Context

Der Kalender (`/kalender`) zeigt heute Spieltage und Trainings in einer Monatsansicht. Klicks auf Spieltage navigieren zu `/kalender/:id` (SpieltagDetailPage mit Dienst-Slots), Klicks auf Trainings öffnen `TrainingEditModal` (Trainer/Admin) oder leiten nach `/termine` weiter (andere Rollen). Es gibt keinen expliziten Modus-Wechsel — der Nutzer kann nicht zwischen "Ich will Dienste planen" und "Ich will Termine verwalten" umschalten.

Bestehendes Pattern für einen solchen Wechsler: `/mitfahrgelegenheiten` hat einen "Team | Meine"-Toggle oben rechts (gleiche CSS-Klassen werden wiederverwendet).

## Goals / Non-Goals

**Goals:**
- Modus-Toggle `[Dienste | Termine]` oben rechts in KalenderPage, visuell identisch mit dem Mitfahrgelegenheiten-Toggle
- Dienste-Modus: Spieltag-Klick → SpieltagDetailPage; Training-Klick → keine Aktion
- Termine-Modus: Spieltag-Klick → `GameEditModal`; Training-Klick → `TrainingEditModal` (Berechtigt) / `EventInfoModal` (andere)
- Neues `GameEditModal` (analog `TrainingEditModal`) mit `PUT /api/admin/games/{id}`
- Neues `EventInfoModal` (schreibgeschützt) für Spieler/Elternteile

**Non-Goals:**
- Keine Navigation zwischen `/kalender` und `/mitfahrgelegenheiten` über den Toggle
- Kein neuer Event-Typ (Fest/Veranstaltung)
- Keine Persistenz des gewählten Modus über Sitzungen hinaus
- Keine Backend-Änderungen oder Migrationen

## Decisions

### Toggle als lokaler State, kein URL-Parameter
Der `kalenderMode`-State lebt als `useState` in KalenderPage. Kein `?mode=` Query-Parameter, kein `localStorage`.

**Alternativen:** URL-Parameter würde Deep-Linking ermöglichen, ist aber Overkill für diesen Use-Case. localStorage würde den Stand sitzungsübergreifend merken — wurde explizit als Non-Goal definiert.

### GameEditModal als eigenständiges Komponent
`GameEditModal` wird als eigene Datei unter `web/src/components/GameEditModal.tsx` implementiert, analog zu `TrainingEditModal`. Nicht inline in KalenderPage.

**Begründung:** Trennung von Darstellung (KalenderPage) und Formular-Logik (Modal). Einfacher testbar und wiederverwendbar (z.B. in SpieltagDetailPage).

### Rollenprüfung im Click-Handler, nicht im Modal
Der Click-Handler in KalenderPage prüft die Rolle und öffnet entweder `GameEditModal` oder `EventInfoModal`. Das Modal selbst enthält keine Rollenlogik.

**Begründung:** Konsistent mit dem heutigen Training-Click-Handler, der dieselbe Weiche hat.

### Dienste-Modus: Training-Klick als No-Op
Trainings erhalten im Dienste-Modus kein `onClick` und werden mit `cursor-default` gerendert (kein `hover:bg-*`-Effekt). Keine Tooltip-Erklärung.

**Begründung:** Trainings haben keine Dienst-Slots — im Dienste-Kontext sind sie irrelevant. Subtile visuelle Unterscheidung (kein Hover) ist ausreichend.

### EventInfoModal zeigt RSVP-Zahlen ohne eigenen API-Call
Die RSVP-Zählwerte (`confirmed_count`, `declined_count`, `maybe_count`) sind bereits im `Training`-Objekt vorhanden (vom `/training-sessions`-Aufruf). Kein weiterer Fetch nötig.

## Risks / Trade-offs

- **Verwirrung über Standard-Modus:** Dienste-Modus als Standard kann unintuitiv wirken für Nutzer ohne Admin/Trainer-Rechte (sie sehen keine Dienste-Details). → Akzeptiert; Trainings bleiben im Kalender sichtbar und die Toggle-Position ist prominent.
- **GameEditModal vs. Admin-Seite:** Das neue Modal erlaubt Trainer das Bearbeiten von Spieltagen direkt im Kalender — bisher nur im Admin-Bereich möglich. → Gewollt; kein doppelter State, da beide auf `PUT /api/admin/games/{id}` zeigen.
- **PUT /api/admin/games/{id} erlaubt trainer?** Muss vor Implementierung geprüft werden — lt. CLAUDE.md ist die Route nur "Admin only". Ggf. muss die Middleware angepasst werden auf `admin + trainer + vorstand`.

## Migration Plan

1. Keine DB-Migration nötig
2. Frontend-Only-Deployment via `make deploy`
3. Rollback: Vorherigen Commit deployen (kein State in DB)
