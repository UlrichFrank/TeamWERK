## Kontext

`internal/carpooling/handler.go` ruft heute beim Anlegen einer neuen Suche `notifyOpposite(...)` auf — Push an alle `biete`-User des Spiels. Reichweite ist zu klein: in Jugendmannschaften fahren primär Eltern, die selten vorab ein Angebot eingestellt haben.

## Entscheidungen

### Trigger nur auf Insert, nicht auf Update

Der Handler unterscheidet bereits via `isNewEntry`. Ein Update bedeutet typischerweise Korrektur von Notiz/Treffpunkt — das ist keine neue Information für den Team-Kreis. Andernfalls würde jedes Edit den Team-Push erneut auslösen.

### „Nächstes Spiel" als Spam-Bremse

Server-seitig per Query:

```sql
SELECT g2.id
FROM games g2
JOIN game_teams gt ON gt.game_id = g2.id
WHERE gt.team_id = ?
  AND g2.date >= date('now')
ORDER BY g2.date, g2.time
LIMIT 1
```

Wenn die Antwort die aktuelle `game_id` ist, qualifiziert das Team. Sonst nicht.

**Konsequenz:** Wer drei Spiele im Voraus eine Suche einstellt, löst keinen Team-Push aus — bewusst. Der Push soll **akut** wirken.

### `game_teams` ist m:n — pro Team einzeln prüfen

Ein Spiel kann mehreren Teams zugeordnet sein (z.B. zwei Mannschaften reisen gemeinsam). Pro assoziiertem Team wird die „nächstes-Spiel"-Prüfung **getrennt** geführt. Empfänger der qualifizierenden Teams werden vereinigt; nicht-qualifizierende Teams tragen nichts bei.

### Kein Fallback bei fehlendem Kader

Wenn `kader` für `(team_id, games.season_id)` nicht existiert, schweigt der Push. Ein Fallback auf alle `members.team_id`-Mitglieder oder gar alle User wäre laut und unscharf — der Push richtet sich gezielt an Eltern eines konkret nominierten Kreises. Ohne Kader gibt es diesen Kreis nicht.

### Empfänger-Auflösung

Pro qualifizierendem `kader_id`:

```sql
-- Eltern der Spieler (regulär + erweitert)
SELECT DISTINCT fl.parent_user_id
FROM family_links fl
WHERE fl.member_id IN (
    SELECT member_id FROM kader_members         WHERE kader_id = ?
    UNION
    SELECT member_id FROM kader_extended_members WHERE kader_id = ?
)

UNION

-- Trainer mit User-Account
SELECT u.id
FROM kader_trainers kt
JOIN members m ON m.id = kt.member_id
JOIN users   u ON u.id = m.user_id  -- members.user_id ist nullable
WHERE kt.kader_id = ?
```

Resultat ist ein `[]int user_id`; danach `excludeUserID` (Steller) abziehen und über mehrere Teams vereinigen (DISTINCT).

### Keine Dedup mit `notifyOpposite`

`notifyOpposite` läuft unverändert weiter. Ein Bieter, der gleichzeitig Trainer/Elternteil des Kaders ist, kann zwei Pushes erhalten. Der Erwartungswert ist klein (Bieter zu einem Spiel sind selten), die Kosten der Dedup-Logik (zweistufige Push-Liste mit Differenzbildung) sind höher als der Nutzen. Akzeptiert.

### Kategorie bleibt `"carpooling"`

Keine neue Kategorie `"carpooling_team_request"`. Begründung: Wer Carpooling-Pushes komplett abschaltet, will auch keinen Team-Push. Wer Carpooling-Pushes will, will den Team-Push umso mehr — er ist der hier ja gewünschte Verstärker.

### Push-Text

```
Titel: "Mitfahrgelegenheit"
Body : "{Name} sucht eine Mitfahrgelegenheit zu {opponent}, {Datum}"
URL  : "/mitfahrgelegenheiten"
```

Identisch zum bestehenden `notifyOpposite`-Text (`suche`-Variante) → kein zusätzlicher i18n-Aufwand, gleicher Tonfall.

## Risiken und Nichtziele

- **Risiko Doppel-Push** (siehe oben) — akzeptiert.
- **Risiko leeres Empfänger-Set bei Erwachsenenteams ohne `family_links` und ohne Trainer** — kein Push, kein Schaden. Der bestehende `notifyOpposite` greift weiter.
- **Nicht-Ziel:** Cross-Team-Pool (gleicher Tag + Venue) wie in `dashboard-offene-gesuche-cross-team`. Bewusst out of scope — kann separat ergänzt werden, sobald die Cross-Team-Logik im Backend etabliert ist.
- **Nicht-Ziel:** Eingrenzung auf Suchen, die *näher als X Tage* am Spiel liegen. „Nächstes Spiel des Teams" ist die einzige zeitliche Bremse.
- **Nicht-Ziel:** Frontend-Änderungen. Subscription läuft bereits über `usePushSubscription` in `AppShell.tsx`; Anzeige in `/mitfahrgelegenheiten` unverändert.
