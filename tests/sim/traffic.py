"""
This file contains functions that generate DHCP and DNS traffic.
"""

import re
import subprocess
import sys
import shlex
import logging

log = logging.getLogger(__name__)

# Pattern that matches the client classes in the subnets defined in the demo
# Kea configurations.
# Example of valid client class: client-class-01-01.
# The first number is the subnet index to use: 00, 01 (IPv4 and IPv6),
# 02, 03 (IPv4 only).
# The numbers are used as first bytes of the MAC address.
_client_class_pattern = re.compile(r"class-(\d{2})-(\d{2})")


def get_subnet_client_classes(subnet):
    """Returns the client classes for a subnet."""
    return (
        subnet.get("localSubnets", [{}])[0]
        .get("keaConfigSubnetParameters", {})
        .get("subnetLevelParameters", {})
        .get("clientClasses", None)
    )


def start_perfdhcp(subnet):
    """Generates traffic for a given network."""
    rate, clients = subnet["rate"], subnet["clients"]

    m = None

    # In latest Kea versions the client classes are defined in the local-subnet
    # under client-classes array.
    client_classes = get_subnet_client_classes(subnet)
    if client_classes:
        for client_class in client_classes:
            m = _client_class_pattern.match(client_class)
            if m is not None:
                break

    # If client classes are not defined in client-classes let's check the
    # client-class parameter.
    if m is None and "clientClass" in subnet:
        client_class = subnet["clientClass"]
        m = _client_class_pattern.match(client_class)

    if m is None:
        # Client class not found, so we cannot generate traffic.
        raise ValueError(f"Missing client class for subnet: {subnet['subnet']}")

    client_class_bytes = m.groups()

    if "." in subnet["subnet"]:
        # IPv4
        kea_addr = f"172.1{client_class_bytes[0]}.0.100"
        cmd = [
            "/usr/sbin/perfdhcp",
            "-4",
            "-r",
            str(rate),
            "-R",
            str(clients),
            "-b",
            f"mac={client_class_bytes[0]}:{client_class_bytes[1]}:00:00:00:00",
            kea_addr,
        ]
    else:
        # IPv6
        cmd = [
            "/usr/sbin/perfdhcp",
            "-6",
            "-r",
            str(rate),
            "-R",
            str(clients),
            "-l",
            "eth1",
            "-b",
            "duid=000000000000",
            "-b",
            f"mac={client_class_bytes[0]}:{client_class_bytes[1]}:00:00:00:00",
        ]
    return subprocess.Popen(cmd)


def run_dig(server):
    """Generates DNS traffic to a given server using dig."""
    clients = server["clients"]
    qname = server["qname"]
    qtype = server["qtype"]
    tcp = "+notcp"
    if server["transport"] == "tcp":
        tcp = "+tcp"
    address = server["machine"]["address"]
    cmd = f"dig {tcp} +tries=1 +retry=0 @{address} {qname} {qtype}"
    print(f"exec {clients} times: {cmd}", file=sys.stderr)
    for _ in range(0, clients):
        args = shlex.split(cmd)
        subprocess.run(args, check=False)


def start_flamethrower(server):
    """Generates DNS traffic to a given server using flamethrower."""
    rate = server["rate"] * 1000
    clients = server["clients"]
    qname = server["qname"]
    qtype = server["qtype"]
    transport = "udp"
    if server["transport"] == "tcp":
        transport = "tcp"
    address = server["machine"]["address"]
    # send one query (-q) per client (-c) every 'rate' millisecond (-d)
    # on transport (-P) with qname (-r) and qtype (-T)
    cmd = f"flame -q 1 -c {clients} -d {rate} -P {transport} -r {qname} -T {qtype} {address}"
    args = shlex.split(cmd)
    return subprocess.Popen(args)
