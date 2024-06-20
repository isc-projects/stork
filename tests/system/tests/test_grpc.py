"""
Tests in this file send gRPC commands directly to the Stork agent. The server
is responsible only for generating the TLS certificates.

The GRPC commands are sent from the host machine to the Stork agent running in
a Docker container.

To avoid implementing the registration process in Python, we use a separate
container with the Stork agent and copy its TLS certificates to the host.

For example, we instantiate two containers with Kea and Bind9 services. Their
Stork agents perform the registration at startup. We copy the certificates from
BIND 9 container and use them to send gRPC commands to the Kea container.
"""

import pytest
from grpclib.exceptions import StreamTerminatedError

from core.wrappers import Kea, Bind9
from core.grpc_client import StorkAgentGRPCClient, GetStateRspAppAccessPoint


def test_grpc_ping(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC Ping command doesn't work if the request was not signed
    by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    pytest.raises((ConnectionResetError, StreamTerminatedError), client.ping)


def test_grpc_get_state(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC GetState command doesn't work if the request was not
    signed by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    pytest.raises(StreamTerminatedError, client.get_state)


def test_grpc_forward_to_kea_over_http(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC ForwardToKeaOverHTTP command doesn't work if the request
    was not signed by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    pytest.raises(
        StreamTerminatedError,
        client.forward_to_kea_over_http,
        GetStateRspAppAccessPoint("control", "foo", 42, False),
        {
            "command": "version-get",
            "service": ["dhcp4"],
        },
    )
