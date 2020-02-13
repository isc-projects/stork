package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
             -- Create a table of shared networks. Multiple subnets may belong
             -- to a single shared network. The shared network groups the subnets
             -- together.
             CREATE TABLE IF NOT EXISTS shared_network (
                 id bigserial NOT NULL,
                 created TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 name text NOT NULL,
                 CONSTRAINT shared_network_pkey PRIMARY KEY (id)
             );

             -- Create a table of subnets. It holds both IPv4 and IPv6 subnets.
             -- A subnet may belong to a shared network. If it doesn't, the
             -- shared_network_id is set to null.
             CREATE TABLE IF NOT EXISTS subnet (
                 id bigserial NOT NULL,
                 created TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 prefix cidr NOT NULL,
                 shared_network_id bigint,
                 CONSTRAINT subnet_pkey PRIMARY KEY (id),
                 CONSTRAINT subnet_shared_network_fkey FOREIGN KEY (shared_network_id)
                     REFERENCES shared_network (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE SET NULL
             );

             -- It is common to select subnet by prefix.
             CREATE INDEX subnet_prefix_idx ON subnet(prefix);

             -- Create a table of pools. A pool always belongs to a subnet. The pool
             -- specification consists of a lower_bound and upper_bound address, which
             -- designates the first and the last address belonging to the pool.
             CREATE TABLE IF NOT EXISTS address_pool (
                 id bigserial NOT NULL,
                 created TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 lower_bound inet NOT NULL,
                 upper_bound inet NOT NULL,
                 subnet_id bigint NOT NULL,
                 CONSTRAINT address_pool_pkey PRIMARY KEY (id),
                 CONSTRAINT address_pool_subnet_fkey FOREIGN KEY (subnet_id)
                     REFERENCES subnet (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE CASCADE,
                 CONSTRAINT address_pool_lower_upper_family_check
                     CHECK (family(lower_bound) = family(upper_bound)),
                 CONSTRAINT address_pool_lower_upper_check CHECK (lower_bound <= upper_bound)
             );

             -- Create a table with pools of delegated prefixes. The prefix pool always
             -- belongs to an IPv6 subnet. The delegated_prefix designates the length of
             -- the prefix returned to the client as a result of prefix delegation request.
             CREATE TABLE IF NOT EXISTS prefix_pool (
                 id bigserial NOT NULL,
                 created TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT timezone('utc'::text, now()),
                 prefix cidr NOT NULL,
                 delegated_len smallint NOT NULL,
                 subnet_id bigint NOT NULL,
                 CONSTRAINT prefix_pool_pkey PRIMARY KEY (id),
                 CONSTRAINT prefix_pool_subnet_fkey FOREIGN KEY (subnet_id)
                     REFERENCES subnet (id) MATCH SIMPLE
                         ON UPDATE CASCADE
                         ON DELETE CASCADE,
                 CONSTRAINT prefix_pool_delegated_len_check
                     CHECK (delegated_len > 0 AND delegated_len <= 128),
                 CONSTRAINT prefix_pool_ipv6_only_check CHECK (family(prefix::inet) = 6)
             );

             -- Provides M:M relationship between app and subnet tables.
             CREATE TABLE IF NOT EXISTS app_to_subnet (
                 app_id bigint NOT NULL,
                 subnet_id bigint NOT NULL,
                 local_subnet_id bigint,
                 CONSTRAINT app_to_subnet_pkey PRIMARY KEY (app_id, subnet_id),
                 CONSTRAINT app_to_subnet_app_id FOREIGN KEY (app_id)
                     REFERENCES app (id) MATCH SIMPLE
                     ON UPDATE NO ACTION
                     ON DELETE CASCADE,
                 CONSTRAINT app_to_subnet_subnet_id FOREIGN KEY (subnet_id)
                     REFERENCES subnet (id) MATCH SIMPLE
                     ON UPDATE NO ACTION
                     ON DELETE CASCADE
             );
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             DROP TABLE IF EXISTS app_to_subnet;
             DROP TABLE IF EXISTS prefix_pool;
             DROP TABLE IF EXISTS address_pool;
             DROP TABLE IF EXISTS subnet;
             DROP TABLE IF EXISTS shared_network;
        `)
		return err
	})
}
