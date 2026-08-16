-- Event-Log (event-log): je Empfänger eine Zeile pro versandter Meldung.
--
-- Geschrieben wird in notify.Send, VOR und unabhängig von jeder
-- Präferenz-Filterung. notification_preferences steuert damit ausschließlich
-- den Zustellkanal (Push/Email), niemals die Sichtbarkeit hier. Wer alle
-- Pushes abgeschaltet hat, findet die Meldung trotzdem im Log — das ist der
-- ganze Zweck der Tabelle.
--
-- Bewusst KEIN Fremdschlüssel auf ein Domänen-Objekt: die Meldung wird zum
-- Sendezeitpunkt eingefroren, weil die referenzierten Objekte (gelöschte
-- Termine, entfernte Dienst-Slots) danach nicht mehr existieren — genau die
-- Fälle, in denen Nachlesen am wichtigsten ist. `url` ist ein Sprungziel,
-- kein Fremdschlüssel, und darf ins Leere zeigen.
--
-- Der CHECK listet die acht Nicht-Chat-Kategorien. Chat und Mitteilungen
-- laufen über push.SendToUserWithBadge, haben eigene Ungelesen-Zähler und den
-- App-Icon-Badge; sie gehören bewusst nicht hierher. Wer das ändern will,
-- schreibt eine Migration — die richtige Reibung für eine Scope-Entscheidung.
-- Nebeneffekt: Tippfehler in `category` fallen hier auf, statt still dazu zu
-- führen, dass FilterByPushPref niemanden matcht.

CREATE TABLE user_events (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category   TEXT     NOT NULL
               CHECK (category IN ('games','trainings','duties','duty_reminders',
                                   'carpooling','membership','operativ','sonstiges')),
    title      TEXT     NOT NULL,
    body       TEXT     NOT NULL DEFAULT '',
    url        TEXT     NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    seen_at    DATETIME
);

-- Dashboard-Read: die 30 jüngsten Zeilen eines Nutzers.
CREATE INDEX idx_user_events_user_created ON user_events(user_id, created_at DESC);

-- Retention-Purge: beide Zweige (seen_at + 3 Tage / created_at + 90 Tage).
CREATE INDEX idx_user_events_retention ON user_events(seen_at, created_at);
