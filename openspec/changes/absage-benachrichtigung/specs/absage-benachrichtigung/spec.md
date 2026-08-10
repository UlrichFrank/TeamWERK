## ADDED Requirements

### Requirement: Absage-Benachrichtigungen benennen Termin und Auslöser

Das System SHALL bei jeder Löschung eines Termins oder Dienstes eine Benachrichtigung
versenden, deren Text den **Namen des betroffenen Termins** (Gegner, Titel, Serienname bzw.
Dienstart + Event) und ein **Datum im Format `TT.MM.JJJJ`** enthält. Platzhaltertexte ohne
diese Angaben SHALL das System NICHT mehr versenden.

Das System SHALL den **Namen des auslösenden Nutzers** (`users.first_name` +
`users.last_name`) im Text nennen. Sind beide Felder leer, SHALL das System auf eine
generische Formulierung ohne Namen zurückfallen und SHALL NICHT die E-Mail-Adresse
preisgeben.

Betroffen sind `DELETE /api/games/{id}`, `DELETE /api/training-sessions/{id}`,
`DELETE /api/training-series/{id}` und `DELETE /api/duty-slots/{id}`.

#### Scenario: Spiel wird gelöscht
- **WHEN** ein Nutzer mit Löschrecht `DELETE /api/games/{id}` für ein Spiel gegen „HSG Ostfildern" am 14.09.2026 aufruft
- **THEN** enthält der Body der `games`-Benachrichtigung den Text „HSG Ostfildern" und „14.09.2026"
- **THEN** enthält der Body den Vor- und Nachnamen des auslösenden Nutzers

#### Scenario: Auslöser hat keinen hinterlegten Namen
- **WHEN** der auslösende Nutzer leere `first_name` und `last_name` hat
- **THEN** enthält der Body eine generische Formulierung ohne Namen
- **THEN** enthält der Body NICHT die E-Mail-Adresse des Nutzers

#### Scenario: Dienst-Slot wird gelöscht
- **WHEN** `DELETE /api/duty-slots/{id}` für einen Slot aufgerufen wird, für den Nutzer eingetragen sind
- **THEN** enthält der Body den Namen der Dienstart, den Event-Namen und das Event-Datum

### Requirement: Optionaler Löschgrund im Request

Das System SHALL bei den vier Löschrouten einen optionalen JSON-Body
`{ "reason": string, "silent": boolean }` entgegennehmen und den `reason` in den
Benachrichtigungstext aufnehmen.

Das System SHALL einen fehlenden, leeren oder nicht dekodierbaren Body als „kein Grund,
nicht stumm" behandeln und mit dem bisherigen Erfolgsstatus antworten — NICHT mit HTTP 400.

Das System SHALL `reason` trimmen und auf **200 Zeichen still kürzen**. Ein zu langer Grund
SHALL NICHT zu einem Fehlerstatus führen.

Das System SHALL den `reason` **nirgends persistieren** — weder in einer Tabelle noch in
einer Datei noch im Log.

#### Scenario: Löschung mit Grund
- **WHEN** `DELETE /api/games/{id}` mit `{"reason":"Halle gesperrt"}` aufgerufen wird
- **THEN** enthält der Benachrichtigungs-Body den Text „Halle gesperrt"

#### Scenario: Löschung ohne Grund
- **WHEN** `DELETE /api/games/{id}` ohne `reason` aufgerufen wird
- **THEN** endet der Body nach dem Aktor-Satz und enthält KEINE leere Grund-Einleitung

#### Scenario: Alter Client sendet keinen Body
- **WHEN** eine der vier Löschrouten ohne Request-Body aufgerufen wird
- **THEN** antwortet der Server mit dem bisherigen Erfolgsstatus
- **THEN** wird die Benachrichtigung ohne Grund versendet

#### Scenario: Überlanger Grund
- **WHEN** ein `reason` mit 500 Zeichen übergeben wird
- **THEN** antwortet der Server mit Erfolg
- **THEN** enthält der Benachrichtigungstext genau die ersten 200 Zeichen des Grundes

#### Scenario: Grund landet in keiner Tabelle
- **WHEN** eine Löschung mit einem eindeutigen Marker-Grund durchgeführt wird
- **THEN** findet ein vollständiger Scan aller Tabellen und Spalten der Datenbank diesen String nicht
- **THEN** enthält auch der Log-Puffer des Requests diesen String nicht

### Requirement: Benachrichtigungen ohne Ziel navigieren nicht

Das System SHALL für Benachrichtigungen, die auf keinen erreichbaren Ort mehr zeigen
können, den leeren String als Ziel-URL zulassen und ihn auf allen Zustellwegen als „kein
Ziel" behandeln — nicht als ungültiges Ziel.

Bei leerer Ziel-URL SHALL die E-Mail keine „Direktlink"-Zeile anhängen.

Bei leerer oder fehlender `url` SHALL der Service Worker das bestehende App-Fenster nur
**fokussieren** bzw. — falls keines existiert — die App-Wurzel öffnen, und SHALL in diesem
Fall KEINE Navigation auslösen.

Die Zielauflösung des Service Workers SHALL in einer exportierten, reinen Funktion liegen,
damit sie ohne Service-Worker-Laufzeit testbar ist.

#### Scenario: E-Mail ohne Direktlink
- **WHEN** eine Benachrichtigung mit leerer URL per E-Mail zugestellt wird
- **THEN** enthält der Mailtext keine Zeile, die mit „Direktlink:" beginnt

#### Scenario: Klick auf eine Absage-Push bei geöffneter App
- **WHEN** der Nutzer auf eine Benachrichtigung mit `url: ""` klickt und ein App-Fenster geöffnet ist
- **THEN** wird das Fenster fokussiert
- **THEN** wird `navigate()` NICHT aufgerufen und die aktuell angezeigte Seite nicht neu geladen

#### Scenario: Klick auf eine Push ohne url-Feld
- **WHEN** der Nutzer auf eine Benachrichtigung klickt, deren `data.url` fehlt
- **THEN** wird das Fenster fokussiert bzw. die App-Wurzel geöffnet, ohne Navigation

#### Scenario: Klick auf eine Push mit Ziel
- **WHEN** der Nutzer auf eine Benachrichtigung mit `url: "/dienste"` klickt
- **THEN** wird das Fenster fokussiert und zu `/dienste` navigiert

### Requirement: Vorstand und Admin können die Benachrichtigung unterdrücken

Das System SHALL die Capability `suppress_event_notification` bereitstellen und sie
Nutzern mit Systemrolle `admin` oder Vereinsfunktion `vorstand` zuweisen
(`policy.IsVorstandLike`). Die Capability SHALL in der Antwort von `GET /api/me` erscheinen.

Ruft ein Nutzer mit dieser Capability eine der vier Löschrouten mit `{"silent": true}` auf,
SHALL das System **keine** Benachrichtigung versenden — weder die Team- noch die
Dienst-Benachrichtigung.

Ruft ein Nutzer **ohne** diese Capability eine Löschroute mit `{"silent": true}` auf,
SHALL das System das Flag ignorieren, die Löschung normal ausführen und die
Benachrichtigungen versenden. Das System SHALL in diesem Fall NICHT mit HTTP 403 antworten.

Das System SHALL das SSE-Live-Update (`Broadcast`) unabhängig von `silent` immer auslösen.

Die Capability wirkt nur, soweit der Nutzer die Route überhaupt erreicht. Die drei
Trainings-Routen liegen im Router hinter `RequireClubFunction("trainer",
"sportliche_leitung")` und weisen einen **reinen** Vorstand bereits mit HTTP 403 ab; dort
greift die Unterdrückung faktisch nur für Admins (oder einen Vorstand, der zusätzlich
`trainer`/`sportliche_leitung` trägt). Bei `DELETE /api/games/{id}` und
`DELETE /api/duty-slots/{id}` gilt die Regel uneingeschränkt. Diese Tier-Differenz ist
Bestand und wird von diesem Change NICHT verändert.

#### Scenario: Vorstand löscht eine Import-Dublette stumm
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `DELETE /api/games/{id}` mit `{"silent":true}` aufruft
- **THEN** wird keine Benachrichtigung versendet
- **THEN** wird das Spiel gelöscht und das Live-Update gesendet

#### Scenario: Stummschaltung deckt beide Meldungen ab
- **WHEN** ein Vorstand ein Spiel mit eingetragenen Dienst-Zuständigen mit `{"silent":true}` löscht
- **THEN** wird weder die `games`- noch die `duties`-Benachrichtigung versendet

#### Scenario: Trainer darf nicht stummschalten
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` (ohne `vorstand`) ein seine Mannschaft betreffendes Spiel mit `{"silent":true}` löscht
- **THEN** antwortet der Server mit Erfolg und löscht das Spiel
- **THEN** werden die Benachrichtigungen trotzdem versendet

#### Scenario: Live-Update ist nicht unterdrückbar
- **WHEN** eine Löschung mit `{"silent":true}` ausgeführt wird
- **THEN** wird der zugehörige `Broadcast` genau wie ohne das Flag ausgelöst

#### Scenario: Reiner Vorstand erreicht die Trainings-Routen nicht
- **WHEN** ein Nutzer mit ausschließlich der Vereinsfunktion `vorstand` `DELETE /api/training-sessions/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403, bevor der Handler läuft
- **THEN** bleibt das Verhalten bei `DELETE /api/games/{id}` und `DELETE /api/duty-slots/{id}` für denselben Nutzer unverändert erlaubt
