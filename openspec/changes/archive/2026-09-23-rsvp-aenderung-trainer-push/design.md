# Design

## Context

Beide RSVP-Handler (`games.RespondToGame`, `trainings.Respond`) machen heute: Autorisierung → Absence-Lock-Check (liest `absence_id` der bestehenden Antwort) → ggf. Serien-Abmeldung → Cutoff → `INSERT … ON CONFLICT DO UPDATE` → Broadcast → 204. Der vorherige Status wird nirgends gelesen; der Upsert überschreibt ihn. Benachrichtigungen laufen ausschließlich über `notify.Send` (Event-Log + Präferenzfilter + Push/Mail im Hintergrund). Die Trainer eines Kaders stehen in `kader_trainers` (Mitglieds-IDs), ihre Konten über `members.user_id`.

## Goals / Non-Goals

**Goals:**
- Trainer erfahren kurzfristige Umentscheidungen (≤ 7 Tage) ohne Polling.
- Keine Änderung am HTTP-Vertrag der RSVP-Routen; eine fehlschlagende Meldung kippt die RSVP nie.

**Non-Goals:**
- **Abwesenheiten** (`internal/absences`), die RSVPs mit `absence_id` automatisch auf `declined` setzen. Eine Abwesenheit kann Dutzende Termine auf einmal betreffen — ein eigener, gebündelter Meldungsweg („X ist vom … bis … abwesend, betrifft 3 Termine dieser Woche") wäre die richtige Form und ist ein Folge-Change.
- Erst-Antworten, reine Grund-Änderungen, Meldungen an Eltern/Mitspieler.
- Debouncing/Coalescing mehrfacher Umentscheidungen.
- Konfigurierbares Fenster (7 Tage ist fest verdrahtet, als benannte Konstante).

## Decisions

1. **Vorherigen Status im selben Lookup wie den Absence-Lock lesen.** Der bestehende `SELECT absence_id …` wird zu `SELECT absence_id, status …` erweitert (`sql.ErrNoRows` = keine vorherige Antwort). Kein zusätzlicher Query, keine Transaktion nötig: ein paralleler Doppel-Request desselben Nutzers könnte schlimmstenfalls eine Meldung zu viel oder zu wenig erzeugen — akzeptabel für eine Informationsmeldung. *Alternative:* `RETURNING` beim Upsert liefert nur den neuen Wert, nicht den alten → verworfen.

2. **Fenster am Terminbeginn in Europe/Berlin, `0 < start − now ≤ 7×24h`.** Wiederverwendung der vorhandenen Parser (`parseBerlinDateTime` in trainings, die Logik hinter `gameLocksAt` in games) und `h.now()` für testbare Zeit. Die Bedingung „in der Zukunft" ist nötig, weil Staff den Cutoff umgehen darf (`CanOverrideRSVPCutoff`) und nachträgliche Korrekturen vergangener Termine keine Meldung auslösen sollen.

3. **Kader-Bezug: Saison des Termins, nicht aktive Saison.** Spiele: `kader k JOIN game_teams gt ON gt.team_id = k.team_id WHERE gt.game_id = ? AND k.season_id = games.season_id`. Trainings: direkt `training_sessions.kader_id` (deckt Übungsgruppen ab). Innerhalb von 7 Tagen fallen beide Varianten praktisch immer zusammen; die Saison des Termins ist aber die fachlich korrekte Frage „wer trainiert diesen Termin?".

4. **Trainer-Auflösung als Helfer in `internal/notify`** (`notify.KaderTrainers(db, kaderIDs ...int) []int`). Foundation, von beiden Domänen nutzbar, gleiche Fehlerpolitik wie `TeamAudience` (kein `error`, Query-Fehler per `slog.Error`). Er verbindet keine `family_links` und fällt damit nicht unter das Audience-Gate (`internal/arch/audience_test.go`); er ist bewusst **nicht** `TeamAudience`, weil nur die Trainer gemeint sind. Kader-IDs ermittelt die jeweilige Domäne selbst (Decision 3).

5. **Auslöser ausfiltern.** Anders als bei `TeamAudience` (Auslöser bekommt Bestätigung) ist hier die Handlung des Auslösers der *Inhalt* der Meldung an Dritte; ein Trainer, der selbst umsagt, weiß davon. Gefiltert wird im Handler nach `claims.UserID`.

6. **„Spieler" = Ziel-Mitglied ist nicht Trainer des betroffenen Kaders.** Trainer führen eigene RSVPs (`trainer-rsvp`); deren Änderung ist keine Spieler-Umentscheidung. Mitglieder des erweiterten Kaders und Förderkinder zählen als Spieler. Prüfung über dieselbe Kader-ID-Menge: `EXISTS kader_trainers WHERE kader_id IN (…) AND member_id = ?`.

7. **Kategorie `operativ`.** Die Meldung richtet sich an Funktionsträger, nicht an die Mannschaft. Wer als Trainer `games`/`trainings` stummschaltet, um Termin-Rundmeldungen der eigenen Mannschaft zu vermeiden, soll diese Planungsinfo trotzdem bekommen; wer sie nicht will, schaltet `operativ` ab. `operativ` ist im `user_events`-CHECK und in `push.ValidCategories` bereits vorhanden → keine Migration. Im Profil heißt der Schalter „Vereinsaufgaben“; seine Beschreibung sprach bisher nur von Erinnerungen (Anwesenheit, Spielberichte) und wird um die Umentscheidungen erweitert — sonst findet ein Trainer, den die Meldung stört, den zuständigen Schalter nicht. *Alternative:* `games`/`trainings` — verworfen, weil dann die Stummschaltung der Mannschafts-Rundmeldungen diese Meldung mitnähme.

8. **Text über einen Baustein in `notify`** (`notify.RSVPChangeBody(member, subject, when, from, to, reason string)`), Status-Labels „Zusage"/„Absage"/„Vielleicht". Titel: „Umentscheidung: <Name>". `when` über `notify.EventWhen`, Grund über `notify.TrimReason`. Beispiel: „Max Muster hat für Spiel gegen TV Beispiel am Sa, 27.09.2026 um 15:00 von Zusage auf Absage umgestellt. Grund: krank".

9. **Aufruf nach dem Upsert, vor dem 204**, über `notify.SendAsync`: die Empfänger- und Namensauflösung ist synchron billig, die Zustellung läuft ohnehin im Hintergrund; `SendAsync` hält den Request frei und fängt Panics.

## Risks / Trade-offs

- [Spieler schaltet mehrfach hin und her → mehrere Pushes] → selten; bewusst kein Debouncing. Falls es stört, analog `pending_event_notes_push` nachrüsten.
- [Abwesenheiten bleiben stumm] → explizit Non-Goal, Folge-Change notiert.
- [Race zweier paralleler Requests desselben Mitglieds] → max. eine Meldung zu viel/zu wenig; keine Datenfolgen.
- [Test mit In-Memory-SQLite und `SendAsync`] → Goroutine zieht eine zweite Verbindung; Tests nutzen `testutil.NewDB` (Datei-DB) und fangen die Meldung über die `notify.Send`-Paketvariable ab.

## Migration Plan

Kein Schema, keine Konfiguration. Deploy wie üblich; Rollback per `make deploy-rollback` ohne Folgen.
