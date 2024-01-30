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


def generate_reservations(
    version, reservation_range, mac_selector, address_modifier=1, subnet=""
):
    """
    Generates the host reservations' part of configuration.

    Parameters
    ----------
    version: int, 4 or 6
        IP family
    reservation_range: int
        The number of reservations to generate. Min: 0, max: 254.
    mac_selector: generator[str]
        Generator of the MAC addresses.
    address_modifier:
        The fixed octet included in all reserved addresses. Min: 0, max 255.
    subnet: str
        The two first octets of the subnet IP.
    """
    if reservation_range == 0:
        return {}

    # this is for usage outside generate_v4/6_subnet e.g. global
    if subnet == "" and version == 4:
        subnet = "11.0"  # default value for all tests
    elif subnet == "" and version == 6:
        subnet = "2001:db8"  # default value for all tests

    reservations = []
    for i in range(1, reservation_range + 1):
        if version == 4:
            single_reservation = {
                "hostname": f"reserved-hostname-{subnet}-{i}",
                # "option-data": [random.choice(optiondata4)],
                "hw-address": mac_selector(),
                "ip-address": f"{subnet}.{address_modifier}.{i}",
            }

            reservations.append(single_reservation)
        elif version == 6:
            single_reservation = {
                "hostname": f"reserved-hostname-{subnet}:{address_modifier}-{i}",
                "hw-address": mac_selector(),
                "ip-addresses": [f"{subnet}:{address_modifier}::{hex(i)[2:]}"],
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


def generate_v4_subnet(
    range_of_outer_scope,
    range_of_inner_scope,
    mac_selector,
    reservation_count=0,
    subnet_id_start=1,
    **kwargs,
):
    """
    Generates the DHCPv4 subnet's configuration entry.

    Parameters
    ----------
    range_of_outer_scope: int
        Number of generated group of addresses. The group is a first octet of
        IP address. Min: 0, max: 254.
    range_of_inner_scope: int
        Number of generated addresses in each group. Min: 0, max: 254.
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
    config = {"subnet4": []}
    netmask = 8 if range_of_inner_scope == 0 else 16
    for outer_scope in range(1, range_of_outer_scope + 1):
        for inner_scope in range(0, range_of_inner_scope + 1):
            subnet = {
                "pools": [
                    {
                        "pool": f"{outer_scope}.{inner_scope}.0.4-{outer_scope}.{inner_scope if netmask == 16 else 255}.255.254"
                    }
                ],
                "subnet": f"{outer_scope}.{inner_scope}.0.0/{netmask}",
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
                    address_modifier=inner_scope,
                    subnet=f"{outer_scope}.{inner_scope}",
                )
            )
            config["subnet4"].append(subnet)
            subnet_id += 1
    return config


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

    args = parser.parse_args()

    # If user specified a seed value, use it. If not, pass None to the seed(), so
    # system clock will be used.
    s = args.seed or None
    random.seed(s)

    number_of_subnets = args.n

    if number_of_subnets // 256 > 0:
        inner = 255
        outer = number_of_subnets // 256
    else:
        inner = 0
        outer = number_of_subnets

    conf = copy.deepcopy(KEA_BASE_CONFIG)

    if not args.use_hooks:
        conf["Dhcp4"]["hooks-libraries"] = []
    if args.interface is not None:
        conf["Dhcp4"]["interfaces-config"]["interfaces"] = args.interface

    mac_selector = create_mac_selector()
    new_subnets = generate_v4_subnet(
        outer, inner, mac_selector, args.reservations, args.start_id, **args.kwargs
    )
    conf["Dhcp4"].update(new_subnets)

    conf_json = json.dumps(conf)
    args.output.write(conf_json)


if __name__ == "__main__":
    cmd()
