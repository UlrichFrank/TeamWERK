CREATE TABLE member_change_drafts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  member_id INTEGER NOT NULL,
  field_name VARCHAR(50) NOT NULL,
  old_value TEXT NOT NULL,
  new_value TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by_user_id INTEGER,

  FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE,
  FOREIGN KEY (created_by_user_id) REFERENCES users(id),
  UNIQUE(member_id, field_name)
);

CREATE INDEX idx_member_change_drafts_member_id ON member_change_drafts(member_id);
