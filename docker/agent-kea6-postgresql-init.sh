#!/bin/bash

echo "Checking if the database exists"
export PGPASSWORD=kea && psql -U kea -h 172.20.0.116 -d kea -c "select * from schema_version";

if [ $? -eq 0 ]
then
    echo "Database apparently exists"
    exit 0
fi

set -e

echo "Initializing the database"
kea-admin db-init pgsql -u kea -p kea -n kea -h 172.20.0.116

export PGPASSWORD=kea && psql -U kea -h 172.20.0.116 -d kea <<EOF

delete from lease6;
insert into lease6(address, duid, valid_lifetime, expire, subnet_id, pref_lifetime, lease_type, iaid, prefix_len, hwtype, hwaddr_source, state) values('3001:db8:1::1', DECODE('0002000000090c1fef1fef1fef', 'hex'), 3600, NOW() + interval '1' MONTH, 1, 1800, 0, 1, 128, 0, 0, 0);
insert into lease6(address, duid, valid_lifetime, expire, subnet_id, pref_lifetime, lease_type, iaid, prefix_len, hwtype, hwaddr_source, state) values('3001:db8:1::2', DECODE('00', 'hex'), 3600, NOW() + interval '1' MONTH, 1, 1800, 0, 1, 128, 0, 0, 1);
insert into lease6(address, duid, valid_lifetime, expire, subnet_id, pref_lifetime, lease_type, iaid, prefix_len, hwtype, hwaddr_source, state) values('3001:db8:8:10::', DECODE('0002000000090c1fef1fef1fef', 'hex'), 3600, NOW() + interval '1' MONTH, 1, 1800, 2, 1, 64, 0, 0, 0);


EOF
