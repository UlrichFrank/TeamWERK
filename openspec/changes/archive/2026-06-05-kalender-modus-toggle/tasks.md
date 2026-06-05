## 1. Backend: Middleware erweitern

- [x] 1.1 In `internal/games/handler.go` die Middleware für `PUT /api/admin/games/{id}` von `RequireRole("admin")` auf `RequireRole("admin", "trainer", "vorstand")` erweitern
- [x] 1.2 Manuell testen: Trainer-Token kann `PUT /api/admin/games/{id}` aufrufen (HTTP 200); Spieler-Token erhält HTTP 403

## 2. GameEditModal implementieren

- [x] 2.1 `web/src/components/GameEditModal.tsx` erstellen — Props: `game: Game`, `onClose: () => void`, `onSaved: () => void`
- [x] 2.2 Formular-Felder: opponent (Text-Input), date (Date-Input), time (Time-Input), event_type (Select: heim/auswärts/generisch)
- [x] 2.3 Submit: `PUT /api/admin/games/${game.id}` mit geänderten Feldern; Fehler in Alert-Fehler-Klasse anzeigen
- [x] 2.4 Schließen via `<X>`-Button und Escape-Taste (`useEscapeKey`-Hook einbinden)
- [x] 2.5 Standard-Modal-Klassen verwenden: `bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`
- [x] 2.6 Nach Speichern: `onSaved()` aufrufen → triggert Kalender-Reload

## 3. EventInfoModal implementieren

- [x] 3.1 `web/src/components/EventInfoModal.tsx` erstellen — Props: `type: 'game' | 'training'`, `game?: Game`, `training?: Training`, `onClose: () => void`
- [x] 3.2 Spieltag-Ansicht: Event-Typ-Icon, Gegner, Datum, Uhrzeit, RSVP-Zahlen (confirmed/declined/maybe)
- [x] 3.3 Training-Ansicht: Dumbbell-Icon, Titel, Datum, Startzeit–Endzeit, Ort, RSVP-Zahlen
- [x] 3.4 Kein Bearbeiten-Button, keine API-Calls
- [x] 3.5 Schließen via `<X>`-Button und Escape-Taste

## 4. KalenderPage: Modus-Toggle und Click-Handler

- [x] 4.1 State `kalenderMode: 'dienste' | 'termine'` mit Default `'dienste'` zu KalenderPage hinzufügen
- [x] 4.2 Segmentierten Toggle `[Dienste | Termine]` oben rechts neben `<h1>` einfügen — gleiche CSS-Klassen wie Mitfahrgelegenheiten (`flex rounded-lg border border-brand-border-subtle overflow-hidden text-sm`)
- [x] 4.3 Spieltag-Click-Handler: `if (kalenderMode === 'dienste') navigate('/kalender/:id')` else `if (canEdit) setEditingGame(g)` else `setInfoItem({type:'game', game:g})`
- [x] 4.4 Training-Click-Handler: `if (kalenderMode === 'dienste') return` (kein onClick); else `if (canEdit) setEditingTraining(t)` else `setInfoItem({type:'training', training:t})`
- [x] 4.5 Training-Kachel im Dienste-Modus: `cursor-default` setzen, `hover:bg-*`-Klassen entfernen (kein Hover-Effekt)
- [x] 4.6 States für neue Modals hinzufügen: `editingGame: Game | null`, `infoItem: {type, game?, training?} | null`
- [x] 4.7 `GameEditModal` und `EventInfoModal` in KalenderPage einbinden (Import + JSX am Ende der Komponente)
- [x] 4.8 `canEdit`-Prüfung: `user?.role === 'admin' || hasFunction(user, 'trainer') || hasFunction(user, 'vorstand') || hasFunction(user, 'sportliche_leitung')`

## 5. Integration und manuelle QA

- [x] 5.1 Dienste-Modus: Spieltag-Klick navigiert zu SpieltagDetailPage ✓
- [x] 5.2 Dienste-Modus: Training-Klick bewirkt nichts, kein Hover-Effekt ✓
- [x] 5.3 Termine-Modus als Trainer: Spieltag-Klick öffnet GameEditModal, Speichern aktualisiert Kalender ✓
- [x] 5.4 Termine-Modus als Trainer: Training-Klick öffnet TrainingEditModal ✓
- [x] 5.5 Termine-Modus als Spieler: Spieltag-Klick öffnet EventInfoModal (schreibgeschützt) ✓
- [x] 5.6 Termine-Modus als Spieler: Training-Klick öffnet EventInfoModal (schreibgeschützt) ✓
- [x] 5.7 Toggle-Darstellung: aktiver Modus gelb, inaktiver Modus grau ✓
- [x] 5.8 Beide Modals schließen via Escape und X-Button ✓
