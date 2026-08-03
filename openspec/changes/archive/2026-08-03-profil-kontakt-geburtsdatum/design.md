## Context

Der Kontakt-Tab im Profil (`ProfileProfilTab.tsx`, eigenes und Kind-Profil) pflegt Name + Adresse über einen zweistufigen Mechanismus:

1. **Self-Service-Kopie** auf `users` (`street`, `zip`, `city`) — sofortiger Write über `PUT /api/profile/me` bzw. `PUT /api/profile/kind/:memberId/account`. Ohne Vorstands-Freigabe, wirkt sofort für den Nutzer selbst.
2. **Änderungsantrag** ans offizielle Mitglieder-Record (`member_change_drafts`, `field_name='profil'`) — dieselben Werte werden zusätzlich als Draft eingereicht, den der Vorstand annehmen oder ablehnen muss, bevor `members.street/zip/city` tatsächlich geändert wird.

Geburtsdatum existiert bisher ausschließlich in `members.date_of_birth` (Mitgliederbereich, nur Vorstand direkt editierbar via `ProfileMemberTab`/`onSaveDirect`) — es gibt keine `users`-Spalte und keinen Weg für Mitglieder, selbst eine Korrektur anzustoßen. Diese Änderung überträgt das Adress-Muster 1:1 auf Geburtsdatum.

## Goals / Non-Goals

**Goals:**
- Geburtsdatum-Feld im Kontakt-Tab, verhält sich exakt wie Straße/PLZ/Ort (gleicher Save-Pfad, gleiche Drafts, gleiches Zurückziehen).
- Sinnvoller Erstwert: solange der Nutzer noch nichts explizit gesetzt hat, zeigt das Feld das aktuelle `members.date_of_birth` — kein leeres Pflichtfeld beim ersten Öffnen.
- Sichtbar nur dort, wo ein direkt verknüpfter User-Account existiert (eigenes Profil immer, Kind-Profil nur mit `user_contact`).

**Non-Goals:**
- Keine Format-/Plausibilitätsvalidierung (Mindestalter, Zukunftsdatum) — konsistent mit Adresse, die ebenfalls ungeprüft durchgereicht wird. `<input type="date">` liefert bereits ein syntaktisch gültiges Datum oder leer.
- Kein rückwirkendes Befüllen von `users.date_of_birth` für Bestandsnutzer (Migration fügt nur die Spalte hinzu, keine Backfill-Query) — der Default-Fallback im Frontend übernimmt die Anzeige, bis der Nutzer selbst einmal speichert.
- Keine Änderung am Mitgliedsdaten-Tab (`ProfileMemberTab`) außer der generischen `FIELD_LABELS`-Ergänzung für die Diff-Anzeige.

## Decisions

**users.date_of_birth als echte zweite Spalte (nicht nur Frontend-Fallback-State).**
Alternative wäre gewesen, komplett auf eine `users`-Spalte zu verzichten und das Feld direkt gegen `members.date_of_birth` zu initialisieren, ohne sofortigen Write (siehe Diskussion in der Exploration). Verworfen, weil explizit gewünscht: „Es soll sich verhalten wie bei der Adresse" — das schließt den sofortigen Self-Service-Write mit ein, nicht nur das Antrags-Verhalten.

**Default-Fallback lebt im Frontend, nicht im Backend.**
`GET /api/profile/me` liefert `date_of_birth` (aus `users`, ggf. leer) **und** `own_member.date_of_birth` (aus `members`) in derselben Response — beide Werte sind bereits vorhanden, ein Fallback im Backend würde nur unnötig verschleiern, welcher Wert „explizit" und welcher „geerbt" ist. Das Frontend entscheidet beim Initialisieren des State: `date_of_birth || own_member?.date_of_birth?.slice(0,10) || ''`. Für Kind-Profile analog mit `user_contact.date_of_birth` und `member.date_of_birth`.

**`profil`-Feldgruppe erweitern statt neue Draft-Feldgruppe einführen.**
`date_of_birth` wird ein weiterer Key innerhalb des bestehenden `field_name='profil'`-Drafts (`extractFieldValue`/`applyDraftToMember`), nicht ein eigener `field_name`. Das hält die Anzahl gleichzeitig offener Drafts pro Mitglied unverändert (ein Draft pro Kontakt-Formular-Save) und die generische Diff-Anzeige (`FIELD_LABELS` in `ProfileMemberTab`) funktioniert ohne Sonderfall.

**Keine Backfill-Migration.**
`ALTER TABLE users ADD COLUMN date_of_birth TEXT` ohne `UPDATE ... SET date_of_birth = (SELECT date_of_birth FROM members ...)`. Der Frontend-Fallback macht ein Backfill funktional überflüssig; ein Backfill hätte zusätzlich das Risiko, das Feld für alle Bestandsnutzer sofort als „explizit gesetzt" erscheinen zu lassen, obwohl es das nicht ist — genau die Unterscheidung, die der Fallback-Mechanismus eigentlich transportieren soll.

## Risks / Trade-offs

- **[Risiko]** Nutzer ändert das Geburtsdatum im Kontakt-Tab versehentlich (Tippfehler im `<input type="date">`) und löst einen unnötigen Vorstands-Änderungsantrag aus. → **Mitigation**: identisch zum bestehenden Verhalten bei Adresse; der Vorstand kann den Draft ablehnen, der Nutzer kann ihn selbst zurückziehen (bestehender „Zurückziehen"-Button in `ProfileMemberTab`).
- **[Risiko]** Divergenz zwischen `users.date_of_birth` (Self-Service) und `members.date_of_birth` (offiziell), solange ein Draft aussteht oder abgelehnt wurde. → **Mitigation**: bestehendes, akzeptiertes Verhalten bei Adresse; keine neue Eigenschaft dieser Änderung.
- **[Trade-off]** Zwei Kopien des Geburtsdatums (users + members) ohne weiteren Konsumenten der `users`-Kopie außer diesem Formular selbst. Bewusst in Kauf genommen, um exakte Konsistenz mit dem Adress-Verhalten zu erreichen (explizite Nutzer-Vorgabe).

## Migration Plan

1. Migration `039_users_date_of_birth.up.sql` / `.down.sql` (`ALTER TABLE users ADD COLUMN date_of_birth TEXT;` / Spalten-Rebuild ohne die Spalte für down, analog vorheriger `ALTER TABLE users ADD COLUMN`-Migrationen).
2. Backend-Erweiterungen (Response-Felder, Request-Felder, Draft-Feldgruppe) — rückwärtskompatibel, da neues Feld optional/leer bleibt, wenn nicht mitgeschickt.
3. Frontend-Erweiterungen.
4. Rollback: `make migrate-down` entfernt die Spalte; Frontend-Feld verschwindet mit dem zugehörigen Deploy — kein Datenverlust am Mitglieder-Record (`members.date_of_birth` bleibt unberührt).
