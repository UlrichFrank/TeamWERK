## Designentscheidungen

### 1. Neue Spalte `recovery_email`, nicht Wiederverwendung von `users.email`

Die Eltern-E-Mail in `users.email` des Kindes abzulegen scheitert an zwei Stellen:

- **Unique-Index** `users_email_login_unique WHERE can_login=1 AND email IS NOT NULL`: Zwei Geschwister teilen dieselbe Eltern-E-Mail → Kollision, sobald beide aktiv sind.
- **Login-Ambiguität**: Login matcht `email OR login_name` bei `can_login=1`. Wäre `child.email = parent.email`, träfe ein Login mit dieser Adresse **Eltern UND Kind**.

Deshalb eine **separate Spalte**, die per Definition **keine** Login-Identität ist: kein Unique-Index, nie Lookup-Key. Name `recovery_email` beschreibt die Funktion (Passwort-Wiederherstellung); im UI deutsch „Eltern-E-Mail (für Passwort-Reset)".

### 2. Zwei Qualitäten von E-Mail (das verbindende Modell)

Jedes Konto hat bis zu zwei E-Mail-Rollen; Erwachsene fallen beide auf dieselbe Spalte zusammen:

| Qualität | Rolle | Erwachsener | Kind |
|---|---|---|---|
| AccountName / Nutzeremail | Login-Identität, Lookup-Key | `users.email` | `users.login_name` |
| Wiederherstellungs-E-Mail | Ziel für Passwort-Mails | `users.email` (gleich) | `users.recovery_email` |

Forgot-Password ändert dadurch sein Verhalten **nicht** — es trennt nur die beiden Qualitäten:
- Lookup über Qualität 1: `WHERE (LOWER(email)=LOWER(?) OR LOWER(login_name)=LOWER(?)) AND can_login=1`.
- Versand an Qualität 2: `COALESCE(NULLIF(email,''), recovery_email)`.

`recovery_email` ist **nie** Lookup-Key: Tippt jemand die Eltern-E-Mail ein, trifft das den Eltern-Account (dessen `email`), nicht das Kind. Das Kind muss seinen `login_name` verwenden.

### 3. Doppelte Bestätigung ALT → NEU (die zentrale Mechanik)

Eine einzelne Bestätigung reicht nicht — beide Eigenschaften werden gebraucht:

```
confirm at OLD  → „aktueller Inhaber autorisiert"  (Anti-Hijack)
confirm at NEW  → „neue Adresse ist erreichbar"    (Anti-Lockout)
```

Sequenz, **ALT zuerst**:

```
initiate (neue Adresse X)
  token: field=recovery_email, new_email=X, stage=auth
        │
        ▼  ① Mail an ALTE recovery_email: „Auf X ändern? [Bestätigen]"
        │     Klick (stage=auth gültig) → stage=verify, Token rotiert
        ▼  ② Mail an NEUE Adresse X: „Bestätige Wiederherstellungs-Adresse [Bestätigen]"
        │     Klick (stage=verify gültig)
        ▼  ✓ users.recovery_email = X, used_at gesetzt
```

Warum **ALT zuerst** (nicht gleichzeitig, nicht NEU zuerst):

- **Anti-Hijack greift an ①.** Ein Kind, das die Änderung anstößt, kommt nicht über ① hinaus — es kann den Link im Eltern-Postfach nicht klicken. Vor der Autorisierung wird **nichts** an die neue Adresse gesendet (kein Missbrauch zum Anpingen fremder Adressen).
- **Anti-Lockout greift an ②.** Die neue Adresse muss sich als erreichbar beweisen, bevor sie Ziel wird. Tippfehler → ②-Link kommt nie an → Änderung schließt nie ab, alte Adresse bleibt intakt.
- **Der Tote-Mailbox-Fall fällt von selbst heraus.** Existiert die alte Adresse nicht mehr, kann ① nie geklickt werden → der Loop stockt → genau die „deaktivierter Email-Account"-Situation → Admin/Vorstand-Override (siehe 5).

**Bewusster Trade-off:** Bestätigt der Inhaber an ① und vertippt sich gleichzeitig in X, kommt ② nie an — die Änderung schließt nicht ab und nichts geht kaputt (alte Adresse bleibt). Ein echter stiller Lockout entsteht nur, wenn die *alte* Adresse zwischen Anstoß und Abschluss stirbt — dann greift der Override.

### 4. Reuse von `email_change_tokens` mit `field` + `stage`

Statt einer neuen Tabelle: additive Spalten.

- `field TEXT NOT NULL DEFAULT 'email'` — `'email'` (Erwachsenen-Flow, unverändert) | `'recovery_email'`.
- `stage TEXT` (nullable) — `NULL` = klassischer einstufiger Flow; `'auth'` / `'verify'` für den zweistufigen Recovery-Flow.

Der Erwachsenen-`ConfirmEmailChange` bleibt unangetastet (seine Zeilen haben `field='email'`, `stage=NULL`). Der neue `ConfirmRecoveryEmailChange` ist eine eigene Route, die nur `field='recovery_email'`-Zeilen bedient und über `stage` den Übergang steuert. Token-Rotation beim Stufenwechsel: alter Token `used_at` setzen, neuen opaken Token erzeugen (eindeutiger `token`-Constraint bleibt erfüllt), `expires_at` neu setzen.

### 5. Admin/Vorstand-Direkt-Override — Escape-Hatch, nicht maschinell erkannt

„Deaktivierter Email-Account" heißt: die alte Registrierungs-Adresse existiert nicht mehr, der Bestätigungs-Loop ist tot. Das ist **menschliches Urteil**, kein per SQL erkennbarer Zustand. Daher in Code schlicht: **Admin/Vorstand haben immer ein Direkt-Schreib-Endpoint** (`PUT /api/users/{id}/recovery-email`, kein Loop); alle anderen nur den Loop. Sie nutzen es genau in der Tote-Mailbox-Situation, nachdem sie die Identität außerhalb des Systems geprüft haben.

Der Override feuert **keine** Verifikation an die neue Adresse (bewusst, „ohne Workflow" wie gefordert) — die Verantwortung liegt beim Admin/Vorstand.

### 6. Kind kann nicht ändern, nur lesen

Das Self-Edit `PUT /api/profile/account` nimmt `recovery_email` **nicht** ins DTO/SQL auf — ein mitgeschicktes Feld wird ignoriert. Anzeige (read-only) im eigenen Kindprofil und im eingeblendeten Kindprofil der Eltern. Die einzigen Schreibpfade sind der Eltern-Loop (3) und der Admin/Vorstand-Override (5).

### 7. Scope: Erwachsenen-Flow `email-aenderung` bleibt unverändert

Die Doppelbestätigung ALT→NEU gilt ausschließlich für `recovery_email`. Der bestehende Erwachsenen-E-Mail-Wechsel (`POST /api/profile/email`, Bestätigung an die **neue** Adresse, Passwort-Gate) bleibt wie er ist — er hat ein anderes Bedrohungsmodell (Nutzer ändert eigene Adresse, kein Fremd-Kind-Hijack). Eine Vereinheitlichung ist explizit **nicht** Teil dieses Proposals.

### 8. Approval-Wiring & Backfill

- `approveChildRequest` schreibt beim Anlegen `recovery_email = parent_email` (statt sie nach dem Setup-Mail-Versand zu verwerfen).
- Migration backfillt Bestandskinder aus `membership_requests.parent_email`, soweit eindeutig dem Konto zuordenbar. Nicht zuordenbare bleiben `NULL` → für diese muss Admin/Vorstand den Override nutzen.
