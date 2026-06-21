## Designentscheidungen

### 1. `cross_team_visible` auf `members`, nicht auf `user_visibility`

- Was gefiltert wird, ist eine **Member**-bezogene Anzeige (Name + RSVP-Status in Teilnehmerlisten).
- Kinder ohne eigenen Account haben kein `user_visibility`, aber sehr wohl ein `members`-Datensatz — die Privacy-Entscheidung muss dort hängen.
- Eltern stellen über den Familienzugang auf das Kind-Member-Profil ein; keine Sondermechanik nötig.

### 2. Direkter Save, kein Draft-Workflow

- DSGVO-Einwilligungen laufen über Drafts, weil der Verein juristisch dokumentieren muss, wer was wann erlaubt hat.
- `cross_team_visible` ist eine persönliche Anzeige-Präferenz ohne rechtliche Konsequenz für den Verein → direkter `PUT`, sofortige Wirkung.

### 3. Default `0` für Bestandsmitglieder

- Migration setzt `DEFAULT 0`. Bestehende Mitglieder fangen damit privat an.
- Alternative wäre, beim ersten Deploy einen `UPDATE members SET cross_team_visible=1` zu fahren (aufwärtskompatibel), wurde aber bewusst verworfen: das Feature ist genau gedacht als Privacy-Schutz, der per Default greift.
- Konsequenz: Direkt nach Deploy schrumpfen Multi-Team-Sektionen. In `RsvpConfigBadges` o.ä. ggf. ein einmaliger Hinweistext für betroffene Spieler (optional, **nicht** Teil dieses Proposals).

### 4. "Meine Teams im Event" inkludiert Kinder

- Definition: Die Menge aller Teams T, für die ein Member existiert, das entweder (a) der eingeloggte Nutzer selbst ist oder (b) ein Kind des eingeloggten Nutzers (über `family_links`), und in `kader_members` oder `kader_extended_members` für ein Kader steht, das `season_id = games.season_id` und `team_id` in `game_teams(game_id)` hat.
- SQL-Skizze:
  ```sql
  WITH my_member_ids AS (
    SELECT m.id FROM members m WHERE m.user_id = :caller_user_id
    UNION
    SELECT fl.child_member_id FROM family_links fl WHERE fl.parent_user_id = :caller_user_id
  ),
  my_teams_in_event AS (
    SELECT DISTINCT k.team_id
    FROM kader k
    LEFT JOIN kader_members km ON km.kader_id = k.id
    LEFT JOIN kader_extended_members kem ON kem.kader_id = k.id
    WHERE k.season_id = (SELECT season_id FROM games WHERE id = :game_id)
      AND k.team_id IN (SELECT team_id FROM game_teams WHERE game_id = :game_id)
      AND (km.member_id IN (SELECT id FROM my_member_ids)
           OR kem.member_id IN (SELECT id FROM my_member_ids))
  )
  ```

### 5. Counter über sichtbare Zeilen

- Trainer-Counter ("✓ 3 / ✗ 1") und Spieler-Counter divergieren bei Multi-Team-Events.
- Das ist gewollt: der Spieler-Counter ist die Aggregation **dessen, was er sieht**. Alles andere wäre verwirrend (Lücke zwischen Liste und Zahl) oder leakt Information.
- Keine UI-Erklärung dafür — eine `?`-Hilfe würde mehr Verwirrung stiften als beantworten.

### 6. „Weitere Mitglieder nicht sichtbar" ohne Zahl

- Variante mit Zahl ("4 weitere Mitglieder …") leakt die Größe des fremden Kaders.
- Ohne Zahl genügt der Hinweis, dass die Liste **gefiltert** ist; die genaue Größe bleibt privat.
- Hinweis-Style: Footer pro Sektion, klein, `text-brand-text-muted text-xs`, kein Icon.

### 7. Single-Team-Events unverändert

- Für Trainings (per Definition single-team) und für Heim-/Auswärtsspiele mit nur einem Team ändert sich nichts. Der Filter greift erst, wenn `game_teams` mehr als einen Eintrag hat.

### 8. Funktionsbasierter Bypass

- `admin`, `trainer`, `sportliche_leitung`, `vorstand` umgehen den Filter komplett. Begründung:
  - Trainer/sL: brauchen die Aufstellung (`in_lineup`-Checkbox).
  - Vorstand: organisatorische Übersicht.
  - Admin: per Definition.
- `kassierer` und `vorstand_beisitzer` bewusst **nicht** im Bypass — sie haben keinen Bedarf an Multi-Team-Teilnehmerlisten.

## Offene Fragen

Keine bekannten — alle in der Exploration geklärt.
