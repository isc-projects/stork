from core.compose import DockerCompose
from core.wrappers.base import ComposeServiceWrapper
from core.wrappers.server import Server


class Kea(ComposeServiceWrapper):
    def __init__(self, compose: DockerCompose, service_name: str,
                 server_service: Server):
        super().__init__(compose, service_name)
        self._server_service = server_service

    @property
    def server(self):
        return self._server_service
