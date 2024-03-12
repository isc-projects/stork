"""
Stork agent GRPC client.
"""

import abc
from dataclasses import dataclass
import asyncio
import json
import ssl
import tempfile
import typing

from grpclib.client import Channel
from agent_pb2 import GetStateReq, PingReq, ForwardToKeaOverHTTPReq, KeaRequest

from agent_grpc import AgentStub
import core.wrappers.agent


class StatusError(Exception):
    """The exception raised when the status code is not 0."""

    def __init__(self, code, message):
        super().__init__(f"Status code is {code}: {message}")


# The protoc GRPC generator doesn't provide the typing hints for the generated
# classes.
@dataclass()
class GetStateRspAppAccessPoint(abc.ABC):
    """Stub class for the AccessPoint entry in the GetStateRsp message."""

    type: str
    address: str
    port: int
    # pylint: disable=invalid-name
    useSecureProtocol: bool


@dataclass()
class GetStateRspApp(abc.ABC):
    """Stub class for the App entry in the GetStateRsp message."""

    type: str
    # pylint: disable=invalid-name
    accessPoints: typing.List[GetStateRspAppAccessPoint]


@dataclass()
class GetStateRsp(abc.ABC):
    """
    Stub class for the GetStateRsp message.

    There are more available fields but there are not needed now for the
    testing purposes.
    """

    # pylint: disable=invalid-name
    agentVersion: str
    apps: typing.List[GetStateRspApp]


@dataclass()
class Status(abc.ABC):
    """
    Stub class for the general-purpose Status entry.
    """

    code: int
    message: str


@dataclass
class KeaResponse(abc.ABC):
    """
    Stub class for the KeaResponse message.
    """

    status: Status
    response: bytes


@dataclass()
class ForwardToKeaOverHTTPRsp(abc.ABC):
    """
    Stub class for the ForwardToKeaOverHTTPRsp message.
    """

    status: Status
    # TODO: We should refactor the protoc API file to use the proper naming
    # convention.
    # pylint: disable=invalid-name
    keaResponses: typing.List[KeaResponse]


class StorkAgentGRPCClient:
    """
    The GRPC client designed to communicate with the Stork agent.
    The client acts as a context manager.
    """

    def __init__(self, agent_host: str, agent_port: int):
        """Creates an GRPC client for the Stork agent. Accepts its host and
        port."""
        self._agent_host = agent_host
        self._agent_port = agent_port
        self._ssl_context = None

    @staticmethod
    def for_service(agent: core.wrappers.agent.Agent):
        """Extracts the control endpoint from the agent and creates a client."""
        host, port = agent.get_stork_control_endpoint()
        return StorkAgentGRPCClient(host, port)

    @staticmethod
    def _create_secure_context(cert_pem_path, key_pem_path) -> ssl.SSLContext:
        """
        Creates an SSL context object from provided certificate and key files.
        Inspired by https://github.com/vmagamedov/grpclib/blob/master/examples/mtls/client.py
        """
        ctx = ssl.SSLContext()
        ctx.load_cert_chain(cert_pem_path, key_pem_path)
        # The following ciphers was proposed by the above example. I don't know
        # if all of them are necessary.
        ctx.set_ciphers("ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20")
        ctx.set_alpn_protocols(["h2"])
        return ctx

    def fetch_certs_from(self, agent: core.wrappers.agent.Agent):
        """
        Fetches the TLS certificates from the container of the provided
        Stork agent. The agent must be registered in the Stork server before."""
        with tempfile.TemporaryDirectory() as cert_dir:
            certs = agent.download_cert_files(cert_dir)
            self._ssl_context = self._create_secure_context(
                cert_pem_path=certs["cert"], key_pem_path=certs["key"]
            )

    def _call(self, make_request: typing.Callable[[AgentStub], typing.Any]):
        """Call the agent API in the async context."""

        async def send_request():
            async with Channel(
                self._agent_host, self._agent_port, ssl=self._ssl_context
            ) as channel:
                rsp = await make_request(AgentStub(channel))
                return rsp

        return asyncio.run(send_request())

    def ping(self):
        """Sends a Ping request to the agent. No response is expected."""
        self._call(lambda stub: stub.Ping(PingReq()))

    def get_state(self) -> GetStateRsp:
        """Sends a GetState request to the agent and returns the response."""
        rsp: GetStateRsp
        rsp = self._call(lambda stub: stub.GetState(GetStateReq()))
        return rsp

    def forward_to_kea_over_http(
        self, access_point: GetStateRspAppAccessPoint, *cmds
    ) -> typing.List[typing.Any]:
        """
        Forwards the request to the Kea server over HTTP.
        The access_point is the object from the GetStateRsp message.
        Cmds are the Kea commands specified as the Python dictionaries. They
        will be serialized to JSONs.
        Returns deserialized Kea responses.
        """
        protocol = "https" if access_point.useSecureProtocol else "http"
        url = f"{protocol}://{access_point.address}:{access_point.port}"

        grpc_cmds = []
        for cmd in cmds:
            cmd_json = json.dumps(cmd)
            grpc_cmd = KeaRequest(request=cmd_json)
            grpc_cmds.append(grpc_cmd)

        req = ForwardToKeaOverHTTPReq(url=url, keaRequests=grpc_cmds)

        rsp: ForwardToKeaOverHTTPRsp
        rsp = self._call(lambda stub: stub.ForwardToKeaOverHTTP(req))
        if rsp.status.code != 0:
            raise StatusError(rsp.status.code, rsp.status.message)

        deserialized = []
        for r in rsp.keaResponses:
            if r.status.code != 0:
                raise StatusError(r.status.code, r.status.message)
            deserialized.append(json.loads(r.response))
        return deserialized
