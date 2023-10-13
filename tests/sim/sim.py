"""
The DHCP and DNS traffic simulator for the demo environment.
It isn't actively maintained.
"""

import json

from flask import Flask, request
from flask.logging import create_logger

import server
import supervisor
import traffic

app: Flask = None


def _refresh_subnets():
    subnets = server.get_subnets()
    app.subnets = subnets


def _refresh_applications():
    applications = server.get_applications()
    app.applications = applications


def _refresh_services():
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
    log_instance = create_logger(app_instance)
    return app_instance, log_instance


def main():
    """Runs the simulator."""
    _refresh_subnets()
    _refresh_applications()


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

            subnet["proc"] = traffic.start_perfdhcp(subnet)

        subnet["state"] = data["state"]

    return serialize_subnets(app.subnets)


@app.route("/applications")
def get_applications():
    """Servers list HTTP handler."""
    _refresh_applications()
    return serialize_applications(app.applications)


@app.route("/query/<int:index>", methods=["PUT"])
def put_query_params(index):
    """Sends DNS query to a server with the given index."""
    data = json.loads(request.data)
    application = app.applications["items"][index]

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

    return serialize_applications(app.applications)


@app.route("/perf/<int:index>", methods=["PUT"])
def put_perf_params(index):
    """Starts generating DNS traffic to a server with the given index."""
    data = json.loads(request.data)
    application = app.applications["items"][index]

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
            application["proc"].terminate()
            application["proc"].wait()
            application["proc"] = None

        # start dnsperf if requested
        if application["state"] == "stop" and data["state"] == "start":
            application["proc"] = traffic.start_flamethrower(server)

        application["state"] = data["state"]

    return serialize_applications(app.applications)


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
