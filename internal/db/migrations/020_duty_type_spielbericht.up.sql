-- Duty-Type „Spielbericht": pro Heim-/Auswärts-Event ein Slot mit
-- Deadline event-ende + 24h (kein Cash-Substitute, hours_value=0.5 —
-- übliches Bericht-Aufkommen im Presseteam).
--
-- target_role ist syntaktisch 'elternteil' — der eigentliche
-- Presseteam-Filter läuft im duty-board-Handler (Sichtbarkeit) und
-- im slot-Ziehen-Handler (Backend-Guard); duty_types.target_role bleibt
-- die Vereinsfunktions-Achse und wird für diesen Slot nicht als Filter
-- verwendet.
--
-- same_day_behavior='normal': keine Reduktion bei mehreren Events am Tag
-- (jeder Bericht ist eigenständig).

INSERT INTO duty_types (
    name, hours_value, cash_substitute, default_anchor,
    default_offset_minutes, target_role,
    consecutive_behavior, same_day_behavior, adjacent_day_behavior
) VALUES (
    'Spielbericht', 0.5, NULL, 'end',
    1440, 'elternteil',
    'normal', 'normal', 'normal'
);
