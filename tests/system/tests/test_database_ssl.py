import pytest

from core.wrappers import Server
from core.fixtures import server_parametrize


@server_parametrize("server-db-ssl-require")
def test_server_enable_sslmode_require(server_service: Server):
    server_service.log_in_as_admin()


@pytest.mark.skip("Missing trusted SSL certificates")
@server_parametrize("server-db-ssl-verify-ca")
def test_server_enable_sslmode_verify_ca(server_service: Server):
    server_service.log_in_as_admin()
