from core.compose import DockerCompose

class ComposeServiceWrapper:
    def __init__(self, compose: DockerCompose, service_name: str):
        self._compose = compose
        self._service_name = service_name

    def is_operational(self):
        return self._compose.is_operational(self._service_name)
