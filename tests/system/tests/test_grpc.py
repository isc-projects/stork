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

import ssl

from typing import Type
from grpclib.exceptions import StreamTerminatedError

from core.wrappers import Kea, Bind9
from core.grpc_client import StorkAgentGRPCClient, GetStateRspAccessPoint


def assert_raises(exceptions: tuple[Type[Exception], ...], func, *args, **kwargs):
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
    - ssl.SSLCertVerificationError - after changing the SSLContext() protocol
      to ssl.PROTOCOL_TLS_CLIENT (from the default of None), @william observed
      this on aarch64-darwin.

    I spent a lot of time trying to figure out why the tests failed differently
    but I didn't find the reason. I decided to write this helper function as
    a workaround.
    """

    exceptions_t = tuple(exceptions)
    raised = None
    try:
        func(*args, **kwargs)
    except Exception as ex:  # pylint: disable=broad-except
        raised = ex

    if raised is None:
        raise AssertionError(
            "Function did not raise any exceptions, but it was expected to "
            f"raise one of {exceptions}"
        )
    is_expected = isinstance(raised, exceptions_t)
    context = raised.__context__
    is_context_expected = context is not None and isinstance(context, exceptions_t)
    if not is_expected and not is_context_expected:
        raise AssertionError(
            f"Function raised {raised}, but it was expected to raise one of {exceptions}"
        )


def test_grpc_ping(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC Ping command doesn't work if the request was not signed
    by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    assert_raises(
        (ConnectionResetError, StreamTerminatedError, ssl.SSLCertVerificationError),
        client.ping,
    )


def test_grpc_get_state(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC GetState command doesn't work if the request was not
    signed by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    assert_raises(
        (ConnectionResetError, StreamTerminatedError, ssl.SSLCertVerificationError),
        client.get_state,
    )


def test_grpc_forward_to_kea_over_http(kea_service: Kea, bind9_service: Bind9):
    """
    Check if the gRPC ForwardToKeaOverHTTP command doesn't work if the request
    was not signed by the server TLS certificate.
    """
    client = StorkAgentGRPCClient.for_service(kea_service)
    client.fetch_certs_from(bind9_service)

    assert_raises(
        (ConnectionResetError, StreamTerminatedError, ssl.SSLCertVerificationError),
        client.forward_to_kea_over_http,
        GetStateRspAccessPoint("control", "foo", 42, False),
        {
            "command": "version-get",
            "service": ["dhcp4"],
        },
    )
