from core.compose import DockerCompose


class Server:
    def __init__(self, compose: DockerCompose, service_name: str):
        self._compose = compose
        self._service_name = service_name


class Kea:
    def __init__(self, compose: DockerCompose, service_name: str):
        self._compose = compose
        self._service_name = service_name

