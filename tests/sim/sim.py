"""
The main simulator project file.
It defines the Flask application and the main function.
"""

import os
import json
import logging

from flask import Flask, request
from flask.logging import create_logger

import server
import supervisor
import traffic

LOGLEVEL = os.environ.get("LOGLEVEL", "INFO").upper()
logging.basicConfig(level=LOGLEVEL)


app: Flask = None
log: logging.Logger = None


def _refresh_subnets():
    """Fetches list of subnets from Stork server and extends them with fields
    related to generating traffic. Stores the subnets in the app object."""
    app.subnets = {"items": [], "total": 0}
    subnets = server.get_subnets()

    # Add the simulator-specific fields to the subnets.
    for subnet in subnets["items"]:
        subnet["rate"] = 1
        subnet["clients"] = 1000
        subnet["state"] = "stop"
        subnet["proc"] = None
        if "sharedNetwork" not in subnet:
            subnet["sharedNetwork"] = ""
    app.subnets = subnets


def _refresh_bind9_applications():
    """Fetches list of BIND 9 applications from Stork server and extends them with
    fields related to generating traffic. Stores the BIND 9 applications in the app
    object."""
    app.bind9_applications = {"items": [], "total": 0}
    bind9_applications = server.get_bind9_applications()

    # Add the simulator-specific fields to the BIND 9 applications.
    for application in bind9_applications["items"]:
        if application["type"] == "bind9":
            application["clients"] = 1
            application["rate"] = 1
            application["qname"] = "example.com"
            application["qtype"] = "A"
            application["transport"] = "udp"
            application["proc"] = None
            application["state"] = "stop"
    app.bind9_applications = bind9_applications


def _refresh_services():
    """Fetches list of machines from Stork server and executes remote procedure
    call to extract list of services managed by SupervisorD. Stores the list of
    services in the app object."""
    app.services = {"items": [], "total": 0}
    machines = server.get_machines()
    services = supervisor.get_services(machines)
    app.services = services


def serialize_subnets(subnets):
    """Serializes subnets to JSON."""
    data = {"total": subnets["total"], "items": []}
    for subnet in subnets["items"]:
        data["items"].append(
            {
                "subnet": subnet["subnet"],
                "sharedNetwork": subnet["sharedNetwork"],
                "rate": subnet["rate"],
                "clients": subnet["clients"],
                "state": subnet["state"],
                "clientClass": subnet.get("clientClass"),
            }
        )
    return json.dumps(data)


def serialize_applications(applications):
    """Serializes applications to JSON."""
    data = {"total": applications["total"], "items": []}
    for application in applications["items"]:
        data["items"].append(
            {
                "state": application["state"],
                "address": application["machine"]["address"],
                "clients": application["clients"],
                "rate": application["rate"],
                "transport": application["transport"],
                "qtype": application["qtype"],
                "qname": application["qname"],
            }
        )
    return json.dumps(data)


def init():
    """Creates Flask application and logger."""
    app_instance = Flask(__name__, static_url_path="", static_folder="")
    logger_instance = create_logger(app_instance)
    return app_instance, logger_instance


def main():
    """Runs the simulator."""
    _refresh_subnets()
    _refresh_bind9_applications()
    _refresh_services()


# Creates the Flask application and runs the simulator.
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
            log.info("Stopping perfdhcp for subnet %s", subnet["subnet"])
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
                        log.info(
                            "Stopping perfdhcp for subnet %s", related_subnet["subnet"]
                        )
                        related_subnet["proc"].terminate()
                        related_subnet["proc"].wait()
                        related_subnet["proc"] = None
                        related_subnet["state"] = "stop"

            subnet["proc"] = traffic.start_perfdhcp(subnet)
            log.info("Started perfdhcp for subnet %s", subnet["subnet"])

        subnet["state"] = data["state"]

    return serialize_subnets(app.subnets)


@app.route("/applications")
def get_bind9_applications():
    """BIND 9 application list HTTP handler."""
    _refresh_bind9_applications()
    return serialize_applications(app.bind9_applications)


@app.route("/query/<int:index>", methods=["PUT"])
def put_dig_params(index):
    """Sends DNS query to a server with the given index."""
    data = json.loads(request.data)
    application = app.bind9_applications["items"][index]

    if "qname" in data:
        application["qname"] = data["qname"]

    if "qtype" in data:
        application["qtype"] = data["qtype"]

    if "transport" in data:
        application["transport"] = data["transport"]

    if "clients" in data:
        application["clients"] = data["clients"]

    if "rate" in data:
        application["rate"] = data["rate"]

    traffic.run_dig(application)
    log.info("Sent DNS query to %s", application["machine"]["address"])

    return serialize_applications(app.bind9_applications)


@app.route("/perf/<int:index>", methods=["PUT"])
def put_flamethrower_params(index):
    """Starts generating DNS traffic to a server with the given index."""
    data = json.loads(request.data)
    application = app.bind9_applications["items"][index]

    if "qname" in data:
        application["qname"] = data["qname"]

    if "qtype" in data:
        application["qtype"] = data["qtype"]

    if "transport" in data:
        application["transport"] = data["transport"]

    if "clients" in data:
        application["clients"] = data["clients"]

    if "rate" in data:
        application["rate"] = data["rate"]

    if "state" in data:
        # stop dnsperf if requested
        if (
            application["state"] == "start"
            and data["state"] == "stop"
            and application["proc"] is not None
        ):
            log.info("Stopping flamethrower for %s", application["machine"]["address"])
            application["proc"].terminate()
            application["proc"].wait()
            application["proc"] = None

        # start dnsperf if requested
        if application["state"] == "stop" and data["state"] == "start":
            application["proc"] = traffic.start_flamethrower(application)
            log.info("Started flamethrower for %s", application["machine"]["address"])

        application["state"] = data["state"]

    return serialize_applications(app.bind9_applications)


@app.route("/services")
def get_services():
    """The services page HTTP handler."""
    _refresh_services()
    return json.dumps(app.services)


@app.route("/services/<int:index>", methods=["PUT"])
def put_service(index):
    """Toggles a service with the given index."""
    data = json.loads(request.data)
    service = app.services["items"][index]

    if data["operation"] == "stop":
        supervisor.stop_service(service)
    elif data["operation"] == "start":
        supervisor.start_service(service)

    _refresh_services()
    return json.dumps(app.services)
