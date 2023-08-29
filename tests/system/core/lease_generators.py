import csv
from datetime import datetime, timedelta


def gen_dhcp4_lease_file(
    target, start: datetime = None, lifetime: timedelta = timedelta(minutes=10)
):
    """Generates the DHCPv4 lease file."""
    if start is None:
        start = datetime.now()
    expire = start + lifetime

    # DHCPv4 lease file header.
    fileheader = [
        "address",
        "hwaddr",
        "client_id",
        "valid_lifetime",
        "expire",
        "subnet_id",
        "fqdn_fwd",
        "fqdn_rev",
        "hostname",
        "state",
        "user_context",
    ]
    # Create CSV writer and output its header.
    lease_writer = csv.DictWriter(target, fieldnames=fileheader, lineterminator="\n")
    lease_writer.writeheader()

    # Generate some leases and output them to the lease file.
    for i in range(1, 20):
        # Even leases are in default state. Odd leases are declined.
        lease = {
            "hwaddr": "",
            "client_id": "",
            "valid_lifetime": 600,
            "subnet_id": 1,
            "fqdn_fwd": 1,
            "fqdn_rev": 1,
            "hostname": f"host-{i}.example.org",
            "state": i % 2,
        }
        lease["address"] = f"192.0.2.{i}"
        lease["expire"] = str(int(expire.timestamp()))

        # Only non-declined leases contain MAC address and client id.
        if i % 2 == 0:
            lease["hwaddr"] = f"00:01:02:03:04:{i:02d}"
            lease["client_id"] = f"01:02:03:{i:02d}"

        lease_writer.writerow(lease)


def gen_dhcp6_lease_file(
    target, start: datetime = None, lifetime: timedelta = timedelta(minutes=10)
):
    """Generates the DHCPv6 lease file."""
    if start is None:
        start = datetime.now()
    expire = start + lifetime

    # DHCPv6 lease file header.
    fileheader = [
        "address",
        "duid",
        "valid_lifetime",
        "expire",
        "subnet_id",
        "pref_lifetime",
        "lease_type",
        "iaid",
        "prefix_len",
        "fqdn_fwd",
        "fqdn_rev",
        "hostname",
        "hwaddr",
        "state",
        "user_context",
    ]
    # Create CSV writer and output its header.
    lease_writer = csv.DictWriter(target, fieldnames=fileheader, lineterminator="\n")
    lease_writer.writeheader()

    # Generate some leases and output them to the lease file.
    for i in range(1, 20):
        # Even leases are in default state. Odd leases are declined.
        lease = {
            "duid": "00:00:00",
            "valid_lifetime": 600,
            "subnet_id": 1,
            "pref_lifetime": 300,
            "lease_type": 0,
            "iaid": 7,
            "prefix_len": 128,
            "fqdn_fwd": 1,
            "fqdn_rev": 1,
            "hostname": f"host-{i}.example.org",
            "state": i % 2,
        }
        lease["address"] = f"3001:db8:1:42::{i}"
        lease["expire"] = str(int(expire.timestamp()))

        # Only non-declined leases contain MAC address and client id.
        if i % 2 == 0:
            lease["duid"] = f"01:02:03:{i:02d}"

        lease_writer.writerow(lease)
