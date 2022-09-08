INSERT INTO hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) VALUES (decode('010101010101', 'hex'), 0, 1, '192.0.2.42'::inet - '0.0.0.0'::inet);
