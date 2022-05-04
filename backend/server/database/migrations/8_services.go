package dbmigs

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Service type is an enum which is used in the service
             -- table to indicate the type of the functionality that
             -- a group of applications provides, e.g. DHCP High Availability
             -- service. Based on this value, it is possible to determine
             -- where (in which SQL tables) to find any additional
             -- information about the service.
             CREATE TYPE SERVICETYPE AS ENUM
                 ('ha_dhcp');

             -- This enum is used in the ha_service table to indicate whether
             -- the given service is the DHCPv4 or DHCPv6 High Availability
             -- service.
             DO $$ BEGIN
                 CREATE TYPE HADHCPTYPE AS ENUM ('dhcp4', 'dhcp6');
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             -- This trigger function generates a default name for a new service.
             -- The name is only generated if the name specified by the
             -- user is blank.
             CREATE OR REPLACE FUNCTION service_name_gen()
                 RETURNS trigger
                 LANGUAGE 'plpgsql'
                 AS $function$
             BEGIN
                 -- This removes all whitespaces.
                 IF NEW.name IS NOT NULL THEN
	                 NEW.name = TRIM(NEW.name);
                 END IF;
                 IF NEW.name IS NULL OR NEW.name = '' THEN
                   NEW.name := 'service-' || to_char(NEW.id, 'FM0000000000');
                 END IF;
                 RETURN NEW;
             END;
             $function$;

             -- This table holds general information about services. A service
             -- groups multiple cooperating applications, running on distinct
             -- machines, and providing some desired function from the
             -- system administrator's perspective. Application-to-service is
             -- a many-to-many relationship where the app_to_service table describes
             -- this relationship.
             CREATE TABLE IF NOT EXISTS service (
                 id BIGSERIAL NOT NULL,
                 name TEXT COLLATE pg_catalog."default",
                 created TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 service_type SERVICETYPE,
                 CONSTRAINT service_pkey PRIMARY KEY (id),
                 CONSTRAINT service_name_unique_idx UNIQUE (name),
                 CONSTRAINT service_name_not_blank CHECK (name IS NOT NULL AND btrim(name) <> ''::text)
             );

             -- This generates a name for an inserted service if it is blank.
             DO $$ BEGIN
                 CREATE TRIGGER service_before_insert
                     BEFORE INSERT OR UPDATE ON service
                         FOR EACH ROW EXECUTE PROCEDURE service_name_gen();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

             -- This table includes details about the DHCP High Availability
             -- service. This table is in 1:1 relationship with the service table.
             -- A new entry should be created in this table if the type in the
             -- service table is set to ha_dhcp.
              CREATE TABLE IF NOT EXISTS ha_service (
                  id BIGSERIAL NOT NULL,
                  service_id BIGINT NOT NULL,
                  ha_type HADHCPTYPE NOT NULL,
                  ha_mode TEXT,
                  primary_id BIGINT,
                  secondary_id BIGINT,
                  primary_status_time TIME WITHOUT TIME ZONE,
                  secondary_status_time TIME WITHOUT TIME ZONE,
                  primary_last_state TEXT,
                  secondary_last_state TEXT,
                  backup_id BIGINT[],
                  CONSTRAINT ha_service_pkey PRIMARY KEY (id),
                  CONSTRAINT ha_service_id_unique UNIQUE (service_id),
                  CONSTRAINT ha_service_primary_id FOREIGN KEY (primary_id)
                      REFERENCES app (id) MATCH SIMPLE
                      ON UPDATE NO ACTION
                      ON DELETE NO ACTION,
                  CONSTRAINT ha_service_secondary_id FOREIGN KEY (secondary_id)
                      REFERENCES app (id) MATCH SIMPLE
                      ON UPDATE NO ACTION
                      ON DELETE NO ACTION,
                  CONSTRAINT ha_service_service_id FOREIGN KEY (service_id)
                      REFERENCES service (id) MATCH SIMPLE
                      ON UPDATE CASCADE
                      ON DELETE CASCADE);

              -- This automatically sets the service_type to ha_dhcp if inserting a
              -- new entry or updating an existing entry in the ha_service.
              CREATE OR REPLACE FUNCTION ha_service_type_set()
                  RETURNS trigger
                  LANGUAGE 'plpgsql'
                  AS $function$
              BEGIN
                  UPDATE service SET service_type = 'ha_dhcp' WHERE id = NEW.service_id;
                  RETURN NEW;
              END;
              $function$;

             DO $$ BEGIN
                 CREATE TRIGGER ha_service_before_insert_update
                     BEFORE INSERT OR UPDATE ON ha_service
                         FOR EACH ROW EXECUTE PROCEDURE ha_service_type_set();
             EXCEPTION
                 WHEN duplicate_object THEN null;
             END $$;

              -- This describes the M:M relationship between the app and service tables.
              CREATE TABLE IF NOT EXISTS app_to_service (
                  app_id BIGINT NOT NULL,
                  service_id bigint NOT NULL,
                  CONSTRAINT app_to_service_pkey PRIMARY KEY (app_id, service_id),
                  CONSTRAINT app_to_service_app_id FOREIGN KEY (app_id)
                      REFERENCES app (id) MATCH SIMPLE
                      ON UPDATE NO ACTION
                      ON DELETE CASCADE,
                  CONSTRAINT app_to_service_service_id FOREIGN KEY (service_id)
                      REFERENCES service (id) MATCH SIMPLE
                      ON UPDATE NO ACTION
                      ON DELETE CASCADE
               );

               -- This creates an index for searching apps using control address and port.
               CREATE INDEX app_ctrl_address_port_idx ON app (ctrl_address, ctrl_port);
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
               DROP TABLE IF EXISTS app_to_service;
               DROP TABLE IF EXISTS ha_service;
               DROP TRIGGER IF EXISTS service_before_insert ON service;
               DROP TABLE IF EXISTS service;
               DROP FUNCTION IF EXISTS service_name_gen;
               DROP TYPE servicetype;
               DROP TYPE hadhcptype;
           `)
		return err
	})
}
