DROP INDEX IF EXISTS user_invitations_email_pending_idx;
DROP TABLE IF EXISTS user_invitations;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
