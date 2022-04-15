from typing import Union

import requests

from core.compose import DockerCompose
from core.wrappers.base import ComposeServiceWrapper
import core.wrappers.api_types as api


class UnexpectedStatusCodeException(Exception):
    def __init__(self, expected, actual) -> None:
        super().__init__("Unexpected HTTP status. Expected: %d, got: %d"
                         % (expected, actual))


class Server(ComposeServiceWrapper):
    def __init__(self, compose: DockerCompose, service_name: str):
        super().__init__(compose, service_name)
        self._session = requests.Session()
        self._address = self._compose.get_service_ip_address(
            self._service_name, "storknet"
        )
        self._port = 8080

    def _fetch_api(self, method: str, endpoint: str,
                   expected_status: int = None, **kwargs):
        url = "http://%s:%s/api/%s" % (self._address, self._port, endpoint)
        r = self._session.request(method, url, **kwargs)
        if expected_status is not None and expected_status != r.status_code:
            raise UnexpectedStatusCodeException(expected_status, r.status_code)
        return r

    @property
    def ip_address(self):
        return self._address

    def log_in(self, username: str, password: str) -> api.User:
        r = self._fetch_api("POST", "/sessions", expected_status=200,
                            json=dict(useremail=username, userpassword=password))
        return r.json()

    def log_in_as_admin(self):
        return self.log_in("admin", "admin")

    def list_users(self, limit=10, offset=0) -> api.UserList:
        r = self._fetch_api("GET", "/users", expected_status=200,
                            params=dict(start=offset, limit=limit))
        return r.json()

    def list_groups(self, limit=10, offset=0) -> api.GroupList:
        r = self._fetch_api("GET", "/groups", expected_status=200,
                            params=dict(start=offset, limit=limit))
        return r.json()

    def create_user(self, user_create: api.UserCreate):
        self._fetch_api("POST", "/users", json=user_create,
                        expected_status=200)
