DROP TABLE IF EXISTS answer_feedback;

ALTER TABLE policy_overrides
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS revoked_at,
    DROP COLUMN IF EXISTS revoked_by;
