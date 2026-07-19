## Context

`Respond` (`internal/trainings/handler.go`) und `RespondToGame` (`internal/games/handler.go`) bestimmen das Ziel-`member_id` über:

```go
switch claims.Role {
case "spieler":   // toter Zweig — Rolle existiert nicht
case "elternteil": // toter Zweig — prüft parentHasChild, wird nie erreicht
default:           // jeder Standard-User: memberID = req.MemberID, KEINE Prüfung
}
```

`users.role` ist per CHECK auf `admin|standard` beschränkt; Eltern/Spieler werden über `claims.IsParent` (aus `family_links`) bzw. Vereinsfunktionen modelliert. Die `case`-Werte greifen also nie → `default` ist der reale Pfad → beliebige `member_id` wird ungeprüft akzeptiert.

`game`-Variante hat zusätzlich ein `UserCanSeeGame`-Gate (Sichtbarkeit), aber innerhalb sichtbarer Spiele bleibt die Member-Ownership ungeprüft. `training`-Variante hat gar kein Gate über die Authentifizierung hinaus.

## Goals / Non-Goals

**Goals:**
- RSVP-Schreibzugriff rollenmodell-korrekt durchsetzen: eigene Person + eigene Kinder; Manage-Berechtigte für alle.
- Den `eltern-rsvp`-Spec-Anspruch (403 für fremdes Kind) tatsächlich erfüllen.
- Beide Endpunkte (Training + Spiel) identisch absichern.

**Non-Goals:**
- Keine Änderung am Sichtbarkeits-/Anzeige-Pfad (`children_rsvp`, `GetAttendances`).
- Keine Einführung neuer Rollenwerte; das zweidimensionale Modell (System-Rolle + Vereinsfunktion + `IsParent`) bleibt.
- Keine Migration, kein API-Vertragsfeld.

## Decisions

**1. Toten `switch claims.Role` durch ownership-/capability-basierte Prüfung ersetzen.**
Neue Logik (für beide Handler identisch):
1. `own := memberIDForUser(claims.UserID)`.
2. Ziel bestimmen: `target := req.MemberID; if target == 0 { target = own }`. Ist `target == 0` weiterhin → 400/422 (kein Member auflösbar) wie bisher.
3. Autorisierung, falls `target != own`:
   - **erlaubt**, wenn Caller Manage-Rechte für das Event-Team hat (admin · `hasTeamAccess`/trainer-like/vorstand des Teams) — gleiche Helfer wie bei `SaveAttendances`/`canRecordGameAttendance`.
   - sonst **erlaubt**, wenn `parentHasChild(claims.UserID, target)`.
   - sonst **403**.
4. `target == own` ist immer erlaubt.

_Warum so:_ nutzt vorhandene Bausteine (`memberIDForUser`, `parentHasChild`, Team-Access-Checks), entfernt toten Code, ein einziger kohärenter Pfad statt rollenbasierter Sonderfälle.

**2. Gemeinsame Semantik, lokale Implementierung pro Package.**
Kein neues geteiltes Package (Architektur-Test verbietet Domain-Querimporte). Die Logik wird in jedem Handler-Package als kleine Hilfsfunktion umgesetzt (geringe Duplikation, klar testbar) — bewusst keine verfrühte Abstraktion.

## Risks / Trade-offs

- **Risk:** Legit-Flow eines Spielers ohne eigenes Member-Record bricht → **Mitigation:** `target==0 && own==0` liefert wie bisher 400/422; verhaltensgleich.
- **Risk:** Trainer/Vorstand, die für Teammitglieder antworten, werden fälschlich geblockt → **Mitigation:** Manage-Zweig nutzt exakt die bestehenden Team-Access-Checks; Tests decken den Fall ab.
- **Trade-off:** Logik in zwei Packages dupliziert (Training/Spiel) — akzeptiert, da Architektur-Test Domain-Querimporte untersagt und die Funktion klein ist.

## Migration Plan

Reiner Code-Change, keine DB-Migration. Verhaltensänderung: bisher (versehentlich) erlaubte Fremd-Rückmeldungen liefern künftig 403. Kein Rollback-Datenrisiko.

## Open Questions

Keine — die Regel (eigene Person + eigene Kinder; Manage ausgenommen) ist eindeutig und durch die bestehende `eltern-rsvp`-Spec gedeckt.
