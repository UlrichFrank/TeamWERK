# duty-assignment-comments Specification

## Purpose
Erlaubt Personen, die sich für einen Dienst-Slot eingetragen haben, einen kurzen
Freitext-Kommentar zu ihrer eigenen Zuteilung zu hinterlegen (z. B. welche Art
Kuchen sie mitbringen), sichtbar für alle mit Zugriff auf das Duty-Board.

## Requirements

### Requirement: Genau ein Kommentar pro Zuteilung

Das System SHALL pro `duty_assignments`-Zeile höchstens einen Kommentar
vorhalten. Ein erneuter Schreibvorgang derselben Person auf dieselbe Zuteilung
MUSS den bestehenden Kommentar überschreiben (Upsert), nicht einen zweiten
anlegen.

#### Scenario: Erstmaliges Setzen

- **WHEN** eine eingetragene Person `PUT /api/duty-assignments/{id}/comment`
  mit `{"body": "Marmorkuchen"}` aufruft
- **THEN** antwortet der Server mit HTTP 200
- **AND** existiert genau ein Kommentar zu dieser Zuteilung mit diesem Text

#### Scenario: Zweiter Schreibvorgang überschreibt

- **WHEN** dieselbe Person danach erneut `PUT` mit `{"body": "Käsekuchen"}`
  auf dieselbe Zuteilung aufruft
- **THEN** antwortet der Server mit HTTP 200
- **AND** existiert weiterhin genau ein Kommentar zu dieser Zuteilung, jetzt
  mit dem Text „Käsekuchen"

### Requirement: Nur die zugeteilte Person schreibt und löscht ihren eigenen Kommentar

Das System SHALL Schreib- und Löschzugriff auf einen Kommentar auf den
Inhaber der zugehörigen `duty_assignments`-Zeile beschränken, oder auf den
Elternteil-Account, der laut `family_links` für den zugeteilten Proxy-Child-
Nutzer eintragen darf (dieselbe Prüfung wie beim Eintragen selbst). Es SHALL
KEINEN Admin- oder Vorstand-Bypass geben.

#### Scenario: Eigentümer setzt seinen Kommentar

- **WHEN** der Nutzer, dessen `duty_assignments`-Zeile existiert,
  `PUT /api/duty-assignments/{id}/comment` mit gültigem Body aufruft
- **THEN** antwortet der Server mit HTTP 200

#### Scenario: Elternteil setzt Kommentar für sein Proxy-Kind

- **WHEN** ein Elternteil-Account, der laut `family_links` mit dem
  zugeteilten (nicht login-fähigen) Kind verknüpft ist,
  `PUT /api/duty-assignments/{id}/comment` für dessen Zuteilung aufruft
- **THEN** antwortet der Server mit HTTP 200

#### Scenario: Fremder Nutzer wird abgelehnt

- **WHEN** ein Nutzer, der weder Inhaber der Zuteilung noch dessen Elternteil
  ist — auch nicht mit System-Rolle `admin` oder Vereinsfunktion `vorstand` —
  `PUT` oder `DELETE` auf eine fremde Zuteilung aufruft
- **THEN** antwortet der Server mit HTTP 403
- **AND** bleibt ein eventuell vorhandener Kommentar unverändert

#### Scenario: Zuteilung existiert nicht

- **WHEN** der Aufrufer eine `assignmentId` verwendet, zu der keine
  `duty_assignments`-Zeile existiert
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Anonymer Aufruf

- **WHEN** der Aufruf ohne gültigen Bearer-JWT erfolgt
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Kommentar wird mit der Zuteilung automatisch gelöscht

Das System SHALL einen Kommentar entfernen, sobald die zugehörige
`duty_assignments`-Zeile entfällt — unabhängig davon, über welchen Pfad das
geschieht (Austragen, Massen-Regen, Spiel-Löschung, Slot-Löschung über das
Kalender-Modal). Es SHALL keinen Zustand geben, in dem ein Kommentar auf eine
nicht mehr existierende Zuteilung verweist.

#### Scenario: Austragen entfernt den eigenen Kommentar

- **WHEN** eine Person mit gesetztem Kommentar sich über
  `DELETE /api/duty-board/{slotId}/claim` austrägt
- **THEN** ist ihre `duty_assignments`-Zeile gelöscht
- **AND** existiert zu dieser (nicht mehr vorhandenen) Zuteilung kein
  Kommentar mehr

#### Scenario: Slot-Löschung entfernt alle Kommentare der betroffenen Zuteilungen

- **WHEN** ein Slot mit mehreren kommentierten Zuteilungen über das
  Kalender-Modal gelöscht wird
- **THEN** sind alle `duty_assignments`-Zeilen dieses Slots gelöscht
- **AND** existiert zu keiner dieser Zuteilungen mehr ein Kommentar

### Requirement: Leserecht ist universell für Board-Zugriff

Das System SHALL alle Kommentare eines Slots jedem Nutzer mit Zugriff auf
das Duty-Board zeigen, unabhängig davon, ob er selbst für diesen Slot
eingetragen ist.

#### Scenario: Beliebiger Board-Nutzer liest alle Kommentare

- **WHEN** ein eingeloggter Nutzer `GET /api/duty-slots/{id}/comments`
  für einen Slot mit Kommentaren mehrerer Personen aufruft
- **THEN** antwortet der Server mit HTTP 200
- **AND** enthält die Antwort die Kommentare aller Personen, die für diesen
  Slot eingetragen sind und einen Kommentar gesetzt haben

#### Scenario: Unbekannter Slot

- **WHEN** der Aufrufer eine `id` verwendet, zu der kein `duty_slots`-Eintrag
  existiert
- **THEN** antwortet der Server mit HTTP 404

### Requirement: Duty-Board liefert eine Kommentar-Zählung, keinen Volltext

Das System SHALL im Duty-Board-Response (`GET /api/duty-board`) pro Slot ein
Feld `comment_count` liefern, aber NICHT den Kommentartext selbst — der
Volltext wird ausschließlich über `GET /api/duty-slots/{id}/comments`
nachgeladen.

#### Scenario: Slot ohne Kommentare

- **WHEN** ein Slot keine kommentierten Zuteilungen hat
- **THEN** ist `comment_count` im Board-Response für diesen Slot `0`

#### Scenario: Slot mit Kommentaren

- **WHEN** zwei von drei Zuteilungen eines Slots einen Kommentar tragen
- **THEN** ist `comment_count` im Board-Response für diesen Slot `2`
- **AND** enthält der Board-Response an keiner Stelle den Kommentartext selbst

### Requirement: Eingabevalidierung für den Kommentartext

Das System SHALL einen leeren oder rein aus Whitespace bestehenden Body
ablehnen (Löschen läuft über den eigenen `DELETE`-Endpoint, nicht über einen
leeren `PUT`) und den Text auf 280 Byte UTF-8 begrenzen.

#### Scenario: Leerer Body wird abgelehnt

- **WHEN** `PUT /api/duty-assignments/{id}/comment` mit `{"body": ""}` oder
  nur Whitespace aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400
- **AND** bleibt ein eventuell vorhandener Kommentar unverändert

#### Scenario: Zu langer Text wird abgelehnt

- **WHEN** `body` mehr als 280 Byte UTF-8 umfasst
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Mutationen lösen ein SSE-Broadcast aus

Das System SHALL nach jedem erfolgreichen `PUT` oder `DELETE` auf einen
Kommentar ein SSE-Ereignis `duties` senden, damit offene `/dienste`-Sessions
die neue `comment_count` ohne manuellen Reload sehen.

#### Scenario: Setzen broadcastet

- **WHEN** ein Kommentar erfolgreich gesetzt wird
- **THEN** wird ein SSE-Ereignis `duties` gesendet

#### Scenario: Löschen broadcastet

- **WHEN** ein Kommentar erfolgreich gelöscht wird
- **THEN** wird ein SSE-Ereignis `duties` gesendet
