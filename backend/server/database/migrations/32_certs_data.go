package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Sequence returning serial numbers used in PKI certificates.
             CREATE SEQUENCE IF NOT EXISTS certs_serial_number_seq;

             -- Table for storing PKI certifcates and keys
             CREATE TABLE IF NOT EXISTS secret (
                 name TEXT NOT NULL,
                 content TEXT NOT NULL,
                 CONSTRAINT secret_pkey PRIMARY KEY (name)
             );

             -- Add new columns in machine table that are used in agent authentication
             -- and authorization.
             ALTER TABLE machine ADD COLUMN IF NOT EXISTS agent_token TEXT;
             ALTER TABLE machine ADD COLUMN IF NOT EXISTS cert_fingerprint BYTEA;
             ALTER TABLE machine ADD COLUMN IF NOT EXISTS authorized BOOLEAN NOT NULL DEFAULT FALSE;
             CREATE INDEX machine_authorized_idx ON machine (authorized);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             DROP INDEX machine_authorized_idx;
             ALTER TABLE machine DROP COLUMN IF EXISTS authorized;
             ALTER TABLE machine DROP COLUMN IF EXISTS cert_fingerprint;
             ALTER TABLE machine DROP COLUMN IF EXISTS agent_token;
             DROP TABLE IF EXISTS secret;
             DROP SEQUENCE IF EXISTS certs_serial_number_seq;
        `)
		return err
	})
}
