# Ablösung: ein Dienst endet, wenn der nächste gleichartige beginnt

## Why

Bewirtung/Kuchenverkauf ist kein Dienst **am Spiel**, sondern ein Dienst **am Spieltag**.
Die Rotation hat diesen Bruch schon halb vollzogen — der Kuchenbedarf ist tagesweit, der
Cap vereinsweit, und jede gezogene Mannschaft bekommt genau **einen** Slot an ihrem
eigenen ersten Heimspiel. Die **Dauer** dieses Slots ist beim Pro-Spiel-Modell
stehengeblieben: sie kennt nur das Anker-Spiel.

Folge sind systematische Doppelbesetzungen. Bei Spielen, die eng aufeinander folgen,
überlappt jeder Bewirtungsdienst mit seinem Nachfolger — der Vorgänger läuft bis zum Ende
*seines* Spiels weiter, obwohl die nächste Mannschaft längst übernommen hat:

```
 09:30  10:00        11:30 11:45      13:15 13:30      15:00 15:30
   │      │            │     │          │     │          │     │
   │   Spiel1 ─────────┘   Spiel2 ──────┘   Spiel3 ──────┘     │

  ┌──────────────────────┐
  │ A   09:30 – 12:00    │                  ← eigenes Spielende +30
  └──────────────────────┘
             ┌──────────────────────┐
             │ B   11:15 – 13:45    │       ▓ 45 min Doppelbesetzung,
             └──────────────────────┘       ▓ bei jeder Übergabe
                        ┌──────────────────────┐
                        │ C   13:00 – 15:30    │
                        └──────────────────────┘
```

Die überlappenden 45 Minuten sind nicht nur Rauschen im Plan: `duty_accounts.ist` summiert
`duty_slots.hours_value`, die erfundene Zeit wird also als geleistete Dienststunde
gutgeschrieben.

## What Changes

- **Neues Kennzeichen `end_at_next_duty`** auf `duty_types` und `game_template_items`
  (Migration `054`, Default `0` — rein additiv, Bestandsverhalten unverändert). Es gilt
  ausschließlich im Dauer-Modus `dynamisch` („Startzeit + Endzeit"); im Modus `absolut`
  bleibt es wie End-Anker und End-Versatz bedeutungslos.
- **Die Regel ist eine Kappung, kein neues Ende:**

  ```
  Ende = MIN( Start des nächsten gleichartigen Dienstes am selben Tag ,
              gepflegtes Ende aus End-Anker + End-Versatz )
  ```

  Der bestehende Modus-2-Endpunkt bleibt der **Deckel**. Die Ablösung kann ihn nur nach
  vorn ziehen, nie nach hinten. Gibt es keinen Nachfolger, greift der Deckel unverändert.
- **Kein dritter Dauer-Modus.** `duration_mode` bleibt zweiwertig; die Kappung ist ein
  Häkchen unter den End-Feldern. Begründung in `design.md` §1.
- **„Gleichartig" heißt: derselbe Diensttyp**, nicht „das nächste Spiel". Ein Heimspiel
  ohne Bewirtungs-Slot verlängert den Vorgänger **nicht** — sonst erbte die letzte
  eingeteilte Mannschaft Arbeit, für die kein Kuchen gezählt wurde.
- **Die Kappung liest die real entstandenen Slots**, nicht vorhergesagte: ein Nachlauf am
  Ende von `regenSingleDay` selektiert die `duty_slots` des Tages und kürzt. Damit sind
  alle fünf Gates der Erzeugung (Ausrichter, Rotationsplan, Team-Allowlist,
  Varianten-Umschreibung, `is_custom`-Konflikt) automatisch berücksichtigt, ohne dass
  irgendwer sie zweimal auswertet.
- **Keine neue Route, keine neue Validierung.** Ein gesetztes Kennzeichen kann eine Spanne
  nie unmöglich machen (siehe `design.md` §4) — die bestehende
  `impossible_duration_span`-Prüfung bleibt unverändert.

## Capabilities

### Modified Capabilities

- `duties`: Ein Dienst mit dynamischer Dauer kann so definiert werden, dass er endet,
  sobald der nächste gleichartige Dienst desselben Spieltags beginnt — spätestens jedoch
  zum gepflegten Ende.

## Non-Goals

- **Lückenlose Abdeckung des Spieltags.** Deckt die Rotation nicht alle Heimspiele ab
  (Bedarf nach zwei Mannschaften gedeckt, drittes und viertes Spiel ohne Slot), bleibt
  die Lücke. Das ist eine Frage von `bewirtung_verhaeltnis`, nicht der Dauer — die Kette
  regelt die *Übergabe*, nicht die *Abdeckung*.
- **Kein Eingriff in `buildRotationPlan`.** Bedarf, Warteschlange und Cap bleiben
  unangetastet.
- **Kein Anker „Tagesende".** Der Deckel ist und bleibt der eigene Termin.
- **Kein Kettenzustand über Tagesgrenzen.** Wie die Rotation startet die Kette an jedem
  Spieltag neu.
- **Keine eigene Zeile in der Regen-Zusammenfassung.** Die Kappung ist bei aktivem
  Kennzeichen der Normalfall, nicht die Ausnahme; sie zu melden wäre Rauschen.

## Test-Anforderungen

| Route / Pfad | Testname | Erwartung / Invariante |
|---|---|---|
| `POST /api/duty-types` | `TestCreateType_AbloesungWirdPersistiert` | 201, `end_at_next_duty=1` in der DB |
| `POST /api/duty-types` | `TestCreateType_AbloesungOhneDynamischenModusWirdGespeichertAberIgnoriert` | 201; Wert gespeichert, erzeugter Slot trägt die absolute Dauer |
| `PUT /api/duty-types/{id}` | `TestUpdateType_AbloesungWirdPersistiert` | 200, Wert geändert |
| `PUT /api/duty-types/{id}` | `TestUpdateType_AbloesungUnauthentifiziert` | 401 |
| `PUT /api/duty-templates/{id}` | `TestUpdateTemplate_AbloesungJeItemWirdPersistiert` | 200, Wert je Item gespeichert |
| `PUT /api/duty-templates/{id}` | `TestUpdateTemplate_AbloesungOhneFeldErbtVomTyp` | Fehlendes Feld erbt per Copy-on-pick, fällt nicht auf `false` |
| `PUT /api/duty-templates/{id}` | `TestUpdateTemplate_AbloesungOhneVorstandsrecht` | 403, Bestand unverändert |
| `GET /api/duty-types` | (in `TestCreateType_AbloesungWirdPersistiert` mitgeprüft) | Feld im JSON, damit die Maske es zeigen kann |
| Regen | `TestRegen_DienstEndetBeiAbloesung` | Vorgänger-Slot endet exakt zur Startzeit des Nachfolgers |
| Regen | `TestRegen_LetzterDienstDerKetteBehaeltDenDeckel` | Ohne Nachfolger unverändertes Ende aus End-Anker + End-Versatz |
| Regen | `TestRegen_NachfolgerNachDemDeckelKapptNicht` | Startet der Nachfolger nach dem gepflegten Ende, bleibt der Deckel |
| Regen | `TestRegen_NachfolgerVorDemEigenenStartWirdIgnoriert` | Rückwärts liegender Slot löst nicht ab; Deckel greift, Slot bleibt bestehen |
| Regen | `TestRegen_AbloesungNurDurchDenselbenDiensttyp` | Ein Slot eines anderen Diensttyps kappt nicht |
| Regen | `TestRegen_VarianteBestimmtDasKennzeichen` | Bei Varianten-Reduktion gilt `end_at_next_duty` des Varianten-Diensttyps |
| Regen | `TestRegen_ManuellerSlotLoestAbAberWirdNichtGekappt` | `is_custom=1`-Slot kappt den Vorgänger, behält selbst seine Dauer |
| Regen | `TestRegen_AusgenommenerTerminLoestTrotzdemAb` | Slot an einem Termin aus `excluded_game_ids` zählt als Ablösung |
| Regen | `TestRegen_KappungErzeugtNieEineNichtpositiveDauer` | Invariante: nach der Kappung trägt jeder Slot `hours_value > 0` |
| Regen | `TestRegen_AbsoluterModusWirdNichtGekappt` | Im Modus `absolut` bleibt die gepflegte Stundenzahl unangetastet |
| Regen | `TestRegen_OhneKennzeichenUnveraendert` | Charakterisierung: `end_at_next_duty=0` verhält sich exakt wie heute |
| Frontend | `AdminDutyTypesPage.abloesung.test.tsx` | Häkchen nur im Modus „Startzeit + Endzeit" sichtbar; Wert wird gesendet |
| Frontend | `AdminDutyTemplatesPage.abloesung.test.tsx` | Häkchen je Item; Copy-on-pick übernimmt es vom Diensttyp |
