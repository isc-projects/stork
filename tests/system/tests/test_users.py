from core.wrappers import Server, api


def test_users_management(server_service: Server):
    """Check if users can be fetched and added."""
    admin_user = server_service.log_in_as_admin()
    assert admin_user['login'] == 'admin'

    users = server_service.list_users()
    assert users['total'] == 1
    assert users['items'][0]['email'] == ""
    assert users['items'][0]['groups'] == [1]
    assert users['items'][0]['id'] == 1
    assert users['items'][0]['lastname'] == "admin"
    assert users['items'][0]['login'] == "admin"
    assert users['items'][0]['name'] == "admin"

    groups = server_service.list_groups()
    assert groups['total'] == 2
    assert len(groups['items']) == 2
    assert groups['items'][0]['name'] in ['super-admin', 'admin']
    assert groups['items'][1]['name'] in ['super-admin', 'admin']

    server_service.create_user(
        "user", "a@example.org", "John", "Smith", [], "password")
