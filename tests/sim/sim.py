"""
The DHCP and DNS traffic simulator for the demo environment.
It isn't actively maintained.
"""

import os
import sys
import json
import shlex
import pprint
import subprocess
from xmlrpc.client import ServerProxy
from logging import Logger

from flask import Flask, request
from flask.logging import create_logger
import requests

app: Flask = None
log: Logger = None

STORK_SERVER_URL = os.environ.get("STORK_SERVER_URL", "http://server:8080")


def _login_session():
    session = requests.Session()
    credentials = {
        "authenticationMethodId": "internal",
        "identifier": "admin",
        "secret": "admin"
    }
    session.post(f"{STORK_SERVER_URL}/api/sessions", json=credentials)
    return session


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


def _refresh_subnets():
    try:
        app.subnets = {"items": [], "total": 0}

        session = _login_session()

        url = f"{STORK_SERVER_URL}/api/subnets?start=0&limit=100"
        response = session.get(url)
        data = response.json()
        log.info("SN %s", data)

        if not data:
            return

        for subnet in data["items"]:
            subnet["rate"] = 1
            subnet["clients"] = 1000
            subnet["state"] = "stop"
            subnet["proc"] = None
            if "sharedNetwork" not in subnet:
                subnet["sharedNetwork"] = ""

        app.subnets = data
    except Exception as exc:
        log.info("IGNORED EXCEPTION %s", str(exc))


def serialize_subnets(subnets):
    """Serializes subnets to JSON."""
    data = {
        "total": subnets["total"],
        "items": []
    }
    for subnet in subnets["items"]:
        data["items"].append({
            "subnet": subnet["subnet"],
            "sharedNetwork": subnet["sharedNetwork"],
            "rate": subnet["rate"],
            "clients": subnet["clients"],
            "state": subnet["state"],
        })
    return json.dumps(data)


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


def _refresh_servers():
    try:
        app.servers = {"items": [], "total":  0}

        session = _login_session()

        url = f"{STORK_SERVER_URL}/api/apps/"
        response = session.get(url)
        data = response.json()

        if not data:
            return

        for srv in data["items"]:
            if srv["type"] == "bind9":
                srv["clients"] = 1
                srv["rate"] = 1
                srv["qname"] = "example.com"
                srv["qtype"] = "A"
                srv["transport"] = "udp"
                srv["proc"] = None
                srv["state"] = "stop"
                app.servers["items"].append(srv)

        print("data: %s" % app.servers, file=sys.stderr)

    except Exception as exc:
        log.info("IGNORED EXCEPTION %s", str(exc))


def serialize_servers(servers):
    """Serializes servers to JSON."""
    data = {"total": servers["total"], "items": []}
    for srv in servers["items"]:
        data["items"].append({
            "state": srv["state"],
            "address": srv["machine"]["address"],
            "clients": srv["clients"],
            "rate": srv["rate"],
            "transport": srv["transport"],
            "qtype": srv["qtype"],
            "qname": srv["qname"]
        })
    return json.dumps(data)


def init():
    """Creates Flask application and logger."""
    app_instance = Flask(__name__, static_url_path="", static_folder="")
    log_instance = create_logger(app_instance)
    return app_instance, log_instance


def main():
    """Runs the simulator."""
    _refresh_subnets()
    _refresh_servers()


app, log = init()
main()


@app.route("/")
def root():
    """The root HTTP handler."""
    return app.send_static_file("index.html")


@app.route("/subnets")
def get_subnets():
    """Subnets list HTTP handler."""
    _refresh_subnets()
    return serialize_subnets(app.subnets)


@app.route("/subnets/<int:index>", methods=["PUT"])
def put_subnet_params(index):
    """Start generating DHCP traffic for a subnet with a given index."""
    data = json.loads(request.data)
    subnet = app.subnets["items"][index]

    if "rate" in data:
        subnet["rate"] = data["rate"]

    if "clients" in data:
        subnet["clients"] = data["clients"]

    if "state" in data:
        # stop perfdhcp if requested
        if (
            subnet["state"] == "start"
            and data["state"] == "stop"
            and subnet["proc"] is not None
        ):
            subnet["proc"].terminate()
            subnet["proc"].wait()
            subnet["proc"] = None

        # start perfdhcp if requested but if another subnet in the same shared network is running
        # then stop it first
        elif subnet["state"] == "stop" and data["state"] == "start":
            if subnet["sharedNetwork"] != "":
                for related_subnet in app.subnets["items"]:
                    if (
                        related_subnet["sharedNetwork"] == subnet["sharedNetwork"]
                        and related_subnet["state"] == "start"
                    ):
                        related_subnet["proc"].terminate()
                        related_subnet["proc"].wait()
                        related_subnet["proc"] = None
                        related_subnet["state"] = "stop"

            subnet["proc"] = start_perfdhcp(subnet)

        subnet["state"] = data["state"]

    return serialize_subnets(app.subnets)


@app.route("/servers")
def get_servers():
    """Servers list HTTP handler."""
    _refresh_servers()
    return serialize_servers(app.servers)


@app.route("/query/<int:index>", methods=["PUT"])
def put_query_params(index):
    """Sends DNS query to a server with the given index."""
    data = json.loads(request.data)
    server = app.servers["items"][index]

    if "qname" in data:
        server["qname"] = data["qname"]

    if "qtype" in data:
        server["qtype"] = data["qtype"]

    if "transport" in data:
        server["transport"] = data["transport"]

    if "clients" in data:
        server["clients"] = data["clients"]

    if "rate" in data:
        server["rate"] = data["rate"]

    run_dig(server)

    return serialize_servers(app.servers)


@app.route("/perf/<int:index>", methods=["PUT"])
def put_perf_params(index):
    """Starts generating DNS traffic to a server with the given index."""
    data = json.loads(request.data)
    server = app.servers["items"][index]

    if "qname" in data:
        server["qname"] = data["qname"]

    if "qtype" in data:
        server["qtype"] = data["qtype"]

    if "transport" in data:
        server["transport"] = data["transport"]

    if "clients" in data:
        server["clients"] = data["clients"]

    if "rate" in data:
        server["rate"] = data["rate"]

    if "state" in data:
        # stop dnsperf if requested
        if (
            server["state"] == "start"
            and data["state"] == "stop"
            and server["proc"] is not None
        ):
            server["proc"].terminate()
            server["proc"].wait()
            server["proc"] = None

        # start dnsperf if requested
        if server["state"] == "stop" and data["state"] == "start":
            server["proc"] = start_flamethrower(server)

        server["state"] = data["state"]

    return serialize_servers(app.servers)


def _get_services():
    app.services = {"items": [], "total": 0}

    session = _login_session()

    url = f"{STORK_SERVER_URL}/api/machines?start=0&limit=100"
    response = session.get(url)
    machines = response.json()["items"]
    if machines is None:
        machines = []

    data = {"items": [], "total": 0}
    for machine in machines:
        address = machine["address"]
        server = ServerProxy(f"http://{address}:9001/RPC2")
        try:
            services = (
                server.supervisor.getAllProcessInfo()
            )  # pylint: disable=no-member
        except Exception:
            continue
        pprint.pprint(services)
        for srv in services:
            srv["machine"] = address
            data["items"].append(srv)

    data["total"] = len(data["items"])

    app.services = data

    return data


@app.route("/services")
def get_services():
    """The services page HTTP handler."""
    data = _get_services()
    return json.dumps(data)


@app.route("/services/<int:index>", methods=["PUT"])
def put_service(index):
    """Toggles a service with the given index."""
    data = json.loads(request.data)
    service = app.services["items"][index]

    server = ServerProxy(f'http://{service["machine"]}:9001/RPC2')

    if data["operation"] == "stop":
        server.supervisor.stopProcess(service["name"])  # pylint: disable=no-member
    elif data["operation"] == "start":
        server.supervisor.startProcess(service["name"])  # pylint: disable=no-member

    data = _get_services()
    return json.dumps(data)
