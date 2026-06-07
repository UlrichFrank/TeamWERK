ALTER TABLE messages ADD COLUMN reply_to_id INTEGER REFERENCES messages(id);
ALTER TABLE messages ADD COLUMN edited_at DATETIME;
ALTER TABLE messages ADD COLUMN deleted_at DATETIME;
ALTER TABLE broadcasts ADD COLUMN edited_at DATETIME;
