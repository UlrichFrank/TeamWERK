## Designentscheidungen

### 1. Zentrales Visibility-Modul, nicht Copy-paste pro Handler

- `internal/auth/event_visibility.go`:
  ```go
  // UserCanSeeGame liefert true, wenn der User das Game in irgendeiner Liste/Detail sehen darf.
  func UserCanSeeGame(ctx context.Context, db *sql.DB, userID, gameID int) (bool, error)

  // GameIDsVisibleToUser liefert das Set aller game_ids in der Saison, die der User sehen darf.
  // Für Funktionsträger: nil (= kein Filter; aufrufender Handler muss das interpretieren).
  func GameIDsVisibleToUser(ctx context.Context, db *sql.DB, userID, seasonID int) (visibleIDs []int, unrestricted bool, err error)
  ```
- `unrestricted=true` signalisiert „Funktionsträger, kein WHERE-Filter nötig", damit Handler ihre bestehende Query ohne `IN (...)`-Klausel laufen lassen.

### 2. 404 statt 403

- Wir liefern bewusst **404 Not Found** statt 403 Forbidden — das verrät nicht, ob das Game existiert. Konsistenz mit dem Prinzip "Privacy-by-default": die Existenz eines Events fremder Teams ist selbst eine Information.
- Ausnahme: Funktionsträger sehen 200 (kein Filter).

### 3. Funktionsträger-Bypass identisch zu `profile-cross-team-visibility`

- `admin`, `trainer`, `sportliche_leitung`, `vorstand` umgehen den Filter.
- Begründung: Trainer planen Doppel-Veranstaltungen, sL koordiniert Mannschaftsführer, Vorstand braucht Übersicht.
- `kassierer` und `vorstand_beisitzer` bewusst NICHT — sie haben keinen operativen Bedarf an Event-Sichtbarkeit fremder Teams. Wenn sich Wunsch ergibt, später nachziehen.

### 4. Push-Empfänger über denselben Helper

- Heute werden Empfänger an verschiedenen Stellen via `kader_members`-Joins zusammengetragen.
- Zukünftig: Wer ein Push für Game E bekommen kann, ist `users mit UserCanSeeGame(u, E) = true`. Damit Push und API-Sichtbarkeit garantiert synchron sind.
- Funktionsträger erhalten Push nur, wenn das Push **inhaltlich** an sie gerichtet ist (Aufstellung geändert, Spiel abgesagt, …) — die bestehenden inhaltlichen Filter (Trainer/sL bei Aufstellung etc.) bleiben unberührt. Der neue Sichtbarkeitsfilter ist eine ZUSÄTZLICHE Whitelist-Bedingung, kein Replacement.

### 5. Saison-Scope für die Listenoperation

- `GET /api/games` läuft heute pro Saison. `GameIDsVisibleToUser(userID, seasonID)` liefert die Whitelist genau für diese Saison.
- Aufruf-Pattern:
  ```go
  ids, unrestricted, err := auth.GameIDsVisibleToUser(ctx, h.db, userID, seasonID)
  if !unrestricted {
      query += ` AND g.id IN (` + intsToPlaceholders(ids) + `)`
      args = append(args, intsToAnySlice(ids)...)
  }
  ```
- Bei leerer Whitelist (`len(ids)==0`, `unrestricted=false`): Query gar nicht erst laufen lassen, leeres Result zurückgeben.

### 6. Trainings unverändert

- `training_sessions.team_id` ist Single-Team. Bestehende Filter passen bereits.
- Sollte später ein Multi-Team-Training-Modell eingeführt werden, ist diese Capability die Stelle, an der das nachzuziehen ist.

### 7. Bestandstests anpassen

- Tests, die heute Games als "Standard-Nutzer" abrufen, brauchen eine Mitgliedschaft im Team des Test-Games. Fixture-Helper `CreateMembership(userID, teamID, seasonID)` ggf. anlegen oder bestehende `CreateMember`/`CreateKader`-Kette nutzen.
- Aufwand: pro Test 1–2 Zeilen, aber breit gestreut (~20–30 Stellen).

### 8. Mitfahrgelegenheiten

- Bisherige Existenz-Check (`SELECT 1 FROM games WHERE id=?`) wird zu `UserCanSeeGame`-Check.
- Schon vorhandene Mitfahr-Einträge auf nicht-mehr-sichtbaren Games werden NICHT gelöscht; sie sind nur nicht mehr lesbar für den nicht-berechtigten Nutzer. Reine Sichtbarkeitsänderung, kein Datendrop.

## Offene Fragen

- **OF-1:** Soll der Filter auf historische Events (Vergangenheit) ebenfalls greifen? Argument dafür: Konsistenz. Argument dagegen: Mitfahr-Statistiken oder Anwesenheits-Auswertungen könnten betroffen sein. **Vorschlag:** Ja, gleiche Regel — historische Privacy ist genauso wertvoll.
- **OF-2:** Bestehende Push-Notifications, die heute breit gestreut werden (z.B. „Spiel abgesagt" an alle Kader+Erweitert): müssen wir die Empfängerliste explizit kürzen, oder ist bereits sichergestellt, dass die heutigen Empfänger ohnehin Schnittmenge zum sichtbaren Personenkreis sind? **Vorschlag:** Analyse im ersten Task; kein automatischer Trust.
