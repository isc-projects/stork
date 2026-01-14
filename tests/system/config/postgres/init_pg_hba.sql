-- Script modifies the pg_hba.conf file to accept different authentication
-- methods.
-- See: https://www.dbi-services.com/blog/modifying-pg_hba-conf-from-inside-postgresql/

-- Configure pg_hba.conf file.
CREATE TEMPORARY TABLE hba ( line text );
DO $$
BEGIN
    EXECUTE format('COPY hba FROM %L', current_setting('hba_file'));
END $$;
DELETE FROM hba WHERE line ~* '^host\s+all.*$';
INSERT INTO hba (line) VALUES
--    TYPE  DB   USER                         ADDR METHOD
    ('host  all  stork_trust                  all  trust'),
    ('host  all  stork_md5                    all  md5'),
    ('host  all  stork_scram-sha-256          all  scram-sha-256'),
    ('host  all  root                         all  ident'),
    ('host  all  all                          all  md5');
DO $$
BEGIN
    EXECUTE format('COPY hba TO %L', current_setting('hba_file'));
END $$;
SELECT pg_reload_conf();

-- Create users.
CREATE USER stork_trust;
CREATE USER stork_md5 WITH PASSWORD 'stork_md5';
CREATE USER root;
CREATE USER "stork_scram-sha-256";
-- Default encryption is md5. Altering password encryption for a specific user
-- doesn't work for Postgres 11.
SET password_encryption = 'scram-sha-256';
ALTER USER "stork_scram-sha-256" WITH PASSWORD 'stork_scram-sha-256';

-- Grant all privileges on the public schema.
GRANT ALL PRIVILEGES ON SCHEMA public TO stork_trust;
GRANT ALL PRIVILEGES ON SCHEMA public TO stork_md5;
GRANT ALL PRIVILEGES ON SCHEMA public TO root;
GRANT ALL PRIVILEGES ON SCHEMA public TO "stork_scram-sha-256";
