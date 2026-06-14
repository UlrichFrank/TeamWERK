## Context

Das Berechtigungsmodell in TeamWERK unterscheidet zwischen JWT-`role` (admin/trainer/elternteil/spieler) und `club_functions` (vorstand/trainer/sportliche_leitung/etc.). Router-Middleware nutzt `RequireClubFunction()`, nicht `RequireRole()`, für Vereinsfunktions-Gruppen.

Beim Aufbau der vorstand-Gruppe wurden die CRUD-Endpunkte für Duty-Types und Duty-Templates als eine Einheit registriert. Die Lese-Operationen (GET) hätten aber in eine breitere Gruppe gehört, da Trainer diese Daten zum Ausführen ihrer erlaubten Schreib-Operationen (POST /api/duty-slots) zwingend benötigen.

## Goals / Non-Goals

**Goals:**
- Trainer und Sportliche Leiter können Dienst-Typen lesen (für Slot-Erstellung auf Termin-Detailseite)
- Trainer und Sportliche Leiter können Dienst-Templates lesen und Previews abrufen (für Kalender-Event-Wizard)
- `canEdit`-Check im Frontend erkennt Trainer korrekt über `club_functions`, nicht `role`

**Non-Goals:**
- Trainer bekommen keine Schreibrechte auf Duty-Types oder Duty-Templates
- Keine Änderung am Datenbankschema oder an API-Antwortstrukturen
- Keine neue Endpunkte

## Decisions

**Router-Umstrukturierung statt neuer Endpunkte**: Die 4 GET-Routen werden aus der vorstand-Gruppe herausgelöst und in die bestehende `vorstand + trainer + sportliche_leitung`-Gruppe (main.go:375-390) verschoben. Die vorstand-only Gruppe behält nur die Schreib-Routen. Das ist der minimale Eingriff ohne API-Versionierung oder neue Middleware.

**Frontend: hasFunction statt role-Vergleich**: `SpieltagDetailPage` nutzt `user.role === 'trainer'` — das ist falsch, weil `role` die System-Rolle ist (spieler/elternteil/trainer/admin), nicht die Vereinsfunktion. Korrekt ist `hasFunction(user, 'trainer')` aus AuthContext, das `club_functions` im JWT prüft. Dieser Fix verhindert auch, dass zukünftige Nutzer mit `role=spieler + club_function=trainer` kaputte Ansichten erleben.

## Risks / Trade-offs

[Kein Risiko] Duty-Types und Templates enthalten keine sensiblen Daten (nur Namen, Zeiten, Konfiguration). Lese-Zugriff für Trainer ist aus fachlicher Sicht eindeutig korrekt.

[Minimales Risiko] Falls weitere Seiten ähnliche falsche role-Checks haben, sind diese vom Fix nicht erfasst. → Scope ist bewusst auf SpieltagDetailPage beschränkt; CLAUDE.md enthält ohnehin die Konvention `hasFunction()` zu nutzen.

## Migration Plan

1. Backend-Änderung deployen (main.go Router)
2. Frontend-Build deployen (SpieltagDetailPage.tsx)
3. Kein Datenbankeingriff, kein Rollback-Risiko
4. Bestehende Trainer-Sessions funktionieren sofort nach Deploy (JWT enthält bereits club_functions)
