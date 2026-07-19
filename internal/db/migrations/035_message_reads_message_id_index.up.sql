-- Read-Receipts (Absender-Sicht): message_reads.read_at existiert bereits seit
-- Migration 001 (NOT NULL DEFAULT CURRENT_TIMESTAMP) — es fehlt nur ein Index auf
-- message_id. Er beschleunigt die Reader-Liste (GET /api/chat/messages/{id}/reads)
-- und das readCount-Aggregat pro Nachricht in der Message-Listen-Abfrage; der
-- bestehende idx_message_reads_user deckt nur user_id ab.
CREATE INDEX IF NOT EXISTS idx_message_reads_message_id ON message_reads(message_id);
