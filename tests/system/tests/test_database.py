import pytest
from openapi_client.exceptions import ServiceException

from core.wrappers import Postgres, Server
from core.fixtures import server_parametrize


def test_recovery_after_database_connection_failed(
    server_service: Server, postgres_service: Postgres
):
    """Test that the server is operational after temporary database connection failure."""
    server_service.log_in_as_admin()

    with postgres_service.unavailable():
        # The database is paused, so the server waits for it to become
        # available. The timeout should reach and raise an exception.
        pytest.raises(ServiceException, server_service.overview)
    server_service.overview()


def test_recovery_after_database_shutdown(
    server_service: Server, postgres_service: Postgres
):
    """Test that the server is operational after database shutdown and restart."""
    server_service.log_in_as_admin()

    with postgres_service.shutdown():
        # The database is shutdown. The server should immediately recognize
        # it is down and raise an exception.
        pytest.raises(ServiceException, server_service.overview)

    server_service.overview()


@server_parametrize("server")
def test_interrupt_during_database_shutdown(
    server_service: Server, postgres_service: Postgres
):
    """Test that the server is operational after database shutdown is interrupted."""
    with postgres_service.shutdown():
        # The server should gracefully shutdown even if the database is down.
        server_service.interrupt_stork_server()
