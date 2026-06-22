-- Best-effort Down: die ursprünglich vor 006 gesetzten NULL-Werte sind nach dem
-- Up-Backfill nicht mehr von echten User-Wahlen zu unterscheiden. Diese Down
-- ist ein No-Op; ein vollständiges Rollback des Backfills ist nicht möglich,
-- ohne separat protokollierte Vor-Werte (die diese Migration nicht erhebt).
SELECT 1;
