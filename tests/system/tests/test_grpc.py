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

from typing import Tuple

from grpclib.exceptions import StreamTerminatedError

from core.wrappers import Kea, Bind9
from core.grpc_client import StorkAgentGRPCClient, GetStateRspAppAccessPoint


def assert_raises(exceptions: Tuple[Exception], func, *args, **kwargs):
    """
    Assert that the function raises the expected exception.
    It handles the case when the original exception is covered by another one
    thrown in its except block.

    It is needed because the GRPC client raises various exceptions in various
    environments:

    - StreamTerminatedError - on my local machine
    - ConnectionResetError - in Docker-in-Docker running on my local machine
    - StreamTerminatedError but the AttributeError is raised when the exception
        is internally processed by the GRPC library - on CI

    I spent a lot of time trying to figure out why the tests failed differently
    but I didn't find the reason. I decided to write this helper function as
    a workaround.
    """

    raised = False
    try:
        func(*args, **kwargs)
    except exceptions:
        raised = True
    except Exception as ex:  # pylint: disable=broad-except
        original_exception = ex.__context__
        if original_exception is not None and isinstance(
            original_exception, exceptions
        ):
            raised = True

    if not raised:
        raise AssertionError(f"Function did not raise any of {exceptions}")


def test_grpc_ping(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC Ping command doesn't work if the request was not signed
    by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    assert_raises((ConnectionResetError, StreamTerminatedError), client.ping)


def test_grpc_get_state(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC GetState command doesn't work if the request was not
    signed by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    assert_raises((ConnectionResetError, StreamTerminatedError), client.get_state)


def test_grpc_forward_to_kea_over_http(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC ForwardToKeaOverHTTP command doesn't work if the request
    was not signed by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    assert_raises(
        (ConnectionResetError, StreamTerminatedError),
        client.forward_to_kea_over_http,
        GetStateRspAppAccessPoint("control", "foo", 42, False),
        {
            "command": "version-get",
            "service": ["dhcp4"],
        },
    )
