"""
This module is used to perform remote procedure calls to the SupervisorD daemon
running in the demo containers.
"""

import typing
from xmlrpc.client import ServerProxy


class SupervisorRPC:
    """Typing for SupervisorD RPC interface."""

    # pylint: disable=invalid-name
    def getAllProcessInfo(self) -> typing.Sequence:
        """Returns all services managed by SupervisorD."""
        raise TypeError("It is only typing stub")

    # pylint: disable=invalid-name
    def startProcess(self, name: str) -> bool:
        """Starts a given SupervisorD service."""
        raise TypeError("It is only typing stub")

    # pylint: disable=invalid-name
    def stopProcess(self, name: str) -> bool:
        """Stops a given SupervisorD service."""
        raise TypeError("It is only typing stub")


def _create_supervisor_rpc_client(address: str) -> SupervisorRPC:
    server = ServerProxy(f"http://{address}:9001/RPC2")
    return server.supervisor


def get_services(machines):
    """Fetches the list of services managed by SupervisorD daemons running on
    the given machines."""
    data = {"items": [], "total": 0}
    for machine in machines["items"]:
        address = machine["address"]
        rpc_client = _create_supervisor_rpc_client(address)

        try:
            services = rpc_client.getAllProcessInfo()
        except Exception:  # pylint: disable=broad-except
            continue
        for srv in services:
            srv["machine"] = address
            data["items"].append(srv)

    data["total"] = len(data["items"])

    return data


def start_service(service):
    """Starts a given service managed by SupervisorD."""
    address = service["machine"]
    rpc_client = _create_supervisor_rpc_client(address)
    rpc_client.startProcess(service["name"])


def stop_service(service):
    """Stops a given service managed by SupervisorD."""
    address = service["machine"]
    rpc_client = _create_supervisor_rpc_client(address)
    rpc_client.stopProcess(service["name"])
