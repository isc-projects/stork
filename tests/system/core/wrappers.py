from core.compose import DockerCompose


class ComposeServiceWrapper:
    def __init__(self, compose: DockerCompose, service_name: str):
        self._compose = compose
        self._service_name = service_name

    def is_operational(self):
        return self._compose.is_operational(self._service_name)

class Server(ComposeServiceWrapper):
    def __init__(self, compose: DockerCompose, service_name: str):
        super().__init__(compose, service_name)


class Kea(ComposeServiceWrapper):
    def __init__(self, compose: DockerCompose, service_name: str,
            server_service: Server):
        super().__init__(compose, service_name)
        self._server_service = server_service

    @property
    def server(self):
        return self._server_service

