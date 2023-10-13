import logging as log
import subprocess
import sys
import shlex


def start_perfdhcp(subnet):
    """Generates traffic for a given network."""
    log.info("SUBNET %s", subnet)
    rate, clients = subnet["rate"], subnet["clients"]
    client_class = subnet["clientClass"]
    mac_prefix = client_class[6:].replace("-", ":")
    mac_prefix_bytes = mac_prefix.split(":")

    if "." in subnet["subnet"]:
        # ip4
        kea_addr = f"172.1{mac_prefix_bytes[0]}.0.100"
        cmd = [
            "/usr/sbin/perfdhcp",
            "-4",
            "-r",
            str(rate),
            "-R",
            str(clients),
            "-b",
            f"mac={mac_prefix}:00:00:00:00",
            kea_addr,
        ]
    else:
        # ip6
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
            f"mac={mac_prefix}:00:00:00:00",
        ]
    print("exec: %s" % cmd, file=sys.stderr)
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
    print(f"exec: {cmd}", file=sys.stderr)
    return subprocess.Popen(args)
