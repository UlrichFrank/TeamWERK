-- ON DELETE RESTRICT auf duty_slots.duty_type_id → falls Slots existieren,
-- schlägt der Rückbau bewusst fehl. User muss Slots vorher entfernen.

DELETE FROM duty_types WHERE name = 'Spielbericht';
