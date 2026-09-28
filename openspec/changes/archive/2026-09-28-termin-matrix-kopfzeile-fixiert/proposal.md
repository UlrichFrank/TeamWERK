## Why

In der Tabellenansicht von `/termine` (`view=tabelle`) bleibt beim horizontalen Scrollen die Namensspalte stehen, die Titelzeile mit Terminsymbol und Datum aber nicht: bei einem Kader von 20+ Spielern scrollt der Kopf beim vertikalen Scrollen aus dem Bild, und man weiß weiter unten nicht mehr, zu welchem Termin eine Zelle gehört. Die Titelzeile soll sich genauso verhalten wie die erste Spalte.

## What Changes

- Die Titelzeile der Termin-Tabelle (Kopf „Spieler", „Bisher", „Geplant" und die Terminspalten) bleibt beim vertikalen Scrollen am oberen Rand der Tabelle stehen.
- Die Kopfzelle „Spieler" bleibt in beide Richtungen stehen (Schnittpunkt von fixierter Zeile und fixierter Spalte) und liegt über allen anderen Zellen.
- Die Tabelle wird dafür in ihrem Kartenrahmen zu einem eigenen, in der Höhe auf den Sichtbereich begrenzten Scrollbereich (beide Achsen). Nebeneffekt: die horizontale Scrollleiste ist sichtbar, ohne erst ans Tabellenende scrollen zu müssen.
- Keine Änderung an Daten, API, Berechtigungen oder Rückmeldelogik.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `termin-matrix`: Requirement „Tabellenansicht auf /termine" — zusätzlich zur stehenbleibenden Namensspalte bleibt die Titelzeile beim vertikalen Scrollen stehen.

## Impact

- Frontend: `web/src/components/TerminMatrix.tsx` (Scroll-Container und Kopfzellen-Klassen), Test `web/src/components/TerminMatrix.test.tsx`.
- Kein Backend, keine Migration, keine Route, kein SSE-Event. Berechtigungsmodell unberührt; kein zusätzlicher Speicherbedarf auf dem VPS.
