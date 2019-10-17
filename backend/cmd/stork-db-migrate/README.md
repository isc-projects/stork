# stork-db-migrate

This program provides commands to initialize the Stork database and migrate the
database between selected versions. It is possible to migrate both up (from
lower to upper version) and down (from upper to lower version). The migrations
are written in individual .go files and must be placed in the
backend/server/database/migrations directory.

This program prompts for the database user password. It is possible to set
the password in the STORK_DATABASE_PASSWORD in which case the user won't be
prompted. This is particularly useful for testing purposes.


