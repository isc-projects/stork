#!/usr/bin/env python3

"""
Script to generate the Kea configuration with a given number of subnets and/or
host reservations.
It is dedicated for performance tests.
"""

import argparse
import sys
import json
import copy
import random
import itertools


def ipv4_address_generator(number_of_addresses, begin="1.0.0.0", mask=32):
    """
    Generates a sequence of IPv4 addresses.

    Parameters
    ----------
    number_of_addresses : int
        Number of addresses to generate.
    begin : str, optional
        A first address to generate, by default "1.0.0.0"
    mask : int, optional
        An address mask, the part outside mask is preserved, by default 32
    """

    def increment(address):
        address_binary = int(
            "".join(f"{int(octet):08b}" for octet in address.split(".")), 2
        )
        address_binary += 1 << (32 - mask)
        return ".".join(
            str(int(address_binary >> (24 - i * 8) & 0xFF)) for i in range(4)
        )

    address = begin
    for _ in range(number_of_addresses):
        yield address
        address = increment(address)


class ParseKwargs(argparse.Action):
    """Parse key-value pairs from CMD. Source: https://sumit-ghosh.com/articles/parsing-dictionary-key-value-pairs-kwargs-argparse-python/"""

    def __call__(self, parser, namespace, values, option_string=None):
        setattr(namespace, self.dest, {})
        for value in values:
            key, value = value.split("=")
            getattr(namespace, self.dest)[key] = value


# TODO add entire set of v4 options
optiondata4 = [
    {"code": 2, "data": "50", "name": "time-offset", "space": "dhcp4"},
    {
        "code": 3,
        "data": "100.100.100.10,50.50.50.5",
        "name": "routers",
        "space": "dhcp4",
    },
    {
        "code": 4,
        "data": "199.199.199.1,199.199.199.2",
        "name": "time-servers",
        "space": "dhcp4",
    },
    {
        "code": 5,
        "data": "199.199.199.1,100.100.100.1",
        "name": "name-servers",
        "space": "dhcp4",
    },
    {
        "code": 6,
        "data": "199.199.199.1,100.100.100.1",
        "name": "domain-name-servers",
        "space": "dhcp4",
    },
    {
        "code": 7,
        "data": "199.199.199.1,100.100.100.1",
        "name": "log-servers",
        "space": "dhcp4",
    },
    {
        "code": 76,
        "data": "199.1.1.1,200.1.1.2",
        "name": "streettalk-directory-assistance-server",
        "space": "dhcp4",
    },
    {
        "code": 19,
        "csv-format": True,
        "data": "True",
        "name": "ip-forwarding",
        "space": "dhcp4",
    },
    {"code": 20, "data": "True", "name": "non-local-source-routing", "space": "dhcp4"},
    {"code": 29, "data": "False", "name": "perform-mask-discovery", "space": "dhcp4"},
]

optiondata6 = [
    {"code": 7, "data": "123", "name": "preference", "space": "dhcp6"},
    {
        "code": 21,
        "data": "srv1.example.com,srv2.isc.org",
        "name": "sip-server-dns",
        "space": "dhcp6",
    },
    {
        "code": 23,
        "data": "2001:db8::1,2001:db8::2",
        "name": "dns-servers",
        "space": "dhcp6",
    },
    {
        "code": 24,
        "data": "domain1.example.com,domain2.isc.org",
        "name": "domain-search",
        "space": "dhcp6",
    },
    {
        "code": 22,
        "data": "2001:db8::1,2001:db8::2",
        "name": "sip-server-addr",
        "space": "dhcp6",
    },
    {
        "code": 28,
        "data": "2001:db8::abc,3000::1,2000::1234",
        "name": "nisp-servers",
        "space": "dhcp6",
    },
    {
        "code": 27,
        "data": "2001:db8::abc,3000::1,2000::1234",
        "name": "nis-servers",
        "space": "dhcp6",
    },
    {
        "code": 29,
        "data": "ntp.example.com",
        "name": "nis-domain-name",
        "space": "dhcp6",
    },
    {
        "code": 30,
        "data": "ntp.example.com",
        "name": "nisp-domain-name",
        "space": "dhcp6",
    },
    {
        "code": 31,
        "data": "2001:db8::abc,3000::1,2000::1234",
        "name": "sntp-servers",
        "space": "dhcp6",
    },
    {
        "code": 32,
        "data": "12345678",
        "name": "information-refresh-time",
        "space": "dhcp6",
    },
    {"code": 12, "data": "3000::66", "name": "unicast", "space": "dhcp6"},
    {
        "code": 33,
        "data": "very.good.domain.name.com",
        "name": "bcmcs-server-dns",
        "space": "dhcp6",
    },
    {
        "code": 34,
        "data": "3000::66,3000::77",
        "name": "bcmcs-server-addr",
        "space": "dhcp6",
    },
    {"code": 40, "data": "3000::66,3000::77", "name": "pana-agent", "space": "dhcp6"},
    {"code": 41, "data": "EST5EDT4", "name": "new-posix-timezone", "space": "dhcp6"},
    {
        "code": 42,
        "data": "Europe/Zurich",
        "name": "new-tzdb-timezone",
        "space": "dhcp6",
    },
    {
        "code": 59,
        "data": "http://www.kea.isc.org",
        "name": "bootfile-url",
        "space": "dhcp6",
    },
    {
        "code": 60,
        "data": "000B48656C6C6F20776F726C640003666F6F",
        "name": "bootfile-param",
        "space": "dhcp6",
    },
    {
        "code": 65,
        "data": "erp-domain.isc.org",
        "name": "erp-local-domain-name",
        "space": "dhcp6",
    },
    # {"code": 32, "data": "2001:558:ff18:16:10:253:175:76", "name": "tftp-servers", "space": "vendor-4491"},
    # {"code": 33, "data": "normal_erouter_v6.cm", "name": "config-file", "space": "vendor-4491"},
    # {"code": 34, "data": "2001:558:ff18:10:10:253:101", "name": "syslog-servers", "space": "vendor-4491"},
    # {"code": 37, "data": "2001:558:ff18:16:10:253:175:76", "name": "time-servers", "space": "vendor-4491"},
    # {"code": 38, "data": "-10000", "name": "time-offset", "space": "vendor-4491"}
]

KEA_BASE_CONFIG = {
    "Dhcp4": {
        "interfaces-config": {"interfaces": ["eth0"]},
        "control-socket": {
            "socket-type": "unix",
            "socket-name": "/tmp/kea4-ctrl-socket",
        },
        "lease-database": {"type": "memfile", "lfc-interval": 3600},
        "expired-leases-processing": {
            "reclaim-timer-wait-time": 10,
            "flush-reclaimed-timer-wait-time": 25,
            "hold-reclaimed-time": 3600,
            "max-reclaim-leases": 100,
            "max-reclaim-time": 250,
            "unwarned-reclaim-cycles": 5,
        },
        "renew-timer": 90,
        "rebind-timer": 120,
        "valid-lifetime": 180,
        "reservations": [
            {"hw-address": "ee:ee:ee:ee:ee:ee", "ip-address": "10.0.0.123"},
            {"client-id": "aa:aa:aa:aa:aa:aa", "ip-address": "10.0.0.222"},
        ],
        "option-data": [
            {"name": "domain-name-servers", "data": "192.0.2.1, 192.0.2.2"},
            {"code": 15, "data": "example.org"},
            {"name": "domain-search", "data": "mydomain.example.com, example.com"},
            {
                "name": "boot-file-name",
                "data": "EST5EDT4\\,M3.2.0/02:00\\,M11.1.0/02:00",
            },
            {"name": "default-ip-ttl", "data": "0xf0"},
        ],
        "client-classes": [
            {
                "name": "class-00-00",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '00:00'",
            },
            {
                "name": "class-01-00",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:00'",
            },
            {
                "name": "class-01-01",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:01'",
            },
            {
                "name": "class-01-02",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:02'",
            },
            {
                "name": "class-01-03",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:03'",
            },
            {
                "name": "class-01-04",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:04'",
            },
            {
                "name": "class-02-00",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '02:00'",
            },
            {
                "name": "class-02-01",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '02:01'",
            },
            {
                "name": "class-02-02",
                "test": "substring(hexstring(pkt4.mac,':'),0,5) == '02:02'",
            },
        ],
        "hooks-libraries": [
            {"library": "/usr/lib/x86_64-linux-gnu/kea/hooks/libdhcp_lease_cmds.so"},
            {"library": "/usr/lib/x86_64-linux-gnu/kea/hooks/libdhcp_stat_cmds.so"},
        ],
        "subnet4": [],
        "shared-networks": [],
        "loggers": [
            {
                "name": "kea-dhcp4",
                "output_options": [
                    {"output": "stdout", "pattern": "%-5p %m\n"},
                    {"output": "/tmp/kea-dhcp4.log"},
                ],
                "severity": "DEBUG",
                "debuglevel": 0,
            }
        ],
    }
}

KEA_BASE_SUBNET = {
    "subnet": "192.0.2.0/24",
    # "pools": [ { "pool": "192.0.2.1 - 192.0.2.50" },
    #            { "pool": "192.0.2.51 - 192.0.2.100" },
    #            { "pool": "192.0.2.101 - 192.0.2.150" },
    #            { "pool": "192.0.2.151 - 192.0.2.200" } ],
    "client-class": "class-00-00",
    "relay": {"ip-addresses": ["172.100.0.200"]},
    "option-data": [{"name": "routers", "data": "192.0.2.1"}],
    "reservations": [
        {"hw-address": "1a:1b:1c:1d:1e:1f", "ip-address": "192.0.2.101"},
        {
            "client-id": "01:11:22:33:44:55:66",
            "ip-address": "192.0.2.102",
            "hostname": "special-snowflake",
        },
        {
            "duid": "01:02:03:04:05",
            "ip-address": "192.0.2.103",
            "option-data": [
                {"name": "domain-name-servers", "data": "10.1.1.202, 10.1.1.203"}
            ],
        },
        {
            "client-id": "01:12:23:34:45:56:67",
            "ip-address": "192.0.2.104",
            "option-data": [
                {"name": "vivso-suboptions", "data": "4491"},
                {
                    "name": "tftp-servers",
                    "space": "vendor-4491",
                    "data": "10.1.1.202, 10.1.1.203",
                },
            ],
        },
        {
            "client-id": "01:0a:0b:0c:0d:0e:0f",
            "ip-address": "192.0.2.105",
            "next-server": "192.0.2.1",
            "server-hostname": "hal9000",
            "boot-file-name": "/dev/null",
        },
        {"flex-id": "'s0mEVaLue'", "ip-address": "192.0.2.106"},
    ],
}


def create_mac_selector():
    """Returns a generator of sequential mac addresses."""
    mac_addr_iter = 0

    def mac_selector():
        nonlocal mac_addr_iter
        mac_addr_iter += 1
        return ":".join(
            [f"{a}{b}" for a, b in zip(*[iter(f"{mac_addr_iter:012x}")] * 2)]
        )

    return mac_selector


def generate_reservations(version, number_of_reservations, mac_selector, subnet=""):
    """
    Generates the host reservations' part of configuration.

    Parameters
    ----------
    version: int, 4 or 6
        IP family
    number_of_reservations: int
        The number of reservations to generate. Min: 0, max: 254.
    mac_selector: generator[str]
        Generator of the MAC addresses.
    subnet: str
        The three first octets of the subnet IP.
    """
    if number_of_reservations == 0:
        return {}

    # this is for usage outside generate_v4/6_subnet e.g. global
    if subnet == "" and version == 4:
        subnet = "11.0.0"  # default value for all tests
    elif subnet == "" and version == 6:
        subnet = "2001:db8"  # default value for all tests

    reservations = []
    for i in range(1, number_of_reservations + 1):
        if version == 4:
            single_reservation = {
                "hostname": f"reserved-hostname-{subnet}-{i}",
                "hw-address": mac_selector(),
                "ip-address": f"{subnet}.{i}",
            }

            reservations.append(single_reservation)
        elif version == 6:
            single_reservation = {
                "hostname": f"reserved-hostname-{subnet}-{i}",
                "hw-address": mac_selector(),
                "ip-addresses": [f"{subnet}::{hex(i)[2:]}"],
            }

            reservations.append(single_reservation)
        else:
            assert False, "Something wrong, IP version can be 4 or 6"
    return {"reservations": reservations}


def get_option(ip_version, number_of_options=1):
    """Generates a random DHCP option(s). Returns error if number_of_options
    is higher than length of optiondata4/6."""
    if ip_version == 4:
        return {"option-data": random.sample(optiondata4, number_of_options)}
    return {"option-data": random.sample(optiondata6, number_of_options)}


def generate_v4_subnets(
    subnet_generator,
    mac_selector,
    reservation_count=0,
    subnet_id_start=1,
    **kwargs,
):
    """
    Generates the DHCPv4 subnet's configuration entry.

    Parameters
    ----------
    subnet_generator
        A generator of subnets. It is expected that the generated subnets have
        a mask of 24 and end with 0.
    mac_selector: generator
        Generator of MAC addresses.
    reservation_count: int
        Number of host reservations in each subnet.
    subnet_id_start: int
        The first subnet ID. The generated IDs are sequential.
    **kwargs: dict
        Additional subnet properties.
    """

    subnet_id = subnet_id_start

    # TODO move to binary generator
    subnets = []
    for subnet_address in subnet_generator:
        subnet_prefix = f"{subnet_address}/24"
        pool_start = f"{subnet_address[:-2]}.{reservation_count + 1}"
        pool_end = f"{subnet_address[:-2]}.254"

        subnet = {
            "pools": [
                {
                    "pool": f"{pool_start}-{pool_end}",
                }
            ],
            "subnet": subnet_prefix,
            "option-data": random.choices(optiondata4, k=6),
            "client-class": "class-00-00",
            "relay": {"ip-addresses": ["172.100.0.200"]},
            "id": subnet_id,
        }
        subnet.update(kwargs)
        subnet.update(
            generate_reservations(
                4,
                reservation_count,
                mac_selector,
                subnet=subnet_address[:-2],
            )
        )
        subnets.append(subnet)
        subnet_id += 1
    return subnets


def cmd():
    """Parses CLI arguments and executes the program."""
    parser = argparse.ArgumentParser("Kea config generator")
    parser.add_argument("n", type=int, help="Number of subnets")
    parser.add_argument(
        "-s", "--start-id", type=int, default=1, help="Start subnet index"
    )
    parser.add_argument(
        "-r",
        "--reservations",
        type=int,
        default=0,
        help="Number of reservations in subnet",
    )
    parser.add_argument(
        "-k",
        "--kwargs",
        nargs="*",
        action=ParseKwargs,
        default={},
        help="Key-value pairs",
    )
    group = parser.add_mutually_exclusive_group()
    group.add_argument(
        "--use-hooks",
        action="store_true",
        default=True,
        help="Enable hook libraries",
        dest="use_hooks",
    )
    group.add_argument(
        "--no-use-hooks",
        action="store_false",
        help="Disable hook libraries",
        dest="use_hooks",
    )
    parser.add_argument(
        "-i", "--interface", nargs=1, type=str, default=None, help="Interface name"
    )
    parser.add_argument(
        "-o",
        "--output",
        type=argparse.FileType("w"),
        default=sys.stdout,
        help="Output target",
    )
    parser.add_argument(
        "--seed",
        type=int,
        default=0,
        help="Seed used to initialize PRNG, defaults to system time",
    )
    parser.add_argument(
        "-n",
        "--shared-networks",
        type=int,
        default=0,
        help="Number of shared networks",
    )

    args = parser.parse_args()

    # If user specified a seed value, use it. If not, pass None to the seed(), so
    # system clock will be used.
    random.seed(args.seed or None)

    conf = copy.deepcopy(KEA_BASE_CONFIG)

    if not args.use_hooks:
        conf["Dhcp4"]["hooks-libraries"] = []
    if args.interface is not None:
        conf["Dhcp4"]["interfaces-config"]["interfaces"] = args.interface

    mac_selector = create_mac_selector()

    number_of_subnets = args.n
    number_of_shared_networks = args.shared_networks

    subnet_generator = ipv4_address_generator(number_of_subnets, mask=24)

    if number_of_shared_networks == 0:
        new_subnets = generate_v4_subnets(
            subnet_generator,
            mac_selector,
            args.reservations,
            args.start_id,
            **args.kwargs,
        )
        conf["Dhcp4"]["subnet4"] = new_subnets
    else:
        shared_networks = []
        subnet_id = args.start_id
        subnets_per_shared_network = number_of_subnets // number_of_shared_networks
        for i in range(number_of_shared_networks):
            subnets = generate_v4_subnets(
                itertools.islice(subnet_generator, subnets_per_shared_network),
                mac_selector,
                args.reservations,
                subnet_id,
                **args.kwargs,
            )

            shared_network = {
                "name": f"shared-network-{i}",
                "subnet4": subnets,
            }
            shared_networks.append(shared_network)
            subnet_id += subnets_per_shared_network
        conf["Dhcp4"]["shared-networks"] = shared_networks

    args.output.write(json.dumps(conf))


if __name__ == "__main__":
    cmd()
