-- The "down" migration file. It contains SQL to reverse the changes
-- made in the "up" migration, which is essential for rollbacks.

DROP TABLE IF EXISTS chapters;
DROP TABLE IF EXISTS series;
