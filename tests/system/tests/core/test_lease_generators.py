from datetime import datetime
import tempfile

from core.lease_generators import gen_dhcp4_lease_file, gen_dhcp6_lease_file


def test_gen_dhcp4_lease_file():
    expected = """address,hwaddr,client_id,valid_lifetime,expire,subnet_id,fqdn_fwd,fqdn_rev,hostname,state,user_context
192.0.2.1,,,600,642,1,1,1,host-1.example.org,1,
192.0.2.2,00:01:02:03:04:02,01:02:03:02,600,642,1,1,1,host-2.example.org,0,
192.0.2.3,,,600,642,1,1,1,host-3.example.org,1,
192.0.2.4,00:01:02:03:04:04,01:02:03:04,600,642,1,1,1,host-4.example.org,0,
192.0.2.5,,,600,642,1,1,1,host-5.example.org,1,
192.0.2.6,00:01:02:03:04:06,01:02:03:06,600,642,1,1,1,host-6.example.org,0,
192.0.2.7,,,600,642,1,1,1,host-7.example.org,1,
192.0.2.8,00:01:02:03:04:08,01:02:03:08,600,642,1,1,1,host-8.example.org,0,
192.0.2.9,,,600,642,1,1,1,host-9.example.org,1,
192.0.2.10,00:01:02:03:04:10,01:02:03:10,600,642,1,1,1,host-10.example.org,0,
192.0.2.11,,,600,642,1,1,1,host-11.example.org,1,
192.0.2.12,00:01:02:03:04:12,01:02:03:12,600,642,1,1,1,host-12.example.org,0,
192.0.2.13,,,600,642,1,1,1,host-13.example.org,1,
192.0.2.14,00:01:02:03:04:14,01:02:03:14,600,642,1,1,1,host-14.example.org,0,
192.0.2.15,,,600,642,1,1,1,host-15.example.org,1,
192.0.2.16,00:01:02:03:04:16,01:02:03:16,600,642,1,1,1,host-16.example.org,0,
192.0.2.17,,,600,642,1,1,1,host-17.example.org,1,
192.0.2.18,00:01:02:03:04:18,01:02:03:18,600,642,1,1,1,host-18.example.org,0,
192.0.2.19,,,600,642,1,1,1,host-19.example.org,1,
"""

    with tempfile.TemporaryFile("r+t") as f:
        gen_dhcp4_lease_file(f, start=datetime.fromtimestamp(42))

        f.seek(0)
        actual = f.read()
        assert expected == actual


def test_gen_dhcp6_lease_file():
    expected = """address,duid,valid_lifetime,expire,subnet_id,pref_lifetime,lease_type,iaid,prefix_len,fqdn_fwd,fqdn_rev,hostname,hwaddr,state,user_context
3001:db8:1:42::1,00:00:00,600,642,1,300,0,7,128,1,1,host-1.example.org,,1,
3001:db8:1:42::2,01:02:03:02,600,642,1,300,0,7,128,1,1,host-2.example.org,,0,
3001:db8:1:42::3,00:00:00,600,642,1,300,0,7,128,1,1,host-3.example.org,,1,
3001:db8:1:42::4,01:02:03:04,600,642,1,300,0,7,128,1,1,host-4.example.org,,0,
3001:db8:1:42::5,00:00:00,600,642,1,300,0,7,128,1,1,host-5.example.org,,1,
3001:db8:1:42::6,01:02:03:06,600,642,1,300,0,7,128,1,1,host-6.example.org,,0,
3001:db8:1:42::7,00:00:00,600,642,1,300,0,7,128,1,1,host-7.example.org,,1,
3001:db8:1:42::8,01:02:03:08,600,642,1,300,0,7,128,1,1,host-8.example.org,,0,
3001:db8:1:42::9,00:00:00,600,642,1,300,0,7,128,1,1,host-9.example.org,,1,
3001:db8:1:42::10,01:02:03:10,600,642,1,300,0,7,128,1,1,host-10.example.org,,0,
3001:db8:1:42::11,00:00:00,600,642,1,300,0,7,128,1,1,host-11.example.org,,1,
3001:db8:1:42::12,01:02:03:12,600,642,1,300,0,7,128,1,1,host-12.example.org,,0,
3001:db8:1:42::13,00:00:00,600,642,1,300,0,7,128,1,1,host-13.example.org,,1,
3001:db8:1:42::14,01:02:03:14,600,642,1,300,0,7,128,1,1,host-14.example.org,,0,
3001:db8:1:42::15,00:00:00,600,642,1,300,0,7,128,1,1,host-15.example.org,,1,
3001:db8:1:42::16,01:02:03:16,600,642,1,300,0,7,128,1,1,host-16.example.org,,0,
3001:db8:1:42::17,00:00:00,600,642,1,300,0,7,128,1,1,host-17.example.org,,1,
3001:db8:1:42::18,01:02:03:18,600,642,1,300,0,7,128,1,1,host-18.example.org,,0,
3001:db8:1:42::19,00:00:00,600,642,1,300,0,7,128,1,1,host-19.example.org,,1,
"""

    with tempfile.TemporaryFile("r+t") as f:
        gen_dhcp6_lease_file(f, start=datetime.fromtimestamp(42))

        f.seek(0)
        actual = f.read()
        assert expected == actual
