from core.wrappers import Server
from core.fixtures import server_parametrize


@server_parametrize("server-db-auth-trust")
def test_server_database_auth_trust(server_service: Server):
    server_service.log_in_as_admin()


@server_parametrize("server-db-auth-md5")
def test_server_database_auth_md5(server_service: Server):
    server_service.log_in_as_admin()


@server_parametrize("server-db-auth-scram-sha-256")
def test_server_database_auth_scram_sha256(server_service: Server):
    server_service.log_in_as_admin()
