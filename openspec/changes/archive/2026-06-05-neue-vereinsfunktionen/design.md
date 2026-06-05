## Context

TeamWERK verwendet ein zweistufiges Berechtigungsmodell:
1. `users.role` (admin | standard) — System-Zugangsstufe, im JWT als `role`
2. `member_club_functions` — fachliche Vereinsfunktionen in einer Junction-Table, im JWT als `club_functions[]`

Zugriffskontrolle im Router erfolgt über zwei Middleware-Funktionen:
- `auth.RequireRole("admin")` — systemweite Adminrouten
- `auth.RequireClubFunction("trainer", "vorstand", ...)` — fachliche Routen, prüft gegen `claims.ClubFunctions`

Team-Filterung für Trainer passiert in Handlern (nicht in Middleware): `HasFunction("trainer")` → WHERE-Clause auf eigene Kader-Teams.

## Goals / Non-Goals

**Goals:**
- `kassierer` und `sportliche_leitung` als gültige Werte in `member_club_functions` einführen
- `sportliche_leitung` erhält trainer-äquivalenten Zugang (Kader, Spielplan, Dienste)
- `sportliche_leitung` sieht alle Kader-Teams ohne Team-Einschränkung
- Beide Funktionen in Frontend-Dropdowns und Label-Maps

**Non-Goals:**
- Eigene API-Routen für `kassierer` (kommt später)
- Änderung der Mitglieder-Sichtbarkeit für `trainer` (bleibt wie bisher)
- `sportliche_leitung` bekommt keinen Mitglieder-Zugang

## Decisions

### Entscheidung: `IsTrainerLike()` statt `AllTeams bool` im JWT

**Gewählt:** Neue Methode `func (c *Claims) IsTrainerLike() bool` in `auth/tokens.go`, die `HasFunction("trainer") || HasFunction("sportliche_leitung")` zurückgibt. Da `club_functions` bereits vollständig im JWT steckt, ist ein zusätzliches `all_teams`-Flag redundant.

**Abgelehnt:** Neues `AllTeams bool`-Feld im JWT — führt zu zwei Quellen der Wahrheit und erfordert Migration der Token-Ausstellungslogik ohne Mehrwert.

**Team-Filter-Pattern:**
```go
// Vorher
if claims.HasFunction("trainer") { /* nur eigene Teams */ }

// Nachher
if claims.IsTrainerLike() && !claims.HasFunction("sportliche_leitung") {
    /* nur eigene Teams */
}
```

### Entscheidung: Migration via neuer Migrations-Datei

SQLite unterstützt kein `ALTER TABLE ... ALTER COLUMN` für CHECK-Constraints. Die Tabelle `member_club_functions` muss neu erstellt werden (DROP + CREATE + Daten-Copy). Standard-Muster wie in Migration 002.

Da `member_club_functions` keine eingebetteten Foreign-Key-Verweise von anderen Tabellen hat (nur `member_id` → `members`), ist die Neuerstellung risikoarm.

### Entscheidung: `kassierer` ohne Routing-Änderungen

`kassierer` bekommt vorerst nur den Eintrag im CHECK-Constraint und die Frontend-Labels. Keine `RequireClubFunction`-Änderungen, keine neuen Handler. Die Funktion ist damit für Admins assignierbar und im Profil sichtbar — API-Zugänge folgen in einer separaten Änderung.

## Risks / Trade-offs

**[Risiko] SQLite CHECK-Constraint-Migration** → Mitigation: Standard-Muster (Tabelle umbenennen, neu anlegen, Daten kopieren) — bereits in Migration 002 erfolgreich eingesetzt. Down-Migration löscht `kassierer`/`sportliche_leitung` Zeilen vor Tabellen-Umbau.

**[Risiko] Vergessene `HasFunction("trainer")`-Checks in zukünftigen Features** → Mitigation: `IsTrainerLike()` als kanonische Methode etablieren; bei neuen trainer-gated Features konsequent diese Methode verwenden.

**[Trade-off] `sportliche_leitung` ohne Mitglieder-Zugang** → Bewusste Entscheidung: Mitglieder-Vollzugriff ist `vorstand`-Domäne. Falls die Sportliche Leitung Mitglieder-Lesezugang braucht, ist das eine eigene Änderung.

## Migration Plan

1. Migration `0NN_neue_vereinsfunktionen.up.sql`: `member_club_functions` neu erstellen mit erweitertem CHECK
2. Binary deployen (Migration läuft automatisch bei Start via `make deploy`)
3. Keine Datenmigration nötig — bestehende Zeilen bleiben unverändert
4. Rollback: `.down.sql` löscht Zeilen mit neuen Funktionswerten vor Tabellen-Umbau
