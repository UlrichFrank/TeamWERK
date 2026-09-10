-- Rollback von 059_chat_polls: Tabellen in umgekehrter Reihenfolge droppen.
-- Bestehende Umfrage-Nachrichten bleiben als reine Textnachricht (die Frage)
-- im Verlauf stehen — kein Datenverlust außerhalb der Umfrage selbst.
DROP INDEX idx_chat_poll_votes_user;
DROP TABLE chat_poll_votes;
DROP TABLE chat_poll_options;
DROP TABLE chat_polls;
