-- Cleanup existing data before inserting new mock data
-- Generated: 2026-04-13 11:17:49

-- Delete all series_points
DELETE FROM series_points;

-- Delete all alerts
DELETE FROM alert;

-- Delete all slow queries
DELETE FROM slow_query;

-- Reset sequences (optional, to start IDs from 1)
ALTER SEQUENCE alert_id_seq RESTART WITH 1;
ALTER SEQUENCE slow_query_id_seq RESTART WITH 1;
