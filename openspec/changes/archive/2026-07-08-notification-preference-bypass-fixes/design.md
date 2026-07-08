## Context

Ergebnis der Durchsicht der 6 `push.SendToUsers`-Bypass-Sites (aus `notification-test-coverage`). Entscheidungsraster: existiert eine passende Kategorie / filtern Geschwister-Calls / ist es kritisch (Datenverlust) / breite soziale vs. Funktionärs-Zustellung.

## Goals / Non-Goals

**Goals:**
- Nutzer-Kontrolle für die 5 nicht-kritischen Bypass-Sites (via bestehende `carpooling` bzw. zwei neue Kategorien).
- Datenverlust-Warnung (#3) bleibt garantiert zustellbar.
- Bestehende Präferenzen verlustfrei migrieren.

**Non-Goals:**
- Keine E-Mail für `operativ`/`sonstiges` (reine Push-Pfade; kein `notify.Send`).
- Keine Änderung an #3.
- Keine feingranulare Unterscheidung innerhalb `operativ` (ein Toggle für alle Funktionärs-Reminder).

## Decisions

**D1 — Zwei neue Kategorien statt einer.**
`operativ` (Funktionärs-Reminder #1/#2/#6) und `sonstiges` (#4 video-ready) adressieren unterschiedliche Zielgruppen (Funktionäre vs. breites Team) und sollen getrennt schaltbar sein. `#5` nutzt die **bestehende** `carpooling`-Kategorie (kein neuer Toggle) — der Fix ist reine Konsistenz mit den Geschwister-Calls.

**D2 — Filter am Punkt der Empfänger-Ermittlung, vor dem notification_log.**
Bei den Scheduler-Sites (#1/#2) wird ein deaktivierter Empfänger **vor** dem `INSERT OR IGNORE notification_log` aussortiert (nicht nur der Push übersprungen). Damit bleibt ein später wieder aktivierter Nutzer für künftige Läufe erfassbar, statt durch einen „verbrauchten" Log-Eintrag dauerhaft ausgesperrt zu sein.

**D3 — #3 bleibt hart.**
`video-retention-warning` ist eine Datenverlust-Warnung (Video wird in 7 Tagen gelöscht) — Klasse Billing-/Security-Alert. Kein Opt-out, keine Kategorie. Im Code + Test als bewusst dokumentiert.

**D4 — UI: Beschreibungen ergänzen.**
`ProfileMiscTab` bekommt eine `categoryDescriptions`-Map; die neue Zeile rendert Label + Kurzbeschreibung. `operativ`/`sonstiges` sind Push-only — der E-Mail-Toggle bleibt aus Konsistenz sichtbar, ist aber ohne Wirkung (kein E-Mail-Pfad); alternativ nur Push-Spalte. Gewählt: Push-only-Darstellung, um Nutzer nicht mit wirkungslosen E-Mail-Toggles zu verwirren.

## Risks / Trade-offs

- **[Weniger Zustellungen bei aktiviertem Opt-out]** → gewollt; Default bleibt an, Bestandsnutzer ohne Zeile sind unverändert „an".
- **[Funktionäre schalten `operativ` ab und verpassen Aufgaben]** → akzeptiert: Opt-out ist bewusst; Default an. #3 (kritisch) ist ausgenommen.
- **[Migration bricht bei Fehler]** → Rebuild in Transaktion, `INSERT … SELECT` 1:1, Down-Pfad entfernt nur die zwei neuen Kategorie-Zeilen.
- **[Pinning-Tests müssen invertiert werden]** → bewusst: die 5 gedrehten Tests belegen jetzt die neue Invariante; #3 behält den Bypass-Test.

## Migration Plan

1. Migration 027 (Rebuild, CHECK + `operativ`,`sonstiges`).
2. `push.ValidCategories` erweitern.
3. Fünf Sites `FilterByPushPref` ergänzen (#3 unangetastet).
4. UI-Toggles.
5. Tests drehen + Positiv-Fall.
6. Gate; Deploy führt `migrate up` aus. Rollback: `migrate down` (027) + Vorgänger-Binary.

## Open Questions

- Später evtl. eine feinere Aufspaltung von `operativ` (Anwesenheit vs. Spielbericht) — bewusst offen, YAGNI.
