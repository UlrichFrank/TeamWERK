-- SQLite does not support DROP COLUMN in versions before 3.35.
-- This migration cannot be reversed automatically.
-- To roll back: recreate broadcast_reads without hidden_at column.
SELECT 1;
