## REMOVED Requirements

### Requirement: DSGVO-Anzeige (read-only) im Datenschutz-Tab

**Reason:** Die read-only-Formulierung entspricht nicht mehr dem
implementierten Verhalten. Ersetzt durch das neue Requirement
„DSGVO-Einwilligungen mit Change-Request", das die Schalter als
bedienbar spezifiziert und den Draft-Approval-Flow explizit macht.

**Migration:** Kein Code-Impact — dieses Requirement war seit
`7e1a91e "fix(members): DSGVO-Schalter im Profil editierbar via
Change-Request"` implementierungsseitig überholt. Die neue Fassung
dokumentiert den Ist-Stand.

## ADDED Requirements

### Requirement: DSGVO-Einwilligungen mit Change-Request im Datenschutz-Tab

Der Datenschutz-Tab SHALL die DSGVO-Einwilligungen des eigenen Members anzeigen:
„Datenverarbeitung eingewilligt", „Datenweitergabe eingewilligt" und
„Foto-Veröffentlichung eingewilligt" — je mit Status (Ja/Nein) und Datum (falls
vorhanden). Zu **jedem** der drei Schalter SHALL ein kurzer Erklärtext angezeigt
werden, der beschreibt, was mit der Einwilligung verbunden ist (Verarbeitung der
Mitgliedsdaten; Weitergabe an Dritte; Veröffentlichung von Fotos auf
öffentlichen Kanälen des Vereins).

Die Schalter SHALL **aktiv (bedienbar)** sein. Änderungen am Schalter-Zustand
SHALL nur lokal in den Draft-Kandidaten laufen und NIE direkt auf den Member
geschrieben werden. Das Speichern SHALL ausschließlich über den bestehenden
Change-Request-Draft-Workflow erfolgen (`POST /api/members/{id}/change-request`
mit `field_name='dsgvo'`, `new_value={verarbeitung, weitergabe, foto_veroeffentlichung}`).

Der „Änderung anfragen"-Button SHALL gesperrt sein, solange die lokalen Werte
mit den Server-Werten übereinstimmen (kein Draft ohne Diff). Ein ausstehender
Draft SHALL pro geänderter Einwilligung als „(angefragt: Ja|Nein)" hinter dem
Schalter-Label sichtbar sein. Ein „Anfrage zurückziehen"-Button SHALL den Draft
löschen (`DELETE /api/members/{id}/change-drafts/{id}`) und die lokalen Werte
auf den Server-Stand zurücksetzen.

#### Scenario: DSGVO-Status wird angezeigt

- **WHEN** der Tab geladen wird
- **THEN** sieht der Nutzer den Status von `dsgvo_verarbeitung`, `dsgvo_weitergabe`
  und `foto_veroeffentlichung` mit zugehörigen `_date`-Werten
- **AND** alle drei Schalter sind bedienbar (`disabled=false`)

#### Scenario: Erklärtext je Schalter

- **WHEN** der DSGVO-Block gerendert wird
- **THEN** steht unter jedem der drei Schalter ein erläuternder Text zu seiner Bedeutung

#### Scenario: Anfrage-Button ohne Diff gesperrt

- **WHEN** der Nutzer den Tab öffnet und keine Schalter umgestellt hat
- **THEN** ist der Button „Änderung anfragen" gesperrt
- **AND** kein `POST /change-request` wird ausgelöst

#### Scenario: Änderung wird als Draft angefragt

- **WHEN** der Nutzer einen Schalter umstellt und „Änderung anfragen" klickt
- **THEN** wird `POST /api/members/{id}/change-request` mit
  `field_name='dsgvo'` und `new_value` als Objekt der drei Boolwerte gesendet
- **AND** danach zeigt der Tab pro abweichender Einwilligung „(angefragt: …)"
  hinter dem Label
- **AND** der Server-Wert von `dsgvo_verarbeitung` etc. bleibt unverändert bis
  zur Approval durch Vorstand

#### Scenario: Draft zurückziehen

- **WHEN** der Nutzer bei ausstehendem Draft „Anfrage zurückziehen" klickt
- **THEN** wird `DELETE /api/members/{id}/change-drafts/{id}` gesendet
- **AND** die Schalter werden auf den Server-Stand (`ownMember.dsgvo_*`) zurückgesetzt
- **AND** der Anfrage-Button ist wieder gesperrt (kein Diff)
