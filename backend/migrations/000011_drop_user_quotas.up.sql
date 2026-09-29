-- 000011: drop the unenforced per-user quota columns.
--
-- max_sandboxes and max_builds_per_hour were never read by any enforcement
-- path: nothing limited environment creation or build rate by them. The only
-- consumer was the profile page, which rendered them with an "infinite"
-- fallback, so they always displayed as unbounded anyway.

ALTER TABLE users DROP COLUMN IF EXISTS max_sandboxes;
ALTER TABLE users DROP COLUMN IF EXISTS max_builds_per_hour;
