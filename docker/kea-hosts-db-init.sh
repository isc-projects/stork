#!/bin/bash

echo "Checking if the database exists"
mysql --user=kea --password=kea --host=172.20.0.104 kea -e "select * from schema_version"

if [ $? -eq 0 ]
then
    echo "Database apparently exists"
    exit 0
fi

set -e

echo "Initializing the database"
kea-admin db-init mysql -u kea -p kea -n kea -h 172.20.0.104

mysql --user=kea --password=kea --host=172.20.0.104 kea <<EOF
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('010101010101'), 0, 123, inet_aton('192.0.2.230'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('020202020202'), 0, 123, inet_aton('192.0.2.231'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('030303030303'), 0, 123, inet_aton('192.0.2.232'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('040404040404'), 0, 123, inet_aton('192.0.2.233'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('050505050505'), 0, 123, inet_aton('192.0.2.234'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('060606060606'), 0, 123, inet_aton('192.0.2.235'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('07070707'), 2, 123, inet_aton('192.0.2.236'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('08080808'), 2, 123, inet_aton('192.0.2.237'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('09090909'), 1, 123, inet_aton('192.0.2.238'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('0a0a0a0a'), 2, 123, inet_aton('192.0.2.239'));
EOF
