from core.wrappers import Register, Server


def test_register_without_server_token(
    server_service: Server, register_service: Register
):
    register_service.register(server_token=None)

    server_service.log_in_as_admin()
    machines = server_service.list_machines()
    assert machines.total == 1
    machine = machines.items[0]
    assert not machine.authorized


def test_register_with_server_token(server_service: Server, register_service: Register):
    server_service.log_in_as_admin()
    server_token = server_service.read_server_token()

    register_service.register(server_token)

    machines = server_service.list_machines()
    assert machines.total == 1
    machine = machines.items[0]
    assert machine.authorized
