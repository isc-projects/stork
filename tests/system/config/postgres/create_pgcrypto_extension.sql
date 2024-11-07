-- The system tests use a Postgres Docker container that self-create an empty
-- database. There is no stork-tool db-create call, so the database schema is
-- initialized on the server startup.
-- In some cases, we run the server with the non-superadmin database user, so
-- creating an extension fails during the database migration.
CREATE EXTENSION pgcrypto;
