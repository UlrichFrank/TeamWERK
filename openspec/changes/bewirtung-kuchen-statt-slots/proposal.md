## Why

Die Bewirtungsrotation verteilt heute **Slots pro Spiel** statt **Kuchen pro Spieltag**. Bei
fünf Heimspielen entstehen fünf Rotations-Slots, deren Personenzahl aus `slots_count` der
Vorlage kommt — bei `slots_count=2` also zehn Kuchen, verteilt auf fünf Mannschaftszuordnungen.
Gemeint war: der Spieltag hat einen Kuchen-Bedarf, dieser wird auf möglichst wenige
Mannschaften gebündelt, und jede herangezogene Mannschaft bäckt bis zu
`bewirtung_max_per_team` Kuchen an ihrem **eigenen** Termin.

Der Fehler liegt in der Spezifikation, nicht in der Implementierung: die Requirement
„Bedarfsermittlung und Greedy-Zuteilung mit Cap" beschreibt die Zuteilung durchgängig in
Slots. Die Engine setzt sie korrekt um. Damit ist die Rotation im Alltag unbrauchbar — der
27.09.2026 (fünf Heimspiele, vier Mannschaften, Cap 2) erzeugt fünf Dienste zu je zwei Kuchen
statt drei Diensten zu 2 / 2 / 1.

## What Changes

- **BREAKING (fachlich):** Der Tagesbedarf ist eine Zahl **Kuchen**, nicht eine Zahl Slots.
  Aus `Bedarf` Kuchen entstehen `aufgerundet(Bedarf / Cap)` Slots statt `Bedarf` Slots.
- **BREAKING:** `slots_total` eines Rotations-Slots kommt aus der Zuteilung
  (`min(Cap, Rest)`), nicht mehr aus `game_template_items.slots_count`. Für rotations-aktive
  Items ist `slots_count` wirkungslos.
- **BREAKING:** Der Slot einer Mannschaft hängt an **ihrem eigenen** chronologisch ersten
  Heimspiel des Tages — nicht mehr am i-ten Heimspiel in Tagesreihenfolge. Damit ist ein
  Rotations-Slot wieder ein Dienst *dieser* Mannschaft zu *ihrem* Spiel.
- **BREAKING:** Die Deckelung des Bedarfs auf die Anzahl der Heimspiele entfällt. Bedarf ist
  `aufgerundet(Heimspiele × Verhältnis)`; ein Verhältnis > 1 schlägt jetzt durch (5 Spiele ×
  2 = 10 Kuchen), weil die frühere Begründung („höchstens ein Slot pro Spiel") wegfällt.
- **BREAKING:** Reicht `Warteschlange × Cap` nicht für den Bedarf, **verfällt** der Rest. Der
  bisherige Auffang-Slot mit `team_id = NULL` entfällt ersatzlos; die Lücke erscheint nur noch
  als Meldung im `regen_summary` (Datum, Duty-Type, Anzahl nicht zugeteilter Kuchen).
- **Vereinfachung:** Der Sonderfall im Restore-Matching (Rotations-Items matchen ohne
  `team_id`) entfällt. Weil die Team-Zuordnung nicht mehr aus der Tagesreihenfolge abgeleitet
  wird, sondern das spielende Team ist, gilt wieder der normale Dreier-Match
  `(duty_type_id, event_time, team_id)` für alle Items.
- **UI:** Im Vorlagen-Editor wird das Feld „Anzahl" für ein Item mit gesetzter
  Bewirtungsrotation als wirkungslos gekennzeichnet und deaktiviert, mit Verweis auf
  Einstellungen → Bewirtung.

Unverändert bleiben: Aufbau und Reihenfolge der Team-Warteschlange (jedes Team einmal, an der
Position seines ersten Heimspiels), der tageweise Neustart der Warteschlange, das
Ausrichter-Gate vor der Bedarfsrechnung, und die Voraussetzung
`same_day_behavior = adjacent_day_behavior = 'normal'`.

## Capabilities

### New Capabilities

Keine.

### Modified Capabilities

- `bewirtungsrotation`: Die Requirements „Bedarfsermittlung und Greedy-Zuteilung mit Cap" und
  „Restore-Matching für Rotations-Items ignoriert die Team-Zuordnung" werden ersetzt; die
  Requirement „Rotations-Schalter pro Vorlagen-Item" und „Vorlagen-Editor …" werden um die
  Wirkungslosigkeit von `slots_count` ergänzt.

## Test-Anforderungen

Keine Route ändert sich; die Invarianten sitzen in der Regen-Engine und werden über
`internal/games/rotation_regen_test.go` (DB + Regen-Lauf) abgesichert.

| Testname | Garantierte Invariante |
|---|---|
| `TestRotation_FuenfSpieleVierTeamsCapZwei` | Aus 5 Kuchen bei Cap 2 werden 3 Slots mit `slots_total` 2 / 2 / 1; die vierte Mannschaft bekommt keinen. |
| `TestRotation_Spieltag27September` | Regression auf das gemeldete Fehlerbild (gD, wB, mA2, mA1, mA1 bei Verhaeltnis 1, Cap 2): drei Slots 2 / 2 / 1, mA1 leer. |
| `TestRotation_SlotHaengtAmEigenenTermin` | `game_id` und `event_time` jedes Rotations-Slots gehören zum chronologisch ersten Heimspiel der zugeteilten Mannschaft. |
| `TestRotation_SlotsCountOhneWirkung` | `slots_total` folgt der Zuteilung, nicht `game_template_items.slots_count`. |
| `TestRotation_VerhaeltnisGroesserEinsSchlaegtDurch` | Bedarf ist `ceil(Spiele × Verhältnis)` ohne Deckelung auf die Spieleanzahl. |
| `TestRotation_RestbedarfVerfaellt` | Kein Slot mit `team_id IS NULL`; kein Team über dem Cap; `unassigned` weist die Differenz als Zahl aus. |
| `TestRotation_ZweiTeamsAmSelbenSpielBehaltenEigeneZusagen` | Zwei Mannschaften an einem Spiel bekommen eigene Slots zur identischen `event_time` und behalten im Restore ihre eigenen Zusagen (Dreier-Match). |
| `TestRotation_GesunkeneKuchenzahlVerliertJuengsteZusage` | Beim Rückschreiben bis `slots_total` überlebt die ältere Zusage; die jüngere wird als „entfernt" gemeldet. |

## Impact

- `internal/games/regen.go`: `rotationAssignment`/`rotationPlan`/`rotationGroup`,
  `buildRotationPlan`, der Rotations-Zweig in `regenGameItems`, `makeCustomKey`,
  `restoreAssignments`, `UnassignedEntry`.
- `internal/games/rotation_regen_test.go` und `internal/games/ausrichter_regen_test.go`:
  bestehende Erwartungen beschreiben die alte Semantik und werden umgeschrieben.
- Frontend: `web/src/pages/AdminDutyTemplateDetailPage.tsx`,
  `web/src/pages/AdminDutyTemplatesPage.tsx` (Feld „Anzahl" bei aktiver Rotation).
- Keine Migration, kein Schema-Wechsel, keine neue Route.
- **Betrieb:** Bestehende Rotations-Slots ändern sich erst beim nächsten Regen-Lauf für den
  jeweiligen Tag (Massenlauf „Dienste aktualisieren" oder Speichern eines Termins). Zusagen auf
  Slots, die dabei wegfallen oder deren Personenzahl sinkt, laufen regulär in die
  „entfernt"-Benachrichtigung.
