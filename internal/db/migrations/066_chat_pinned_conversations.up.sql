-- Pin-Status pro Nutzer für Chat-Konversationen (chat-pinned-conversations).
-- conversation_members ist bereits die (conversation_id, user_id)-Zeile — der
-- natürliche Ort für rein privaten Pin-Zustand, keine neue Tabelle nötig.
-- Beide NULL = nicht gepinnt (Default, keine Backfill-Logik für Bestandszeilen).
ALTER TABLE conversation_members ADD COLUMN pinned_at DATETIME;
ALTER TABLE conversation_members ADD COLUMN pin_order INTEGER;
