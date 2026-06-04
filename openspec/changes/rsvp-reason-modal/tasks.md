## 1. State-Bereinigung in TerminePage.tsx

- [x] 1.1 Alle Referenzen auf `reasons` und `setReasons` entfernen (undeclared state, verursacht Broken behavior)
- [x] 1.2 Sicherstellen, dass `pendingRSVP` und `modalReason` korrekt deklariert sind (bereits vorhanden laut Code)

## 2. Button-Handler umverdrahten

- [x] 2.1 „Vielleicht"-Button-Handler für Training (eigene RSVP): setzt `pendingRSVP` und resettet `modalReason`, ruft `respondTraining` nicht mehr direkt auf
- [x] 2.2 „Absagen"-Button-Handler für Training (eigene RSVP): setzt `pendingRSVP` und resettet `modalReason`, ruft `respondTraining` nicht mehr direkt auf
- [x] 2.3 „Vielleicht"-Button-Handler für Training (Kind-RSVP, Elternteil): analog mit `memberId`
- [x] 2.4 „Absagen"-Button-Handler für Training (Kind-RSVP, Elternteil): analog mit `memberId`
- [x] 2.5 „Vielleicht"- und „Absagen"-Handler für Spiele (eigene RSVP): analog zu 2.1/2.2 für `respondGame`
- [x] 2.6 „Vielleicht"- und „Absagen"-Handler für Spiele (Kind-RSVP): analog zu 2.3/2.4 für `respondGame`

## 3. Inline-Felder entfernen

- [x] 3.1 `<input>`-Feld für Begründung aus Training-Karte (eigene RSVP) entfernen
- [x] 3.2 `<input>`-Feld für Begründung aus Training-Karte (Kind-RSVP) entfernen
- [x] 3.3 `<input>`-Feld für Begründung aus Spiel-Karte (eigene RSVP) entfernen
- [x] 3.4 `<input>`-Feld für Begründung aus Spiel-Karte (Kind-RSVP) entfernen

## 4. Modal implementieren

- [x] 4.1 Modal-JSX am Ende des `TerminePage`-Returns hinzufügen (nur sichtbar wenn `pendingRSVP !== null`)
- [x] 4.2 Modal-Overlay: `fixed inset-0 z-50 bg-black/40 flex items-center justify-center`
- [x] 4.3 Modal-Inhalt: Titel mit Aktion (Absagen/Vielleicht) und ggf. Kindname, `<textarea>` für Begründung gebunden an `modalReason`
- [x] 4.4 OK-Button: `disabled={modalReason.trim() === ''}`, ruft bei Klick die entsprechende `respondTraining`/`respondGame`-Funktion mit Begründung auf und setzt `pendingRSVP` auf `null`
- [x] 4.5 Abbrechen-Button: setzt `pendingRSVP` auf `null` und `modalReason` auf `''`, kein API-Aufruf
- [x] 4.6 Styling nach Projektkonventionen: Modal-Klassen aus CLAUDE.md (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`), Buttons Primary/Danger
