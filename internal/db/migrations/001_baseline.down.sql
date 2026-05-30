-- Drop all views and tables (reverse dependency order)
PRAGMA foreign_keys = OFF;

DROP VIEW IF EXISTS user_accessible_teams;
DROP VIEW IF EXISTS team_memberships;

DROP TABLE IF EXISTS carpooling_events;
DROP TABLE IF EXISTS mitfahrt_paarungen;
DROP TABLE IF EXISTS mitfahrgelegenheiten;
DROP TABLE IF EXISTS game_teams;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS game_template_items;
DROP TABLE IF EXISTS game_templates;
DROP TABLE IF EXISTS duty_season_targets;
DROP TABLE IF EXISTS duty_accounts;
DROP TABLE IF EXISTS duty_assignments;
DROP TABLE IF EXISTS duty_slots;
DROP TABLE IF EXISTS duty_types;
DROP TABLE IF EXISTS kader_trainers;
DROP TABLE IF EXISTS kader_members;
DROP TABLE IF EXISTS kader;
DROP TABLE IF EXISTS team_trainers;
DROP TABLE IF EXISTS vehicle_info;
DROP TABLE IF EXISTS membership_requests;
DROP TABLE IF EXISTS member_change_drafts;
DROP TABLE IF EXISTS family_links;
DROP TABLE IF EXISTS members;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS user_visibility;
DROP TABLE IF EXISTS user_phones;
DROP TABLE IF EXISTS email_change_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS invitation_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS age_class_game_rules;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS seasons;
DROP TABLE IF EXISTS clubs;

PRAGMA foreign_keys = ON;
