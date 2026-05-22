ALTER TABLE games ADD COLUMN template_id INTEGER REFERENCES game_templates(id);
