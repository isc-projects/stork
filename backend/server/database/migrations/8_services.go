package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Service type is an enum which is used in the service
             -- table to indicate the type of the functionality that
             -- a group applications provide, e.g. DHCP High Availability
             -- service. Based on this value it is possible to determine
             -- where to find (in what SQL tables) any additional
             -- information about the service.
             CREATE TYPE SERVICETYPE AS ENUM
                 ('ha_dhcp');

             -- This enum is used in the ha_service table to indicate if
             -- the given service is the DHCPv4 or DHCPv6 High Availability
             -- service.
             CREATE TYPE HADHCPTYPE AS ENUM
                 ('dhcp4', 'dhcp6');

             -- Trigger function generating a default label for a new service.
             -- The label is only generated if the label specified by the
             -- user is blank.
             CREATE OR REPLACE FUNCTION service_label_gen()
                 RETURNS trigger
                 LANGUAGE 'plpgsql'
                 AS $function$
             BEGIN
                 -- Remove all of the whitespaces.
                 IF NEW.label IS NOT NULL THEN
	                 NEW.label = TRIM(NEW.label);
                 END IF;
                 IF NEW.label IS NULL OR NEW.label = '' THEN
                   NEW.label := 'service-' || to_char(NEW.id, 'FM0000000000');
                 END IF;
                 RETURN NEW;
             END;
             $function$;

             -- This table holds general information about services. A service
             -- groups multiple cooperating applications running on distinct
             -- machines and providing some desired function from the
             -- system administrator's perspective. Application to service is
             -- a many-to-many relationship where app_to_service table provides
             -- this relationship.
             CREATE TABLE IF NOT EXISTS service (
                 id BIGSERIAL NOT NULL,
                 label TEXT COLLATE pg_catalog."default",
                 created TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 service_type SERVICETYPE,
                 CONSTRAINT service_pkey PRIMARY KEY (id),
                 CONSTRAINT service_label_unique_idx UNIQUE (label),
                 CONSTRAINT service_label_not_blank CHECK (label IS NOT NULL AND btrim(label) <> ''::text)
             );

             -- Generate a label for an inserted service if it is blank.
             CREATE TRIGGER service_before_insert
                 BEFORE INSERT OR UPDATE ON service
                    FOR EACH ROW EXECUTE PROCEDURE service_label_gen();

             -- This table includes a details about the DHCP High Availability
             -- service. This table is in 1:1 relationship with the service table.
             -- A new entry should be created in this table of the ha_type in the
             -- service table is set to ha_dhcp.
             CREATE TABLE IF NOT EXISTS ha_service (
                 id BIGSERIAL NOT NULL,
                 service_id bigint NOT NULL,
                 ha_type hadhcptype NOT NULL,
                 CONSTRAINT ha_service_pkey PRIMARY KEY (id),
                 CONSTRAINT ha_service_id_unique UNIQUE (service_id),
                 CONSTRAINT ha_service_service_id FOREIGN KEY (service_id)
                    REFERENCES service (id) MATCH SIMPLE
                    ON UPDATE CASCADE
                    ON DELETE CASCADE
              );

              -- Automatically set service_type to ha_dhcp if inserting the
              -- new entry or updating existing entry in the ha_service.
              CREATE OR REPLACE FUNCTION ha_service_type_set()
                  RETURNS trigger
                  LANGUAGE 'plpgsql'
                  AS $function$
              BEGIN
                  UPDATE service SET service_type = 'ha_dhcp' WHERE id = NEW.service_id;
                  RETURN NEW;
              END;
              $function$;

             CREATE TRIGGER ha_service_before_insert_update
                 BEFORE INSERT OR UPDATE ON ha_service
                    FOR EACH ROW EXECUTE PROCEDURE ha_service_type_set();

              -- Provides M:M relationship between app and service tables.
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

               -- Create index for searching apps using control address and port.
               CREATE INDEX app_ctrl_address_port_idx ON app (ctrl_address, ctrl_port);
           `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
               DROP TABLE IF EXISTS app_to_service;
               DROP TABLE IF EXISTS ha_service;
               DROP TRIGGER IF EXISTS service_before_insert ON service;
               DROP TABLE IF EXISTS service;
               DROP FUNCTION IF EXISTS service_label_gen;
               DROP TYPE servicetype;
               DROP TYPE hadhcptype;
           `)
		return err
	})
}
