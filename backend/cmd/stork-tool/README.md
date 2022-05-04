# stork-tool

This program provides commands to 1) initialize the Stork database and migrate the
database between selected versions, and 2) inspect and export server keys and certificates.

It is possible to migrate both up (from an older to a newer version) and
down (from a newer to an older version). The migrations are written in
individual .go files and must be placed in the
backend/server/database/migrations directory.

This program prompts for the database user's password. It is possible to set
the password in the STORK_DATABASE_PASSWORD variable, in which case the user is not
prompted. This is particularly useful for testing purposes.
