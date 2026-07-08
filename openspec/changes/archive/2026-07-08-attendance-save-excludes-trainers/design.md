## Kontext

Die Anwesenheitsliste enthält seit dem Trainer-RSVP-Feature zwei Arten von Zeilen:
Spieler (mit Anwesenheits-Checkbox) und Trainer (`is_trainer=1`, ohne Checkbox). Der
Speicher-Pfad `toggleAttendance` sendet beim Toggle eines Spielers immer das **gesamte**
aktuelle Roster als Bulk-Upsert (damit alle bekannten Zustände konsistent persistiert
werden). Dieses Bulk-Paket enthält fälschlich auch die Trainer-Zeilen.

## Entscheidung 1 — Fix am Frontend (primär)

`toggleAttendance` filtert `is_trainer`-Zeilen aus dem `ids`-Universum. Das ist die
eigentliche Fehlerstelle: Die Anzeige filtert bereits (`row.is_trainer ? null : <checkbox>`),
der Speicher-Pfad tat es nicht. Der Filter stellt die vor dem Trainer-RSVP-Feature
bestehende Invariante wieder her: **Das Speicher-Paket enthält nur Member mit Checkbox.**

## Entscheidung 2 — Backend härten (Defense-in-Depth)

**Optionen:**

| Ansatz | Verhalten bei Trainer im Paket | Risiko |
|---|---|---|
| A: nur Frontend-Filter | Backend lehnt weiter mit 400 ab | Jeder andere Client / künftige Regression bricht erneut |
| B: Backend überspringt Trainer-Einträge still (`continue`) | Speichert die Spieler, ignoriert Trainer | Kein Fehler mehr sichtbar, wenn versehentlich Trainer gesendet werden |
| C: Backend akzeptiert Trainer-Anwesenheit | ändert Fachlogik | Nicht gewünscht — Trainer haben bewusst keine Anwesenheit |

**Gewählt: A + B.** Frontend filtert (behebt die Ursache), Backend überspringt Trainer-only-
Einträge robust (verhindert Wiederauftreten über andere Pfade). Option C bleibt bewusst
außen vor — die fachliche Regel „Trainer haben keine Anwesenheitserfassung" bleibt.

**Grenzfall:** Ein Paket, das **ausschließlich** Trainer-Einträge (und keinen Spieler)
enthält, ist ein echter Client-Fehler. Zwei Sichtweisen:
- streng: weiter 400 zurückgeben;
- tolerant: 204 mit 0 gespeicherten Zeilen.

Empfehlung: **tolerant (204, 0 Writes)** — nach dem Frontend-Filter kann dieser Fall im
Normalbetrieb ohnehin nicht mehr auftreten; ein leeres/trainer-only-Paket ist dann
schlicht ein No-op. Das hält den Handler einfacher (eine Schleife, `continue`, kein
Sonder-Abbruch). Endgültige Festlegung beim Umsetzen.

## Was NICHT Teil der Änderung ist

- Keine Änderung an Roster-/RSVP-Queries (Trainer-Zeilen bleiben in der Anzeige).
- Keine Änderung an Berechtigungen (`hasTeamAccess`, `canRecordGameAttendance`).
- Keine Season-/Kader-Scoping-Änderung (war eine verworfene Hypothese, nicht die Ursache).
