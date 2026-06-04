## 1. GameModal-Komponente

- [ ] 1.1 `web/src/components/GameModal.tsx` erstellen: Props `game`, `editable`, `onClose`, `onSaved`
- [ ] 1.2 Read-only-Ansicht implementieren: Datum, Uhrzeit, Gegner, Teams als Text + Schließen-Button
- [ ] 1.3 Edit-Formular implementieren: Felder Datum, Uhrzeit, Gegner, Teams (Multi-Select aus vorhandenen Teams)
- [ ] 1.4 PUT-Request `PUT /admin/games/{id}` beim Speichern aufrufen, Fehler inline anzeigen
- [ ] 1.5 Erfolgreich gespeichert: Modal schließen und `onSaved()` aufrufen

## 2. KalenderPage – Toggle und Klick-Routing

- [ ] 2.1 `viewMode`-State (`'spiel' | 'dienst'`, Default `'dienst'`) in `KalenderPage` einführen
- [ ] 2.2 Toggle-Button „Spiel | Dienst" im Header rendern (gleiche Optik wie Mitfahrgelegenheiten-Toggle)
- [ ] 2.3 Klick auf Spiel-Pill: im Dienst-Modus `navigate('/kalender/{id}')`, im Spiel-Modus `GameModal` öffnen
- [ ] 2.4 `GameModal` in KalenderPage einbinden: `editable` aus `user.role === 'admin' || user.role === 'trainer'`
- [ ] 2.5 Nach `onSaved`: Kalenderdaten neu laden (refetch)

## 3. Sonstiges-Filter im Spiel-Modus

- [ ] 3.1 Sonstiges-Filter-Button im Spiel-Modus: `opacity-40 cursor-not-allowed`, Klick-Handler deaktivieren
- [ ] 3.2 Beim Wechsel in Spiel-Modus: `generisch` aus aktivem `filterTypes`-Set entfernen

## 4. Verifikation

- [ ] 4.1 Dienst-Modus: Klick auf Spiel-Pill navigiert zu `/kalender/{id}` (unverändertes Verhalten)
- [ ] 4.2 Spiel-Modus als Admin: GameModal öffnet sich mit Bearbeitungsformular, Speichern aktualisiert Eintrag
- [ ] 4.3 Spiel-Modus als Spieler: GameModal öffnet sich read-only, kein Speichern-Button
- [ ] 4.4 Sonstiges-Button im Spiel-Modus: nicht klickbar; beim Toggle-Wechsel automatisch deaktiviert
