package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
	     	_, err := db.Exec(
	     		`-- Login is convenient alternative to email. Also, the default
             CREATE TABLE public.machine (
                 id                      serial,
	         created                 timestamp without time zone not null default (now() at time zone 'utc'),
	         deleted                 timestamp without time zone,
                 address                 varchar(255),
	         agent_version           varchar(255),
	         cpus                    integer,
	         cpus_load               varchar(30),
	         memory                  integer,
	         hostname                varchar(255),
 	         uptime                  integer,
	         used_memory             integer,
	         os                      varchar(30),
	         platform                varchar(30),
	         platform_family         varchar(30),
	         platform_version        varchar(30),
	         kernel_version          varchar(30),
	         kernel_arch             varchar(10),
	         virtualization_system   varchar(30),
	         virtualization_role     varchar(20),
	         host_id                 varchar(40),
	         last_visited            timestamp without time zone,
	         error                   varchar(255)
             );`)
		return err

	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Remove trigger hashing passwords.
             DROP TABLE public.machine;`)
		return err
	})
}
