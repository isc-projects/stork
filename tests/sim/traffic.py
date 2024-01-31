"""
This file contains functions that generate DHCP and DNS traffic.
"""

import re
import subprocess
import sys
import shlex


# Pattern that matches the client classes in the subnets defined in the demo
# Kea configurations.
# Example of valid client class: client-class-01-01.
# The first number is the subnet index to use: 00, 01 (IPv4 and IPv6),
# 02, 03 (IPv4 only).
# The numbers are used as first bytes of the MAC address.
_client_class_pattern = re.compile(r"class-(\d{2})-(\d{2})")


def start_perfdhcp(subnet):
    """Generates traffic for a given network."""
    rate, clients = subnet["rate"], subnet["clients"]

    client_class_bytes = None
    if "clientClass" not in subnet:
        raise ValueError(f"Missing client class for subnet: {subnet['subnet']}")

    client_class = subnet["clientClass"]
    m = _client_class_pattern.match(client_class)
    if m is None:
        raise ValueError(
            f"Invalid client class: {subnet['clientClass']} for subnet: {subnet['subnet']}"
        )
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
