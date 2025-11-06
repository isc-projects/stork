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
# Examples of valid client classes:
# - client-class-01-01
# - client-class-03-47-RELAY
# - client-class-00-04-eth2
# - client-class-04-10-enp0s4-RELAY
# The first number is the subnet index to use: 00, 01 (IPv4 and IPv6),
# 02, 03 (IPv4 only).
# The numbers are used as first bytes of the MAC address.
# Adding an interface specifier tells perfdhcp which interface to send traffic
# from. By default, eth1.
# Adding -RELAY tells perfdhcp to act like a DHCP relay when sending traffic.
_client_class_pattern = re.compile(r"class-(\d{2})-(\d{2})(-[a-z0-9]+)?(-RELAY)?")


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
        # Also, whatever version of Pylint we're using is not smart enough to
        # realize that the quotes have to be different here, otherwise the
        # f-string is closed prematurely.
        # pylint: disable-next=inconsistent-quotes
        raise ValueError(f"Missing client class for subnet: {subnet['subnet']}")

    client_class_params = m.groups()
    byte0, byte1, iface, relay = client_class_params
    if not iface:
        iface = "eth1"
    else:
        # Trim off dash in front of interface name.
        iface = iface[1:]
    is_relay = bool(relay == "-RELAY")

    if "." in subnet["subnet"]:
        # IPv4
        kea_addr = f"172.1{byte0}.0.100"
        cmd = [
            "/usr/sbin/perfdhcp",
            "-4",
            "-r",
            str(rate),
            "-R",
            str(clients),
        ]
        if is_relay:
            cmd += ["-A", "1"]
        cmd += [
            "-b",
            f"mac={byte0}:{byte1}:00:00:00:00",
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
            iface,
        ]
        if is_relay:
            cmd += ["-A", "1"]
        cmd += [
            "-b",
            "duid=000000000000",
            "-b",
            f"mac={byte0}:{byte1}:00:00:00:00",
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
