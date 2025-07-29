insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('010101010101'), 0, 123, inet_aton('192.110.111.230'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address, hostname) values (unhex('020202020202'), 0, 123, inet_aton('192.110.111.231'), 'fish.example.org');
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address, hostname) values (unhex('030303030303'), 0, 123, inet_aton('192.110.111.232'), 'gibberish');
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('040404040404'), 0, 123, inet_aton('192.110.111.233'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('050505050505'), 0, 123, inet_aton('192.110.111.234'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('060606060606'), 0, 123, inet_aton('192.110.111.235'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('07070707'), 2, 123, inet_aton('192.110.111.236'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('08080808'), 2, 123, inet_aton('192.110.111.237'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('09090909'), 1, 123, inet_aton('192.110.111.238'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('0a0a0a0a'), 2, 123, inet_aton('192.110.111.239'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('080808080808'), 0, 0, inet_aton('192.110.111.240'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('090909090909'), 0, 0, inet_aton('192.110.111.241'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp4_subnet_id, ipv4_address) values (unhex('0a0a0a0a0a0a'), 0, 0, inet_aton('192.110.111.242'));
insert into hosts(dhcp_identifier, dhcp_identifier_type, dhcp6_subnet_id) values (unhex('abc76efabdeaae'), 1, 1);

select host_id from hosts where ipv4_address = inet_aton('192.110.111.230') into @selected_host;
insert into dhcp4_options(code, formatted_value, space, persistent, host_id, scope_id) values(14, '/tmp/dump/dhcp', 'dhcp4', 0, @selected_host, 3);
insert into dhcp4_options(code, formatted_value, space, persistent, host_id, scope_id) values(3, '10.2.12.1', 'dhcp4', 1, @selected_host, 3);
insert into dhcp4_options(code, formatted_value, space, persistent, host_id, scope_id) values(20, 'true', 'dhcp4', 0, @selected_host, 3);

select host_id from hosts where ipv4_address = inet_aton('192.110.111.242') into @selected_host;
insert into dhcp4_options(code, formatted_value, space, persistent, host_id, scope_id) values(20, 'true', 'dhcp4', 0, @selected_host, 3);

select host_id from hosts where hex(dhcp_identifier) = 'abc76efabdeaae' into @selected_host;
insert into dhcp6_options(code, formatted_value, space, persistent, host_id, scope_id) values(23, '2001:db8:1::1,2001:db8:1::1', 'dhcp6', 1, @selected_host, 3);
insert into dhcp6_options(code, formatted_value, space, persistent, host_id, scope_id) values(51, 'foo.example.org.', 'dhcp6', 1, @selected_host, 3);
