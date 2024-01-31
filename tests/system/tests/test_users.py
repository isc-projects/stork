from openapi_client.api.users_api import User

from core.wrappers import Server


def test_users_management(server_service: Server):
    """Check if users can be fetched and added."""
    admin_user = server_service.log_in_as_admin()
    assert admin_user.login == "admin"

    users = server_service.list_users()
    assert users.total == 1
    assert "email" not in users.items[0]
    assert users.items[0].groups == [1]
    assert users.items[0].id == 1
    assert users.items[0].lastname == "admin"
    assert users.items[0].login == "admin"
    assert users.items[0].name == "admin"

    groups = server_service.list_groups()
    assert groups.total == 2
    assert len(groups.items) == 2
    assert groups.items[0].name in ["super-admin", "admin"]
    assert groups.items[1].name in ["super-admin", "admin"]

    server_service.create_user(
        User(
            id=0,
            login="user",
            email="a@example.org",
            name="John",
            lastname="Smith",
            groups=[],
            authentication_method_id="internal",
        ),
        "password",
    )


def test_user_without_groups(server_service: Server):
    """Users with no assigned groups must be able to log in and out."""
    server_service.log_in_as_admin()
    server_service.create_user(
        User(
            id=0,
            login="user",
            email="a@example.org",
            name="John",
            lastname="Smith",
            groups=[],
            authentication_method_id="internal",
        ),
        "password",
    )
    server_service.log_out()

    server_service.log_in("user", "password")
    server_service.log_out()
