# stork-db-migrate

This program provides commands to initialize the Stork database and migrate the
database between selected versions. It is possible to migrate both up (from
lower to upper version) and down (from upper to lower version). This program
must be launched from the directory containing the database migration files.
The database migration files for Stork are located in backend/server/database/schema
directory. They must adhere to the following naming conventions:

```
1_<name of the migration to version 1>.up.sql
1_<name of the migration from version 1 to 0>.down.sql
2_<name of the migration from version 1 to 2>.up.sql
2_<name of the migration from version 2 to 1>.down.sql

and so on... 
```

For more usage details run:

```
stork-db-migrate -h
```

